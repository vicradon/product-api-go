package main

import (
	"fmt"
	"net/http"
	"strconv"
    "strings"
)

type Product struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func getItems(w http.ResponseWriter, r *http.Request, id int) {
    if id != 0 {
     	fmt.Fprintf(w, "Got item, %d", id)
    } else {
       fmt.Fprintf(w, "Fetched all items")
    }
}

func createItem(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Created item")
}

func updateItem(w http.ResponseWriter, r *http.Request, id int) {
	fmt.Fprintf(w, "Updated item, %d", id)
}

func deleteItem(w http.ResponseWriter, r *http.Request, id int) {
	fmt.Fprintf(w, "Deleted item, %d", id)
}

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
        fmt.Fprintf(w, "API docs") 
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
			createItem(w, r)
		case http.MethodGet:
			getItems(w, r, id)
		case http.MethodPut:
			updateItem(w, r, id)
		case http.MethodDelete:
			deleteItem(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Server running on port 8000")
	http.ListenAndServe(":8000", nil)
}
