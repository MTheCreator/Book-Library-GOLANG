package Controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	inmemoryStores "finalProject/InmemoryStores"
	"finalProject/StructureData"
	postgresStores "finalProject/postgresStores"
)

func InitializeBookFile() {
    pgStore := postgresStores.GetPostgresBookStoreInstance()
    memStore := inmemoryStores.GetBookStoreInstance()

    pgBooks := pgStore.GetAllBooks()
    
    if len(memStore.GetAllBooks()) == 0 {
        for _, book := range pgBooks {
            _, err := memStore.CreateBook(book)
            if err != nil {
                log.Printf("Error loading book %d into memory: %v", book.ID, err.Message)
            }
        }
        log.Printf("Loaded %d books from PostgreSQL into memory", len(pgBooks))
    }
}

func GetAllBooks(w http.ResponseWriter, r *http.Request) {
	store := postgresStores.GetPostgresBookStoreInstance() // use Postgres store
	books := store.GetAllBooks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func GetBookByID(w http.ResponseWriter, r *http.Request) {
	store := postgresStores.GetPostgresBookStoreInstance() // use Postgres store
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

// BookInput is used for creating a new book.
type BookInput struct {
	Title       string    `json:"title"`
	AuthorID    int       `json:"author_id"`
	Genres      []string  `json:"genres"`
	PublishedAt time.Time `json:"published_at"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	// You can omit review_stats since those are computed later.
}

func CreateBook(w http.ResponseWriter, r *http.Request) {
	bookStore := inmemoryStores.GetBookStoreInstance()
	pgBookStore := postgresStores.GetPostgresBookStoreInstance()
	pgAuthorStore := postgresStores.GetPostgresAuthorStoreInstance()

	var input BookInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate stock.
	if input.Stock < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Stock must be ≥1"})
		return
	}

	// Look up the author in PostgreSQL using the provided author_id.
	if input.AuthorID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Author ID is required"})
		return
	}
	author, errResp := pgAuthorStore.GetAuthor(input.AuthorID)
	if errResp != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Author not found"})
		return
	}

	// Build the Book struct. We ignore any incoming book ID and set CreatedAt to now.
	book := StructureData.Book{
		Title:       input.Title,
		Author:      author, // now using the looked‑up author
		Genres:      input.Genres,
		PublishedAt: input.PublishedAt,
		Price:       input.Price,
		Stock:       input.Stock,
		CreatedAt:   time.Now(),
	}

	// Create the book in PostgreSQL.
	createdPgBook, pgErr := pgBookStore.CreateBook(book)
	if pgErr != nil {
		log.Printf("Error creating book in PostgreSQL: %v", pgErr.Message)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(pgErr)
		return
	}

	// Create the book in the in-memory store.
	createdBook, errResp := bookStore.CreateBook(createdPgBook)
	if errResp != nil {
		// Roll back PostgreSQL creation if needed.
		pgBookStore.DeleteBook(createdPgBook.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdBook)
}

func UpdateBook(w http.ResponseWriter, r *http.Request) {
    // Get instances of the in-memory and PostgreSQL BookStores,
    // and the in-memory OrderStore.
    bookStore := inmemoryStores.GetBookStoreInstance()
    pgBookStore := postgresStores.GetPostgresBookStoreInstance()
    orderStore := inmemoryStores.GetOrderStoreInstance()

    // Extract book ID from URL.
    idStr := r.URL.Path[len("/books/"):]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book ID"})
        return
    }

    // Check if the book is referenced by any orders.
    orders := orderStore.GetAllOrders()
    for _, order := range orders {
        for _, item := range order.Items {
            if item.Book.ID == id {
                w.WriteHeader(http.StatusConflict)
                json.NewEncoder(w).Encode(StructureData.ErrorResponse{
                    Message: "Cannot update book that exists in existing orders",
                })
                return
            }
        }
    }

    // Retrieve the existing book from the in-memory store to preserve its author.
    existingBook, errResp := bookStore.GetBook(id)
    if errResp != nil {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(errResp)
        return
    }

    // Decode the updated book data from the request body.
    var updatedBook StructureData.Book
    if err := json.NewDecoder(r.Body).Decode(&updatedBook); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
        return
    }

    // Prevent author update by preserving the existing author.
    updatedBook.Author = existingBook.Author

    // Validate stock.
    if updatedBook.Stock < 1 {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Stock must be ≥1"})
        return
    }

    // Update the book in PostgreSQL first.
    pgUpdatedBook, pgErr := pgBookStore.UpdateBook(id, updatedBook)
    if pgErr != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(pgErr)
        return
    }

    // Then update the in-memory store to synchronize.
    finalBook, errResp := bookStore.UpdateBook(id, pgUpdatedBook)
    if errResp != nil {
        // If in-memory update fails, you might consider rolling back the PG update.
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(errResp)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(finalBook)
}


func DeleteBook(w http.ResponseWriter, r *http.Request) {
	// Get the necessary store instances.
	bookStore := inmemoryStores.GetBookStoreInstance()
	orderStore := inmemoryStores.GetOrderStoreInstance()
	pgBookStore := postgresStores.GetPostgresBookStoreInstance()

	// Extract the book ID from the URL.
	idStr := r.URL.Path[len("/books/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book ID"})
		return
	}

	// Check if the book is referenced in any orders.
	orders := orderStore.GetAllOrders()
	for _, order := range orders {
		for _, item := range order.Items {
			if item.Book.ID == id {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Book linked to orders"})
				return
			}
		}
	}

	// Delete the book from the in-memory store.
	errResp := bookStore.DeleteBook(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Delete the book from PostgreSQL.
	pgErr := pgBookStore.DeleteBook(id)
	if pgErr != nil {
		log.Printf("Error deleting book from PostgreSQL: %v", pgErr.Message)
	}

	w.WriteHeader(http.StatusNoContent)
}


func SearchBooks(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetBookStoreInstance()
	var criteria StructureData.BookSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid criteria"})
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