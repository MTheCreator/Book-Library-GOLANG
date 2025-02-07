package Controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	inmemoryStores "finalProject/InmemoryStores"
	"finalProject/StructureData"
)

// JSON file path for book persistence
var bookFile = "books.json"

func InitializeBookFile() {
	if _, err := os.Stat(bookFile); os.IsNotExist(err) {
		// If the file doesn't exist, create an empty one
		file, _ := os.Create(bookFile)
		file.Write([]byte("[]"))
		file.Close()
	} else {
		// Load books from the JSON file into the in-memory store
		file, err := os.Open(bookFile)
		if err != nil {
			panic("Failed to open book file")
		}
		defer file.Close()

		var books []StructureData.Book
		if err := json.NewDecoder(file).Decode(&books); err != nil {
			panic("Failed to decode book file")
		}

		// Populate the in-memory store directly without validation
		store := inmemoryStores.GetBookStoreInstance()
		for _, book := range books {
			store.AddBookDirectly(book) // Bypassing validation
			log.Printf("Book with ID %d loaded into store (Stock: %d)", book.ID, book.Stock)
		}
	}
}

// GetAllBooks handles the GET /books request
func GetAllBooks(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()

	// Retrieve all books
	books := store.GetAllBooks()

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

// GetBookByID handles the GET /books/{id} request
func GetBookByID(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/books/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book ID"})
		return
	}

	// Retrieve the book by ID
	book, errResp := store.GetBook(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// CreateBook handles the POST /books request
func CreateBook(w http.ResponseWriter, r *http.Request) {
	bookStore := inmemoryStores.GetBookStoreInstance()
	authorStore := inmemoryStores.GetAuthorStoreInstance()

	// Decode the request body
	var book StructureData.Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate stock
	if book.Stock < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Stock must be at least 1"})
		return
	}

	// Check if the author exists
	authors := authorStore.GetAllAuthors()
	authorExists := false
	for _, existingAuthor := range authors {
		if existingAuthor.FirstName == book.Author.FirstName &&
			existingAuthor.LastName == book.Author.LastName &&
			existingAuthor.Bio == book.Author.Bio {
			book.Author = existingAuthor // Link existing author
			authorExists = true
			break
		}
	}

	// If author doesn't exist, create the author
	if !authorExists {
		createdAuthor, errResp := authorStore.CreateAuthor(book.Author)
		if errResp != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errResp)
			return
		}
		book.Author = createdAuthor

		// Persist the new author to the JSON file
		if err := persistAuthorsToFile(authorStore); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving author data"})
			return
		}
	}

	// Create the book in the store
	createdBook, errResp := bookStore.CreateBook(book)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist to JSON file
	if err := persistBooksToFile(bookStore.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving book data"})
		return
	}

	// Return the created book
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdBook)
}

// UpdateBook handles the PUT /books/{id} request
func UpdateBook(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/books/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book ID"})
		return
	}

	// Decode the request body
	var book StructureData.Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate stock
	if book.Stock < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Stock must be at least 1"})
		return
	}

	// Update the book in the store
	updatedBook, errResp := store.UpdateBook(id, book)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist to JSON file
	if err := persistBooksToFile(store.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Return the updated book
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedBook)
}

// DeleteBook handles the DELETE /books/{id} request
func DeleteBook(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()
	orderStore := inmemoryStores.GetOrderStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/books/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book ID"})
		return
	}

	// Check if the book is linked to any orders
	orders := orderStore.GetAllOrders()
	for _, order := range orders {
		for _, item := range order.Items {
			if item.Book.ID == id {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Book cannot be deleted as it is linked to existing orders"})
				return
			}
		}
	}

	// Delete the book from the store
	errResp := store.DeleteBook(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist to JSON file
	if err := persistBooksToFile(store.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusNoContent)
}

// SearchBooks handles the POST /books/search request
func SearchBooks(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()

	// Decode the search criteria from the request body
	var criteria StructureData.BookSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
		return
	}

	// Perform the search
	searchResults, errResp := store.SearchBooks(criteria)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Return the search results as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResults)
}

// persistBooksToFile saves all books to the JSON file in a pretty JSON format
func persistBooksToFile(books []StructureData.Book) error {
	file, err := os.Create(bookFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use a pretty JSON encoder
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Add indentation for better readability

	if err := encoder.Encode(books); err != nil {
		return err
	}

	return nil
}
