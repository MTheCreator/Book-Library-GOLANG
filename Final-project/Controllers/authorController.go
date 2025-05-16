package Controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	inmemoryStores "finalProject/InmemoryStores"
	postgresStores "finalProject/postgresStores"
	"finalProject/StructureData"
)

func InitializeAuthorFile() {
    pgStore := postgresStores.GetPostgresAuthorStoreInstance()
    memStore := inmemoryStores.GetAuthorStoreInstance()

    pgAuthors := pgStore.GetAllAuthors()
    
    if len(memStore.GetAllAuthors()) == 0 {
        for _, author := range pgAuthors {
            _, err := memStore.CreateAuthor(author)
            if err != nil {
                log.Printf("Error loading author %d into memory: %v", author.ID, err.Message)
            }
        }
        log.Printf("Loaded %d authors from PostgreSQL into memory", len(pgAuthors))
    }
}
func GetAllAuthors(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()
	authors, _ := store.SearchAuthors(StructureData.AuthorSearchCriteria{})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authors)
}

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

func CreateAuthor(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()
	pgStore := postgresStores.GetPostgresAuthorStoreInstance()

	var author StructureData.Author
	if err := json.NewDecoder(r.Body).Decode(&author); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Check if author already exists in PostgreSQL
	existingAuthor, err := pgStore.GetAuthorByDetails(
		author.FirstName,
		author.LastName,
		author.Bio,
	)

	if err == nil {
		// Author exists in PostgreSQL, check in-memory store
		memAuthor, memErr := store.GetAuthor(existingAuthor.ID)
		if memErr != nil {
			// Add to in-memory store if missing
			_, _ = store.CreateAuthor(existingAuthor)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(memAuthor)
		return
	}

	// Create new author in PostgreSQL first
	createdPgAuthor, pgErr := pgStore.CreateAuthor(author)
	if pgErr != nil {
		json.NewEncoder(w).Encode(pgErr)
		return
	}

	// Sync to in-memory store
	createdAuthor, errResp := store.CreateAuthor(createdPgAuthor)
	if errResp != nil {
		// Rollback PostgreSQL creation
		pgStore.DeleteAuthor(createdPgAuthor.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdAuthor)
}

func UpdateAuthor(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetAuthorStoreInstance()
	pgStore := postgresStores.GetPostgresAuthorStoreInstance()

	idStr := r.URL.Path[len("/authors/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid author ID"})
		return
	}

	// Check if any book is associated with this author.
	bookStore := inmemoryStores.GetBookStoreInstance()
	books := bookStore.GetAllBooks()
	for _, book := range books {
		if book.Author.ID == id {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{
				Message: "Author cannot be updated because it is referenced by existing books",
			})
			return
		}
	}

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

	_, pgErr := pgStore.UpdateAuthor(id, updatedAuthor)
	if pgErr != nil {
		log.Printf("Error updating author in PostgreSQL: %v", pgErr.Message)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAuthor)
}

func DeleteAuthor(w http.ResponseWriter, r *http.Request) {
	authorStore := inmemoryStores.GetAuthorStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()
	orderStore := inmemoryStores.GetOrderStoreInstance()

	pgAuthorStore := postgresStores.GetPostgresAuthorStoreInstance()
	pgBookStore := postgresStores.GetPostgresBookStoreInstance() // Assumes this function exists

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

	// Iterate over all books to check if any reference this author.
	books := bookStore.GetAllBooks()
	for _, book := range books {
		if book.Author.ID == id {
			// Check if the book is referenced in any orders.
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
			// If the book is not referenced, delete it from both stores.
			if !bookInOrder {
				bookStore.DeleteBook(book.ID)
				pgBookStore.DeleteBook(book.ID)
			}
		}
	}

	// Delete the author from PostgreSQL.
	pgErr := pgAuthorStore.DeleteAuthor(id)
	if pgErr != nil {
		log.Printf("Error deleting author from PostgreSQL: %v", pgErr.Message)
	}

	w.WriteHeader(http.StatusNoContent)
}


func SearchAuthors(w http.ResponseWriter, r *http.Request) {
    pgStore := postgresStores.GetPostgresAuthorStoreInstance() // Use PostgreSQL
    var criteria StructureData.AuthorSearchCriteria
    if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid criteria"})
        return
    }
    authors, errResp := pgStore.SearchAuthors(criteria) // Query PostgreSQL
    if errResp != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(errResp)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(authors)
}