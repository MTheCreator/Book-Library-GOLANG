package Controllers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	inmemoryStores "finalProject/InmemoryStores"
	interfaces "finalProject/Interfaces"
	"finalProject/StructureData"
)

// JSON file path for author persistence
var authorFile = "authors.json"

// InitializeAuthorFile ensures the JSON file for authors exists
func InitializeAuthorFile() {
	if _, err := os.Stat(authorFile); os.IsNotExist(err) {
		file, _ := os.Create(authorFile)
		file.Write([]byte("[]"))
		file.Close()
	} else {
		// Load authors from the JSON file into the in-memory store
		file, err := os.Open(authorFile)
		if err != nil {
			panic("Failed to open author file")
		}
		defer file.Close()

		var authors []StructureData.Author
		if err := json.NewDecoder(file).Decode(&authors); err != nil {
			panic("Failed to decode author file")
		}

		// Populate the in-memory store
		store := inmemoryStores.GetAuthorStoreInstance()
		for _, author := range authors {
			store.CreateAuthor(author)
		}
	}
}

// GetAllAuthors handles the GET /authors request
func GetAllAuthors(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()

	// Retrieve all authors
	authors, _ := store.SearchAuthors(StructureData.AuthorSearchCriteria{}) // Empty criteria for all authors

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authors)
}

// GetAuthorByID handles the GET /authors/{id} request
func GetAuthorByID(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/authors/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid author ID"})
		return
	}

	// Retrieve the author by ID
	author, errResponse := store.GetAuthor(id)
	if errResponse != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResponse)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(author)
}

// CreateAuthor handles the POST /authors request
func CreateAuthor(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()

	// Decode the request body
	var author StructureData.Author
	if err := json.NewDecoder(r.Body).Decode(&author); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Create the author in the store
	createdAuthor, errResponse := store.CreateAuthor(author)
	if errResponse != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResponse)
		return
	}

	// Persist to JSON file
	if err := persistAuthorsToFile(store); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Return the created author
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdAuthor)
}

// UpdateAuthor handles the PUT /authors/{id} request
func UpdateAuthor(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/authors/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid author ID"})
		return
	}

	// Decode the request body
	var author StructureData.Author
	if err := json.NewDecoder(r.Body).Decode(&author); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Update the author in the store
	updatedAuthor, errResponse := store.UpdateAuthor(id, author)
	if errResponse != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResponse)
		return
	}

	// Persist to JSON file
	if err := persistAuthorsToFile(store); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Return the updated author
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAuthor)
}

// DeleteAuthor handles the DELETE /authors/{id} request
func DeleteAuthor(w http.ResponseWriter, r *http.Request) {
	authorStore := inmemoryStores.GetAuthorStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()
	orderStore := inmemoryStores.GetOrderStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/authors/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid author ID"})
		return
	}

	// Delete the author from the store
	errResponse := authorStore.DeleteAuthor(id)
	if errResponse != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResponse)
		return
	}

	// Delete all books associated with the author if not in an order
	books := bookStore.GetAllBooks()
	for _, book := range books {
		if book.Author.ID == id {
			// Check if the book exists in any order
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

			// If the book is not in any order, delete it
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

	// Persist updated books to file
	if err := persistBooksToFile(bookStore.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving books data"})
		return
	}

	// Persist updated authors to file
	if err := persistAuthorsToFile(authorStore); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusNoContent)
}

// SearchAuthors handles the POST /authors/search request
func SearchAuthors(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()

	// Decode the search criteria
	var criteria StructureData.AuthorSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
		return
	}

	// Search authors
	authors, errResponse := store.SearchAuthors(criteria)
	if errResponse != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResponse)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authors)
}

// persistAuthorsToFile saves all authors to the JSON file in a pretty JSON format
func persistAuthorsToFile(store interfaces.AuthorStore) *StructureData.ErrorResponse {
	authors, errResponse := store.SearchAuthors(StructureData.AuthorSearchCriteria{})
	if errResponse != nil {
		return &StructureData.ErrorResponse{Message: errResponse.Error()}
	}

	file, err := os.Create(authorFile)
	if err != nil {
		return &StructureData.ErrorResponse{Message: "Failed to create author file: " + err.Error()}
	}
	defer file.Close()

	// Use a pretty JSON encoder
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Add indentation for better readability

	if err := encoder.Encode(authors); err != nil {
		return &StructureData.ErrorResponse{Message: "Failed to encode authors to file: " + err.Error()}
	}

	return nil
}
