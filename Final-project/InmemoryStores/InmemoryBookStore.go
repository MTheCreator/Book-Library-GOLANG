// File: InmemoryStores/bookStore.go
package InmemoryStores

import (
	"sync"

	interfaces "finalProject/Interfaces"
	
	data "finalProject/StructureData"
	"finalProject/utils"
)

type InMemoryBookStore struct {
	mu     sync.RWMutex
	books  map[int]data.Book
	nextID int
}

var (
	bookStoreInstance *InMemoryBookStore
	bookOnce          sync.Once
)

// GetBookStoreInstance returns the singleton instance of InMemoryBookStore.
func GetBookStoreInstance() interfaces.BookStore {
	bookOnce.Do(func() {
		bookStoreInstance = &InMemoryBookStore{
			books:  make(map[int]data.Book),
			nextID: 1,
		}
	})
	return bookStoreInstance
}

// CreateBook adds a new book to the store.
func (store *InMemoryBookStore) CreateBook(book data.Book) (data.Book, *data.ErrorResponse) {
	store.mu.Lock()
	defer store.mu.Unlock()

	if book.Stock < 1 {
		return data.Book{}, &data.ErrorResponse{Message: "Book stock must be at least 1"}
	}

	if book.ID != 0 {
		if _, exists := store.books[book.ID]; exists {
			return data.Book{}, &data.ErrorResponse{Message: "Book ID already exists"}
		}
		store.books[book.ID] = book
		if book.ID >= store.nextID {
			store.nextID = book.ID + 1
		}
	} else {
		book.ID = store.nextID
		store.nextID++
		store.books[book.ID] = book
	}
	return book, nil
}

// (Other methods remain unchanged.)


// GetBook retrieves a book by its ID
func (store *InMemoryBookStore) GetBook(id int) (data.Book, *data.ErrorResponse) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	book, exists := store.books[id]
	if !exists {
		return data.Book{}, &data.ErrorResponse{Message: "Book not found"}
	}
	return book, nil
}

// UpdateBook updates the details of an existing book
func (store *InMemoryBookStore) UpdateBook(id int, book data.Book) (data.Book, *data.ErrorResponse) {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.books[id]
	if !exists {
		return data.Book{}, &data.ErrorResponse{Message: "Book not found"}
	}
	book.ID = id
	store.books[id] = book
	return book, nil
}

// DeleteBook removes a book from the store
func (store *InMemoryBookStore) DeleteBook(id int) *data.ErrorResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.books[id]
	if !exists {
		return &data.ErrorResponse{Message: "Book not found"}
	}
	delete(store.books, id)
	return nil
}

// GetAllBooks retrieves all books
func (store *InMemoryBookStore) GetAllBooks() []data.Book {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var books []data.Book
	for _, book := range store.books {
		books = append(books, book)
	}
	return books
}

func (store *InMemoryBookStore) SearchBooks(criteria data.BookSearchCriteria) ([]data.Book, *data.ErrorResponse) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var result []data.Book
	for _, book := range store.books {
		// Filter by basic book fields.
		if len(criteria.IDs) > 0 && !utils.ContainsInt(criteria.IDs, book.ID) {
			continue
		}
		if len(criteria.Titles) > 0 && !utils.ContainsString(criteria.Titles, book.Title) {
			continue
		}
		if len(criteria.Genres) > 0 && !utils.ContainsAnyString(criteria.Genres, book.Genres) {
			continue
		}
		if !criteria.MinPublishedAt.IsZero() && book.PublishedAt.Before(criteria.MinPublishedAt) {
			continue
		}
		if !criteria.MaxPublishedAt.IsZero() && book.PublishedAt.After(criteria.MaxPublishedAt) {
			continue
		}
		if criteria.MinPrice > 0 && book.Price < criteria.MinPrice {
			continue
		}
		if criteria.MaxPrice > 0 && book.Price > criteria.MaxPrice {
			continue
		}
		if criteria.MinStock > 0 && book.Stock < criteria.MinStock {
			continue
		}
		if criteria.MaxStock > 0 && book.Stock > criteria.MaxStock {
			continue
		}
		if !utils.MatchAuthorCriteria(book.Author, criteria.AuthorCriteria) {
			continue
		}

		// Optional: Filter by review statistics.
		// Use 0 as default if ReviewStats is nil.
		avgRating := 0.0
		reviewCount := 0
		if book.ReviewStats != nil {
			avgRating = book.ReviewStats.AverageRating
			reviewCount = book.ReviewStats.ReviewCount
		}
		if criteria.MinAverageRating > 0 && avgRating < criteria.MinAverageRating {
			continue
		}
		if criteria.MaxAverageRating > 0 && avgRating > criteria.MaxAverageRating {
			continue
		}
		if criteria.MinReviewCount > 0 && reviewCount < criteria.MinReviewCount {
			continue
		}
		if criteria.MaxReviewCount > 0 && reviewCount > criteria.MaxReviewCount {
			continue
		}

		result = append(result, book)
	}

	return result, nil
}

func (store *InMemoryBookStore) AddBookDirectly(book data.Book) {
	store.mu.Lock()
	defer store.mu.Unlock()

	// Ensure the next ID is updated to prevent ID collisions
	if book.ID >= store.nextID {
		store.nextID = book.ID + 1
	}

	store.books[book.ID] = book
}
