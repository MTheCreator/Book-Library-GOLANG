// File: Controllers/book.go
package Controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	inmemoryStores "finalProject/InmemoryStores"
	postgresStores "finalProject/postgresStores"
	"finalProject/StructureData"
)

// JSON file path for book persistence.
var bookFile = "books.json"

// InitializeBookFile ensures the JSON file for books exists and loads data into both the in-memory and PostgreSQL stores.
func InitializeBookFile() {
	pgStore := postgresStores.GetPostgresBookStoreInstance()
    existingBooks := pgStore.GetAllBooks()
    if len(existingBooks) > 0 {
        log.Println("Books already exist in PostgreSQL; skipping JSON initialization for Books.")
        return
    }
	if _, err := os.Stat(bookFile); os.IsNotExist(err) {
		file, _ := os.Create(bookFile)
		file.Write([]byte("[]"))
		file.Close()
	} else {
		file, err := os.Open(bookFile)
		if err != nil {
			panic("Failed to open book file")
		}
		defer file.Close()

		var books []StructureData.Book
		if err := json.NewDecoder(file).Decode(&books); err != nil {
			panic("Failed to decode book file")
		}

		store := inmemoryStores.GetBookStoreInstance()
		pgStore := postgresStores.GetPostgresBookStoreInstance()
		for _, book := range books {
			store.CreateBook(book) // This uses AddBookDirectly below or CreateBook with explicit ID.
			log.Printf("Book with ID %d loaded into store (Stock: %d)", book.ID, book.Stock)
			_, errResp := pgStore.CreateBook(book)
			if errResp != nil {
				log.Printf("Error persisting book ID %d to PostgreSQL: %v", book.ID, errResp.Message)
			}
		}
	}
}

// GetAllBooks handles the GET /books request.
func GetAllBooks(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()
	books := store.GetAllBooks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

// GetBookByID handles the GET /books/{id} request.
func GetBookByID(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()
	idStr := r.URL.Path[len("/books/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book ID"})
		return
	}
	book, errResp := store.GetBook(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// CreateBook handles the POST /books request.
// (For synchronization you might want to create in PostgreSQL first as shown in other controllers.)
func CreateBook(w http.ResponseWriter, r *http.Request) {
	bookStore := inmemoryStores.GetBookStoreInstance()
	authorStore := inmemoryStores.GetAuthorStoreInstance()
	pgStore := postgresStores.GetPostgresBookStoreInstance()

	var book StructureData.Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	if book.Stock < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Stock must be at least 1"})
		return
	}

	// Check if the author exists.
	authors := authorStore.GetAllAuthors()
	authorExists := false
	for _, existingAuthor := range authors {
		if existingAuthor.FirstName == book.Author.FirstName &&
			existingAuthor.LastName == book.Author.LastName &&
			existingAuthor.Bio == book.Author.Bio {
			book.Author = existingAuthor // Link existing author.
			authorExists = true
			break
		}
	}

	if !authorExists {
		createdAuthor, errResp := authorStore.CreateAuthor(book.Author)
		if errResp != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errResp)
			return
		}
		book.Author = createdAuthor
		if err := persistAuthorsToFile(authorStore); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving author data"})
			return
		}
	}

	// Create in PostgreSQL first (optional for synchronization).
	createdPgBook, pgErr := pgStore.CreateBook(book)
	if pgErr != nil {
		log.Printf("Error persisting book to PostgreSQL: %v", pgErr.Message)
		// You may choose to return an error here.
	}
	// Now create in the in-memory store using the PostgreSQL book (which carries the synchronized ID).
	createdBook, errResp := bookStore.CreateBook(createdPgBook)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if err := persistBooksToFile(bookStore.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving book data"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdBook)
}

// (The UpdateBook, DeleteBook, and SearchBooks functions remain unchanged.)

// persistBooksToFile saves all books to the JSON file in a pretty JSON format.
func persistBooksToFile(books []StructureData.Book) error {
	file, err := os.Create(bookFile)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(books)
}


// UpdateBook handles the PUT /books/{id} request.
func UpdateBook(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()
	pgStore := postgresStores.GetPostgresBookStoreInstance()

	idStr := r.URL.Path[len("/books/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book ID"})
		return
	}

	var book StructureData.Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate stock.
	if book.Stock < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Stock must be at least 1"})
		return
	}

	updatedBook, errResp := store.UpdateBook(id, book)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if err := persistBooksToFile(store.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	_, pgErr := pgStore.UpdateBook(id, updatedBook)
	if pgErr != nil {
		log.Printf("Error updating book in PostgreSQL: %v", pgErr.Message)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedBook)
}

// DeleteBook handles the DELETE /books/{id} request.
func DeleteBook(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()
	orderStore := inmemoryStores.GetOrderStoreInstance()
	pgStore := postgresStores.GetPostgresBookStoreInstance()

	idStr := r.URL.Path[len("/books/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book ID"})
		return
	}

	// Check if the book is linked to any orders.
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

	errResp := store.DeleteBook(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if err := persistBooksToFile(store.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Delete from PostgreSQL.
	pgErr := pgStore.DeleteBook(id)
	if pgErr != nil {
		log.Printf("Error deleting book from PostgreSQL: %v", pgErr.Message)
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchBooks handles the POST /books/search request.
func SearchBooks(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()

	var criteria StructureData.BookSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
		return
	}

	searchResults, errResp := store.SearchBooks(criteria)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResults)
}


