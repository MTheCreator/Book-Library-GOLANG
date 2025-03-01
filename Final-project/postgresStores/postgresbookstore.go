// File: postgresStores/bookStore.go
package postgresStores

import (
	"database/sql"
	"finalProject/StructureData"
	"fmt"
	"log"

	"github.com/lib/pq"
)

// PostgresBookStore implements the BookStore interface using PostgreSQL.
type PostgresBookStore struct {
	db *sql.DB
}

// Close gracefully closes the underlying DB connection.
func (store *PostgresBookStore) Close() error {
	return store.db.Close()
}

var postgresBookStoreInstance *PostgresBookStore

// GetPostgresBookStoreInstance returns a singleton instance of PostgresBookStore.
func GetPostgresBookStoreInstance() *PostgresBookStore {
	if postgresBookStoreInstance == nil {
		connStr := "user=postgres password=root dbname=booklibrary sslmode=disable"
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			panic(fmt.Sprintf("Failed to connect to Postgres: %v", err))
		}
		if err := db.Ping(); err != nil {
			panic(fmt.Sprintf("Failed to ping Postgres: %v", err))
		}
		postgresBookStoreInstance = &PostgresBookStore{db: db}
	}
	return postgresBookStoreInstance
}

// CreateBook inserts a new book into the database.
func (store *PostgresBookStore) CreateBook(book StructureData.Book) (StructureData.Book, *StructureData.ErrorResponse) {
	var query string
	var args []interface{}
	if book.ID != 0 {
		query = `INSERT INTO books (id, title, author_id, genres, published_at, price, stock)
		          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
		args = []interface{}{
			book.ID,
			book.Title,
			book.Author.ID,
			pq.Array(book.Genres),
			book.PublishedAt,
			book.Price,
			book.Stock,
		}
	} else {
		query = `INSERT INTO books (title, author_id, genres, published_at, price, stock)
		          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
		args = []interface{}{
			book.Title,
			book.Author.ID,
			pq.Array(book.Genres),
			book.PublishedAt,
			book.Price,
			book.Stock,
		}
	}
	err := store.db.QueryRow(query, args...).Scan(&book.ID)
	if err != nil {
		return StructureData.Book{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to insert book: %v", err)}
	}
	return book, nil
}

func (store *PostgresBookStore) GetBook(id int) (StructureData.Book, *StructureData.ErrorResponse) {
	var book StructureData.Book
	var genres []string
	var authorID int

	query := `SELECT id, title, author_id, genres, published_at, price, stock FROM books WHERE id=$1`
	row := store.db.QueryRow(query, id)
	err := row.Scan(&book.ID, &book.Title, &authorID, pq.Array(&genres), &book.PublishedAt, &book.Price, &book.Stock)
	if err != nil {
		if err == sql.ErrNoRows {
			return StructureData.Book{}, &StructureData.ErrorResponse{Message: "Book not found"}
		}
		return StructureData.Book{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Error fetching book: %v", err)}
	}
	book.Genres = genres

	// Now fetch the author from PostgresAuthorStore to get full details.
	authorStore := GetPostgresAuthorStoreInstance()
	author, authErr := authorStore.GetAuthor(authorID)
	if authErr != nil {
		// If the author isn't found, you might decide to return an error
		// or keep a partial author. Let's just keep a partial author for now.
		// Or you can do:
		// return StructureData.Book{}, authErr
		log.Printf("Warning: Author ID %d not found for book %d: %v", authorID, book.ID, authErr)
	} else {
		book.Author = author
	}

	// Retrieve and set review stats, if available.
	reviewStore := GetPostgresReviewStoreInstance()
	stats, err := reviewStore.GetBookReviewStats(id)
	if err == nil {
		book.ReviewStats = &stats
	}

	return book, nil
}

// UpdateBook updates an existing book in the database.
func (store *PostgresBookStore) UpdateBook(id int, book StructureData.Book) (StructureData.Book, *StructureData.ErrorResponse) {
	query := `UPDATE books SET title=$1, author_id=$2, genres=$3, published_at=$4, price=$5, stock=$6 WHERE id=$7`
	res, err := store.db.Exec(query,
		book.Title,
		book.Author.ID,
		pq.Array(book.Genres),
		book.PublishedAt,
		book.Price,
		book.Stock,
		id,
	)
	if err != nil {
		return StructureData.Book{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to update book: %v", err)}
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return StructureData.Book{}, &StructureData.ErrorResponse{Message: "Book not found"}
	}
	book.ID = id
	return book, nil
}

// DeleteBook removes a book from the database.
func (store *PostgresBookStore) DeleteBook(id int) *StructureData.ErrorResponse {
	query := `DELETE FROM books WHERE id=$1`
	res, err := store.db.Exec(query, id)
	if err != nil {
		return &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to delete book: %v", err)}
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return &StructureData.ErrorResponse{Message: "Book not found"}
	}
	return nil
}

func (store *PostgresBookStore) GetAllBooks() []StructureData.Book {
	books := []StructureData.Book{}
	query := `SELECT id, title, author_id, genres, published_at, price, stock FROM books`
	rows, err := store.db.Query(query)
	if err != nil {
		return books
	}
	defer rows.Close()

	authorStore := GetPostgresAuthorStoreInstance()

	for rows.Next() {
		var book StructureData.Book
		var genres []string
		var authorID int
		err := rows.Scan(&book.ID, &book.Title, &authorID, pq.Array(&genres), &book.PublishedAt, &book.Price, &book.Stock)
		if err != nil {
			continue
		}
		book.Genres = genres

		// Retrieve full author details
		author, authErr := authorStore.GetAuthor(authorID)
		if authErr == nil {
			book.Author = author
		} else {
			log.Printf("Warning: Author ID %d not found for book %d: %v", authorID, book.ID, authErr)
		}

		// Retrieve and set review stats
		reviewStore := GetPostgresReviewStoreInstance()
		stats, statsErr := reviewStore.GetBookReviewStats(book.ID)
		if statsErr == nil {
			book.ReviewStats = &stats
		}

		books = append(books, book)
	}
	return books
}

// SearchBooks retrieves all books and filters them in memory based on search criteria.
func (store *PostgresBookStore) SearchBooks(criteria StructureData.BookSearchCriteria) ([]StructureData.Book, *StructureData.ErrorResponse) {
	// For simplicity, we retrieve all books and apply in-memory filtering.
	allBooks := store.GetAllBooks()
	var result []StructureData.Book
	for _, book := range allBooks {
		if len(criteria.IDs) > 0 && !containsInt(criteria.IDs, book.ID) {
			continue
		}
		if len(criteria.Titles) > 0 && !containsString(criteria.Titles, book.Title) {
			continue
		}
		if len(criteria.Genres) > 0 && !containsAnyString(criteria.Genres, book.Genres) {
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
		// (Optional: match AuthorCriteria if needed.)
		result = append(result, book)
	}
	return result, nil
}

// Helper functions for in-memory filtering.
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func containsString(slice []string, val string) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func containsAnyString(target []string, source []string) bool {
	for _, t := range target {
		for _, s := range source {
			if t == s {
				return true
			}
		}
	}
	return false
}
