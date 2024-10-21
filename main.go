package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	_ "github.com/mattn/go-sqlite3"
)

type Product struct {
	Id   int    `json:"id"`
	Name string `json:"name" validate:"required"`
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to write JSON", http.StatusInternalServerError)
	}
}

func getProducts(w http.ResponseWriter, _ *http.Request, id int, db *sql.DB) {
	if id != 0 {
		var product Product

		err := db.QueryRow("SELECT * FROM products WHERE id=?", id).Scan(&product.Id, &product.Name)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, fmt.Sprintf("No such product with id %d", id), http.StatusBadRequest)
				return
			}
			http.Error(w, "An error occured", http.StatusInternalServerError)
			return
		}

		writeJSON(w, product)
	} else {
		rows, err := db.Query("SELECT * FROM products")
		if err != nil {
			log.Fatal(err)
			http.Error(w, "Unable to read from database", http.StatusInternalServerError)
		}
		defer rows.Close()

		var products []Product

		for rows.Next() {
			var product Product
			if err := rows.Scan(&product.Id, &product.Name); err != nil {
				http.Error(w, "Bad reading of database content", http.StatusInternalServerError)
			}
			products = append(products, product)

			if err = rows.Err(); err != nil {
				http.Error(w, "An error occured in retrieving the rows", http.StatusInternalServerError)
			}
		}
		writeJSON(w, products)
	}
}

func createProduct(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	var product Product

	if err = json.Unmarshal(body, &product); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	validate := validator.New()

	if err := validate.Struct(product); err != nil {
		var errMessages []string
		for _, err := range err.(validator.ValidationErrors) {
			errMessages = append(errMessages, fmt.Sprintf("Field '%s': %s", err.Field(), err.Tag()))
		}
		http.Error(w, fmt.Sprintf("Validation errors: %s", errMessages), http.StatusBadRequest)
		return
	}

	result, err := db.Exec("INSERT INTO products (name) VALUES (?)", product.Name)
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Unable to write to datbase", http.StatusInternalServerError)
	}

	newProductId, _ := result.LastInsertId()

	var newProduct Product

	if err = db.QueryRow("SELECT * FROM products WHERE id = ?", newProductId).Scan(&newProduct.Id, &newProduct.Name); err != nil {
		http.Error(w, "An error occured while fetching the newly created row", http.StatusInternalServerError)
		return
	}

	writeJSON(w, newProduct)
}

func updateProducts(w http.ResponseWriter, r *http.Request, id int, db *sql.DB) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	var newProduct Product

	if err = json.Unmarshal(body, &newProduct); err != nil {
		http.Error(w, "Error parsing request body as json", http.StatusBadRequest)
		return
	}

	fmt.Println(newProduct.Name, id)
	result, err := db.Exec("UPDATE products SET name = ? WHERE id = ?", newProduct.Name, id)

	if err != nil {
		http.Error(w, "An error occured while updating the rows", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("An error occured for product with id %d", id), http.StatusBadRequest)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, fmt.Sprintf("No such product with id %d", id), http.StatusBadRequest)
		return
	}

	if err = db.QueryRow("SELECT * FROM products WHERE id=?", id).Scan(&newProduct.Id, &newProduct.Name); err != nil {
		http.Error(w, "An error occured while writing the rows", http.StatusInternalServerError)
	}

	writeJSON(w, newProduct)
}

func deleteProduct(w http.ResponseWriter, _ *http.Request, id int, db *sql.DB) {
	result, err := db.Exec("DELETE from products WHERE id = ?", id)
	if err != nil {
		http.Error(w, "An error occured while deleting your data", http.StatusInternalServerError)
		return
	}
	rowsAffected, err := result.RowsAffected()

	if err != nil {
		log.Fatal(err)
	}

	if rowsAffected == 0 {
		http.Error(w, fmt.Sprintf("No such product with id, %d", id), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Deleted product successfully")
}

func initDB(db *sql.DB) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS products(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	db, err := sql.Open("sqlite3", "database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB(db)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "API docs")
	})

	http.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createProduct(w, r, db)
		default:
			http.Redirect(w, r, "/products/", http.StatusMovedPermanently)
		}
	})

	http.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/products")
		path = strings.Trim(path, "/")

		var id int
		var err error

		if path != "" {
			id, err = strconv.Atoi(path)
			if err != nil {
				http.Error(w, "Invalid item Id", http.StatusBadRequest)
				return
			}
		}

		switch r.Method {
		case http.MethodPost:
			createProduct(w, r, db)
		case http.MethodGet:
			getProducts(w, r, id, db)
		case http.MethodPut:
			updateProducts(w, r, id, db)
		case http.MethodDelete:
			deleteProduct(w, r, id, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Server running on port 8000")
	http.ListenAndServe(":8000", nil)
}
