// @title			Product API
// @version		1.0
// @description	This is a sample API for managing products
// @host			{host}
// @BasePath		/
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/vicradon/internpulse/stage3/docs"
)

// Product represents the product model
type Product struct {
	Id   int    `json:"id"`   //	@Description	The unique ID of the product
	Name string `json:"name"` //	@Description	The name of the product
}

var validate = validator.New()

// initDB initializes the database
func initDB(db *sql.DB) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS products(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE)")
	if err != nil {
		log.Fatal(err)
	}
}

// @Summary     Get a product
// @Description Get a product by its ID
// @Tags        products
// @Produce     json
// @Param       id path int true "Product ID"
// @Success     200 {object} Product
// @Router      /products/{id} [get]
func getProduct(c *gin.Context, db *sql.DB) {
	id, _ := strconv.Atoi(c.Param("id"))
	var product Product

	err := db.QueryRow("SELECT * FROM products WHERE id=?", id).Scan(&product.Id, &product.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No such product with id %d", id)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
		return
	}

	c.JSON(http.StatusOK, product)
}

// @Summary     List all products or get a product by name
// @Description Get all products or retrieve a specific product by name. Name query parameter is optional.
// @Tags        products
// @Produce     json
// @Param       name  query      string false "Name of the product to retrieve"
// @Success     200 {array}  Product
// @Router      /products [get]
func getProducts(c *gin.Context, db *sql.DB) {
	productName := c.Query("name")

	var rows *sql.Rows
	var err error

	if productName != "" {
		rows, err = db.Query("SELECT * FROM products WHERE name=?", productName)
	} else {
		rows, err = db.Query("SELECT * FROM products")
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "Unable to read from database"})
		return
	}
	defer rows.Close()

	var products []Product

	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.Id, &product.Name); err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": "Bad reading of database content"})
			return
		}
		products = append(products, product)
	}

	if productName != "" && len(products) == 0 {
		c.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("No such product with name '%s'. Product names must be exact", productName)})
		return
	}

	if productName != "" {
		c.JSON(http.StatusOK, products[0])
	} else {
		c.JSON(http.StatusOK, products)
	}
}

// @Summary     Create a new product
// @Description Add a new product to the database
// @Tags        products
// @Accept      json
// @Produce     json
// @Param       product body Product true "Product object"
// @Success     201 {object} Product
// @Router      /products [post]
func createProduct(c *gin.Context, db *sql.DB) {
	var product Product

	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if err := validate.Struct(product); err != nil {
		var errMessages []string
		for _, err := range err.(validator.ValidationErrors) {
			errMessages = append(errMessages, fmt.Sprintf("Field '%s': %s", err.Field(), err.Tag()))
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Validation errors: %s", errMessages)})
		return
	}

	result, err := db.Exec("INSERT INTO products (name) VALUES (?)", product.Name)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			c.JSON(http.StatusConflict, gin.H{"error": "Product name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to write to database"})
		return
	}

	newProductId, _ := result.LastInsertId()

	if err = db.QueryRow("SELECT * FROM products WHERE id = ?", newProductId).Scan(&product.Id, &product.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while fetching the newly created row"})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// @Summary     Update a product
// @Description Update a product's information
// @Tags        products
// @Accept      json
// @Produce     json
// @Param       id path int true "Product ID"
// @Param       product body Product true "Updated product object"
// @Success     200 {object} Product
// @Router      /products/{id} [put]
func updateProduct(c *gin.Context, db *sql.DB) {
	id, _ := strconv.Atoi(c.Param("id"))
	var newProduct Product

	if err := c.ShouldBindJSON(&newProduct); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing request body as JSON"})
		return
	}

	result, err := db.Exec("UPDATE products SET name = ? WHERE id = ?", newProduct.Name, id)

	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			c.JSON(http.StatusConflict, gin.H{"error": "A product with this name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while updating the rows"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No such product with id %d", id)})
		return
	}

	if err = db.QueryRow("SELECT * FROM products WHERE id=?", id).Scan(&newProduct.Id, &newProduct.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while returning the updated data"})
		return
	}

	c.JSON(http.StatusOK, newProduct)
}

// @Summary     Update a product by name
// @Description Update a product's information by name
// @Tags        products
// @Accept      json
// @Produce     json
// @Param       name  query      string true "Name of the product to update"
// @Param       product body Product true "Updated product object"
// @Success     200 {object} Product
// @Router      /products [put]
func updateProductByName(c *gin.Context, db *sql.DB) {
	productName := c.Query("name")
	var newProduct Product

	if err := c.ShouldBindJSON(&newProduct); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing the request body as JSON"})
		return
	}

	result, err := db.Exec("UPDATE products SET name = ? WHERE name = ?", newProduct.Name, productName)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			c.JSON(http.StatusConflict, gin.H{"error": "A product with this name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occured while updating the product"})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No such product with name '%s'. Product names must be exact", productName)})
		return
	}

	if err = db.QueryRow("SELECT * FROM products WHERE name=?", newProduct.Name).Scan(&newProduct.Id, &newProduct.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while returning the updated data"})
		return
	}

	c.JSON(http.StatusOK, newProduct)
}

// @Summary     Delete a product
// @Description Delete a product by its ID
// @Tags        products
// @Param       id path int true "Product ID"
// @Success     200 {object} map[string]string
// @Router      /products/{id} [delete]
func deleteProduct(c *gin.Context, db *sql.DB) {
	id, _ := strconv.Atoi(c.Param("id"))
	result, err := db.Exec("DELETE from products WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while deleting your data"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No such product with id, %d", id)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted product successfully"})
}

// @Summary     Delete a product by name
// @Description Remove a product's information by name
// @Tags        products
// @Accept      json
// @Produce     json
// @Param       name  query      string true "Name of the product to delete"
// @Success     204 {object} nil
// @Router      /products [delete]
func deleteProductByName(c *gin.Context, db *sql.DB) {
	productName := c.Query("name")

	result, err := db.Exec("DELETE FROM products WHERE name=?", productName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "An error occurred while deleting the product"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "An error occurred while retrieving deletion status"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("No such product with name '%s'. Product names must be exact", productName)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted product successfully"})
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

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/products", func(c *gin.Context) {
		getProducts(c, db)
	})

	r.POST("/products", func(c *gin.Context) {
		createProduct(c, db)
	})

	r.PUT("/products", func(c *gin.Context) {
		updateProductByName(c, db)
	})

	r.GET("/products/:id", func(c *gin.Context) {
		getProduct(c, db)
	})
	r.PUT("/products/:id", func(c *gin.Context) {
		updateProduct(c, db)
	})
	r.DELETE("/products/:id", func(c *gin.Context) {
		deleteProduct(c, db)
	})
	r.DELETE("/products", func(c *gin.Context) {
		deleteProductByName(c, db)
	})

	fmt.Printf("Server running on port %s\n", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatal(err)
	}
}
