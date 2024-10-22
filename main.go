// @title			Product API
// @version		1.0
// @description	This is a sample API for managing products
// @host			{host}
// @BasePath		/
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"github.com/go-playground/validator/v10"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/mattn/go-sqlite3"

	"github.com/vicradon/internpulse/stage3/docs"
)

// Product represents the product model
//
//	@Description	Product model
type Product struct {
	Id   int    `json:"id"`   //	@Description	The unique ID of the product
	Name string `json:"name"` //	@Description	The name of the product
}

// writeJSON writes the response in JSON format
//
//	@Summary	Write JSON response
//	@Param		data	body		Product	true	"Product data"
//	@Success	200		{object}	Product
//	@Failure	500		{object}	string
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to write JSON", http.StatusInternalServerError)
	}
}

// getProduct retrieves a specific product
//
//	@Summary		Get product
//	@Description	Get a specific product by ID
//	@Param			id	path		int	false	"Product ID"
//	@Success		200	{array}		Product
//	@Failure		400	{object}	string
//	@Failure		500	{object}	string
//	@Router			/products/{id} [get]
func getProduct(w http.ResponseWriter, _ *http.Request, id int, db *sql.DB) {
	var product Product

	err := db.QueryRow("SELECT * FROM products WHERE id=?", id).Scan(&product.Id, &product.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("No such product with id %d", id), http.StatusBadRequest)
			return
		}
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}

	writeJSON(w, product)

}

// getProducts retrieves all products
//
//	@Summary		Get products
//	@Description	Get all products
//	@Success		200	{array}		Product
//	@Failure		400	{object}	string
//	@Failure		500	{object}	string
//	@Router			/products [get]
func getProducts(w http.ResponseWriter, _ *http.Request, db *sql.DB) {
	rows, err := db.Query("SELECT * FROM products")
	if err != nil {
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
			http.Error(w, "An error occurred in retrieving the rows", http.StatusInternalServerError)
		}
	}

	writeJSON(w, products)
}

// createProduct creates a new product
//
//	@Summary		Create a new product
//	@Description	Create a product with the provided data
//	@Accept			json
//	@Produce		json
//	@Param			product	body		Product	true	"Product data"
//	@Success		201		{object}	Product
//	@Failure		400		{object}	string
//	@Failure		500		{object}	string
//	@Router			/products [post]
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
		http.Error(w, "Unable to write to database", http.StatusInternalServerError)
	}

	newProductId, _ := result.LastInsertId()

	var newProduct Product

	if err = db.QueryRow("SELECT * FROM products WHERE id = ?", newProductId).Scan(&newProduct.Id, &newProduct.Name); err != nil {
		http.Error(w, "An error occurred while fetching the newly created row", http.StatusInternalServerError)
		return
	}

	writeJSON(w, newProduct)
}

// updateProduct updates an existing product
//
//	@Summary		Update a product
//	@Description	Update a product by ID
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int		true	"Product ID"
//	@Param			product	body		Product	true	"Product data"
//	@Success		200		{object}	Product
//	@Failure		400		{object}	string
//	@Failure		500		{object}	string
//	@Router			/products/{id} [put]
func updateProduct(w http.ResponseWriter, r *http.Request, id int, db *sql.DB) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	var newProduct Product

	if err = json.Unmarshal(body, &newProduct); err != nil {
		http.Error(w, "Error parsing request body as JSON", http.StatusBadRequest)
		return
	}

	result, err := db.Exec("UPDATE products SET name = ? WHERE id = ?", newProduct.Name, id)

	if err != nil {
		http.Error(w, "An error occurred while updating the rows", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("An error occurred for product with id %d", id), http.StatusBadRequest)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, fmt.Sprintf("No such product with id %d", id), http.StatusBadRequest)
		return
	}

	if err = db.QueryRow("SELECT * FROM products WHERE id=?", id).Scan(&newProduct.Id, &newProduct.Name); err != nil {
		http.Error(w, "An error occurred while writing the rows", http.StatusInternalServerError)
	}

	writeJSON(w, newProduct)
}

// deleteProduct deletes a product
//
//	@Summary		Delete a product
//	@Description	Delete a product by ID
//	@Param			id	path		int		true	"Product ID"
//	@Success		200	{string}	string	"Deleted successfully"
//	@Failure		400	{object}	string
//	@Failure		500	{object}	string
//	@Router			/products/{id} [delete]
func deleteProduct(w http.ResponseWriter, _ *http.Request, id int, db *sql.DB) {
	result, err := db.Exec("DELETE from products WHERE id = ?", id)
	if err != nil {
		http.Error(w, "An error occurred while deleting your data", http.StatusInternalServerError)
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

	writeJSON(w, "Deleted product successfully")
}

// initDB initializes the database
func initDB(db *sql.DB) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS products(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	databaseFile := os.Getenv("DATABASE_FILE")
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")

	if host == "" {
		host = fmt.Sprintf("localhost:%s", port)
	}

	docs.SwaggerInfo.Host = host

	db, err := sql.Open("sqlite3", databaseFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB(db)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host = r.Host
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})

	http.Handle("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	http.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createProduct(w, r, db)
		case http.MethodGet:
			getProducts(w, r, db)
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
			if id > 0 {
				getProduct(w, r, id, db)
			} else {
				getProducts(w, r, db)
			}
		case http.MethodPut:
			updateProduct(w, r, id, db)
		case http.MethodDelete:
			deleteProduct(w, r, id, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Printf("Server running on port %s", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
