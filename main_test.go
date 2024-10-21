package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	_, err := db.Exec("INSERT INTO products (name) VALUES (?)", "Test Product")
	if err != nil {
		t.Fatalf("Failed to insert test product to test db %v", err)
	}

	req, err := http.NewRequest("POST", "/products", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getProducts(w, r, 1, db)
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
}

func TestListProducts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO products (name) VALUES (?), (?), (?)", "Tissue Paper", "Ribbons", "Band Aid")
	if err != nil {
		t.Fatalf("Failed to insert test products to db: %v", err)
	}

	requestrecorder := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getProducts(w, r, 0, db)
	})

	request, err := http.NewRequest("GET", "/products", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(requestrecorder, request)

	if status := requestrecorder.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v expected %v", status, http.StatusOK)
	}

	var products []Product
	err = json.NewDecoder(requestrecorder.Body).Decode(&products)
	if err != nil {
		t.Errorf("Could not decode JSON body: %v", err)
	}
}

func TestGetSingleProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO products (name) VALUES (?), (?), (?)", "Toothpaste", "Toothbrush", "Soap")
	if err != nil {
		t.Fatalf("Failed to insert into products: %v", err)
	}

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getProducts(w, r, 1, db)
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
}

func TestUpdateProduct(t *testing.T) {

}

func TestDeleteProduct(t *testing.T) {

}
