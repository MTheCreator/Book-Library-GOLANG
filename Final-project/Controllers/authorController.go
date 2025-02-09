// File: Controllers/author.go
package Controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	inmemoryStores "finalProject/InmemoryStores"
	postgresStores "finalProject/postgresStores"
	interfaces "finalProject/Interfaces"
	"finalProject/StructureData"
)

// JSON file path for author persistence.
var authorFile = "authors.json"

// InitializeAuthorFile ensures the JSON file for authors exists and loads data into both the in-memory and PostgreSQL stores.
func InitializeAuthorFile() {
	pgStore := postgresStores.GetPostgresAuthorStoreInstance()
    existingAuthors := pgStore.GetAllAuthors()
    if len(existingAuthors) > 0 {
        log.Println("Authors already exist in PostgreSQL; skipping JSON initialization for authors.")
        return
    }
	if _, err := os.Stat(authorFile); os.IsNotExist(err) {
		file, _ := os.Create(authorFile)
		file.Write([]byte("[]"))
		file.Close()
	} else {
		file, err := os.Open(authorFile)
		if err != nil {
			panic("Failed to open author file")
		}
		defer file.Close()

		var authors []StructureData.Author
		if err := json.NewDecoder(file).Decode(&authors); err != nil {
			panic("Failed to decode author file")
		}

		store := inmemoryStores.GetAuthorStoreInstance()
		pgStore := postgresStores.GetPostgresAuthorStoreInstance()
		for _, author := range authors {
			// Create in PostgreSQL first.
			createdPgAuthor, errResp := pgStore.CreateAuthor(author)
			if errResp != nil {
				log.Printf("Error creating author in PostgreSQL for ID %d: %v", author.ID, errResp.Message)
				continue
			}
			_, _ = store.CreateAuthor(createdPgAuthor)
		}
	}
}

// GetAllAuthors handles the GET /authors request.
func GetAllAuthors(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()
	authors, _ := store.SearchAuthors(StructureData.AuthorSearchCriteria{})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authors)
}

// GetAuthorByID handles the GET /authors/{id} request.
func GetAuthorByID(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()
	idStr := r.URL.Path[len("/authors/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid author ID"})
		return
	}
	author, errResp := store.GetAuthor(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(author)
}

// CreateAuthor handles the POST /authors request.
func CreateAuthor(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()
	pgStore := postgresStores.GetPostgresAuthorStoreInstance()

	var author StructureData.Author
	if err := json.NewDecoder(r.Body).Decode(&author); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Create in PostgreSQL first.
	createdPgAuthor, pgErr := pgStore.CreateAuthor(author)
	if pgErr != nil {
		log.Printf("Error creating author in PostgreSQL: %v", pgErr.Message)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving author"})
		return
	}

	// Now create in the in-memory store using the PostgreSQL author.
	createdAuthor, errResp := store.CreateAuthor(createdPgAuthor)
	if errResp != nil {
		pgStore.DeleteAuthor(createdPgAuthor.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if err := persistAuthorsToFile(store); err != nil {
		store.DeleteAuthor(createdAuthor.ID)
		pgStore.DeleteAuthor(createdPgAuthor.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdAuthor)
}

// (UpdateAuthor, DeleteAuthor, and SearchAuthors remain unchanged.)

// persistAuthorsToFile saves all authors to the JSON file in a pretty JSON format.
func persistAuthorsToFile(store interfaces.AuthorStore) *StructureData.ErrorResponse {
	authors, errResp := store.SearchAuthors(StructureData.AuthorSearchCriteria{})
	if errResp != nil {
		return &StructureData.ErrorResponse{Message: errResp.Message}
	}

	file, err := os.Create(authorFile)
	if err != nil {
		return &StructureData.ErrorResponse{Message: "Failed to create author file: " + err.Error()}
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(authors); err != nil {
		return &StructureData.ErrorResponse{Message: "Failed to encode authors to file: " + err.Error()}
	}

	return nil
}


// UpdateAuthor handles the PUT /authors/{id} request.
func UpdateAuthor(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()
	pgStore := postgresStores.GetPostgresAuthorStoreInstance()

	// Extract the author ID from the URL.
	idStr := r.URL.Path[len("/authors/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid author ID"})
		return
	}

	// Decode the request body.
	var author StructureData.Author
	if err := json.NewDecoder(r.Body).Decode(&author); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	updatedAuthor, errResp := store.UpdateAuthor(id, author)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist the updated authors to JSON.
	if err := persistAuthorsToFile(store); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Persist the update to PostgreSQL.
	_, pgErr := pgStore.UpdateAuthor(id, updatedAuthor)
	if pgErr != nil {
		log.Printf("Error updating author in PostgreSQL: %v", pgErr.Message)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAuthor)
}

// DeleteAuthor handles the DELETE /authors/{id} request.
func DeleteAuthor(w http.ResponseWriter, r *http.Request) {
	authorStore := inmemoryStores.GetAuthorStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()
	pgStore := postgresStores.GetPostgresAuthorStoreInstance()
	orderStore := inmemoryStores.GetOrderStoreInstance()

	// Extract the author ID from the URL.
	idStr := r.URL.Path[len("/authors/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid author ID"})
		return
	}

	// Delete the author from the in-memory store.
	errResp := authorStore.DeleteAuthor(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Delete all books associated with the author if they are not linked to any orders.
	books := bookStore.GetAllBooks()
	for _, book := range books {
		if book.Author.ID == id {
			orders := orderStore.GetAllOrders()
			bookInOrder := false
			for _, order := range orders {
				for _, item := range order.Items {
					if item.Book.ID == book.ID {
						bookInOrder = true
						break
					}
				}
				if bookInOrder {
					break
				}
			}
			if !bookInOrder {
				errResp := bookStore.DeleteBook(book.ID)
				if errResp != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(errResp)
					return
				}
			}
		}
	}

	// Persist updated books.
	if err := persistBooksToFile(bookStore.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving books data"})
		return
	}

	// Persist updated authors.
	if err := persistAuthorsToFile(authorStore); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving authors data"})
		return
	}

	// Delete the author from PostgreSQL.
	pgErr := pgStore.DeleteAuthor(id)
	if pgErr != nil {
		log.Printf("Error deleting author from PostgreSQL: %v", pgErr.Message)
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchAuthors handles the POST /authors/search request.
func SearchAuthors(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()
	var criteria StructureData.AuthorSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
		return
	}
	authors, errResp := store.SearchAuthors(criteria)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authors)
}

