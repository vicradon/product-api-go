package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	req, err := http.NewRequest("POST", "/products", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		createProduct(w, r, db)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v expected %v", status, http.StatusOK)
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

	responserecorder := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getProducts(w, r, 0, db)
	})

	request, err := http.NewRequest("GET", "/products", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(responserecorder, request)

	if status := responserecorder.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v expected %v", status, http.StatusOK)
	}

	var products []Product
	err = json.NewDecoder(responserecorder.Body).Decode(&products)
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

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getProducts(w, r, 3, db)
	})

	req, err := http.NewRequest("GET", "/products/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

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

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		updateProduct(w, r, 2, db)
	})

	reqBody := fmt.Sprintf(`{"name":"%s"}`, product1NewName)

	req, err := http.NewRequest("PUT", "/products/2", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	status := rr.Code
	if status != http.StatusOK {
		t.Errorf("Hanlder returned wrong status code: got %v expected %v", status, http.StatusOK)
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

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteProduct(w, r, 2, db)
	})

	req, err := http.NewRequest("DELETE", "/products/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	status := rr.Code
	if status != http.StatusOK {
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
