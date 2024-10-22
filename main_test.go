package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db %v", err)
	}
	initDB(db)
	return db
}

func TestCreateProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productName := "Test Product"
	reqBody := fmt.Sprintf(`{"name":"%s"}`, productName)

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/products", func(c *gin.Context) {
		createProduct(c, db) // Only pass context and db
	})

	req, err := http.NewRequest("POST", "/products", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v expected %v", status, http.StatusCreated)
	}

	var product Product
	err = json.NewDecoder(rr.Body).Decode(&product)
	if err != nil {
		t.Errorf("Could not decode JSON body: %v", err)
	}

	if product.Name != productName {
		t.Fatalf("Expected product name to be %v but got %v", productName, product.Name)
	}
}

func TestListProducts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO products (name) VALUES (?), (?), (?)", "Tissue Paper", "Ribbons", "Band Aid")
	if err != nil {
		t.Fatalf("Failed to insert test products to db: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/products", func(c *gin.Context) {
		getProducts(c, db) // Only pass context and db
	})

	req, err := http.NewRequest("GET", "/products", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v expected %v", status, http.StatusOK)
	}

	var products []Product
	err = json.NewDecoder(rr.Body).Decode(&products)
	if err != nil {
		t.Errorf("Could not decode JSON body: %v", err)
	}

	if len(products) != 3 {
		t.Fatalf("Expected 3 products in db, instead got %v", len(products))
	}
}

func TestGetSingleProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	product2Name := "Soap"
	_, err := db.Exec("INSERT INTO products (name) VALUES (?), (?), (?)", "Toothpaste", "Toothbrush", product2Name)
	if err != nil {
		t.Fatalf("Failed to insert into products: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/products/:id", func(c *gin.Context) {
		getProduct(c, db) // Only pass context and db; function should handle ID extraction
	})

	req, err := http.NewRequest("GET", "/products/3", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v expected %v", status, http.StatusOK)
	}

	var product Product
	err = json.NewDecoder(rr.Body).Decode(&product)
	if err != nil {
		t.Errorf("Could not decode JSON body: %v", err)
	}

	if product.Name != product2Name {
		t.Fatalf("expected %v but got %v", product2Name, product.Name)
	}
}

func TestUpdateProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	product1Name := "Sugar"
	product1NewName := "Honey"

	_, err := db.Exec("INSERT INTO products (name) VALUES (?), (?), (?)", "Brownies", product1Name, "Flour")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.PUT("/products/:id", func(c *gin.Context) {
		updateProduct(c, db) // Only pass context and db
	})

	reqBody := fmt.Sprintf(`{"name":"%s"}`, product1NewName)
	req, err := http.NewRequest("PUT", "/products/2", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v expected %v", status, http.StatusOK)
	}

	var product1 Product
	err = json.NewDecoder(rr.Body).Decode(&product1)
	if err != nil {
		t.Errorf("Failed to decode JSON body: %v", err)
	}

	if product1.Name != product1NewName {
		t.Errorf("expected %v but got %v", product1Name, product1.Name)
	}
}

func TestDeleteProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if _, err := db.Exec("INSERT INTO products (name) VALUES (?), (?), (?)", "Yams", "Eggs", "Berries"); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.DELETE("/products/:id", func(c *gin.Context) {
		deleteProduct(c, db) // Only pass context and db
	})

	req, err := http.NewRequest("DELETE", "/products/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v but expected %v", status, http.StatusOK)
	}

	var rowCount int
	err = db.QueryRow("SELECT COUNT(*) FROM products").Scan(&rowCount)
	if err != nil {
		t.Log(err)
	}

	expectedRows := 2
	if expectedRows != rowCount {
		t.Errorf("expected %v number of rows remaining, but got %v", expectedRows, rowCount)
	}
}
