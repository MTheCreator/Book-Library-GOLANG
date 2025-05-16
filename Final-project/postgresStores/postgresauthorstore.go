// File: postgresStores/authorStore.go
package postgresStores

import (
	"database/sql"
	"finalProject/StructureData"
	"fmt"
	_ "log"
	"strings"
	"os"

	_ "github.com/lib/pq"
)

type PostgresAuthorStore struct {
	db *sql.DB
}

// Close gracefully closes the underlying DB connection.
func (store *PostgresAuthorStore) Close() error {
	return store.db.Close()
}

var postgresAuthorStoreInstance *PostgresAuthorStore



// GetPostgresAuthorStoreInstance returns a singleton instance of PostgresAuthorStore.
func GetPostgresAuthorStoreInstance() *PostgresAuthorStore {
	if postgresAuthorStoreInstance == nil {
		// Get database connection parameters from environment variables
		dbHost := getEnvA("DB_HOST", "localhost")
		dbPort := getEnvA("DB_PORT", "5432")
		dbUser := getEnvA("DB_USER", "postgres")
		dbPassword := getEnvA("DB_PASSWORD", "root")
		dbName := getEnvA("DB_NAME", "booklibrary")

		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
		
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			panic(fmt.Sprintf("Failed to connect to Postgres: %v", err))
		}
		if err := db.Ping(); err != nil {
			panic(fmt.Sprintf("Failed to ping Postgres: %v", err))
		}
		postgresAuthorStoreInstance = &PostgresAuthorStore{db: db}
	}
	return postgresAuthorStoreInstance
}

// Helper function to get environment variable with default fallback
func getEnvA(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// Rest of the file remains the same...
// CreateAuthor inserts a new author into the database.
func (store *PostgresAuthorStore) CreateAuthor(author StructureData.Author) (StructureData.Author, *StructureData.ErrorResponse) {
	var query string
	var args []interface{}
	if author.ID != 0 {
		query = `INSERT INTO authors (id, first_name, last_name, bio) VALUES ($1, $2, $3, $4) RETURNING id`
		args = []interface{}{
			author.ID,
			author.FirstName,
			author.LastName,
			author.Bio,
		}
	} else {
		query = `INSERT INTO authors (first_name, last_name, bio) VALUES ($1, $2, $3) RETURNING id`
		args = []interface{}{
			author.FirstName,
			author.LastName,
			author.Bio,
		}
	}
	err := store.db.QueryRow(query, args...).Scan(&author.ID)
	if err != nil {
		return StructureData.Author{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to insert author: %v", err)}
	}
	return author, nil
}

// GetAuthor retrieves an author by its ID.
func (store *PostgresAuthorStore) GetAuthor(id int) (StructureData.Author, *StructureData.ErrorResponse) {
	var author StructureData.Author
	query := `SELECT id, first_name, last_name, bio FROM authors WHERE id=$1`
	row := store.db.QueryRow(query, id)
	err := row.Scan(&author.ID, &author.FirstName, &author.LastName, &author.Bio)
	if err != nil {
		if err == sql.ErrNoRows {
			return StructureData.Author{}, &StructureData.ErrorResponse{Message: "Author not found"}
		}
		return StructureData.Author{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Error fetching author: %v", err)}
	}
	return author, nil
}
func (s *PostgresAuthorStore) GetAuthorByDetails(firstName, lastName, bio string) (StructureData.Author, *StructureData.ErrorResponse) {
	var author StructureData.Author
	err := s.db.QueryRow(
		"SELECT id, first_name, last_name, bio FROM authors WHERE first_name = $1 AND last_name = $2 AND bio = $3",
		firstName,
		lastName,
		bio,
	).Scan(&author.ID, &author.FirstName, &author.LastName, &author.Bio)

	if err != nil {
		if err == sql.ErrNoRows {
			return StructureData.Author{}, &StructureData.ErrorResponse{
				Message: "Author not found",
			}
		}
		return StructureData.Author{}, &StructureData.ErrorResponse{
			Message: fmt.Sprintf("Database error: %v", err),
		}
	}
	return author, nil
}
// UpdateAuthor updates an existing author in the database.
func (store *PostgresAuthorStore) UpdateAuthor(id int, author StructureData.Author) (StructureData.Author, *StructureData.ErrorResponse) {
	query := `UPDATE authors SET first_name=$1, last_name=$2, bio=$3 WHERE id=$4`
	res, err := store.db.Exec(query, author.FirstName, author.LastName, author.Bio, id)
	if err != nil {
		return StructureData.Author{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to update author: %v", err)}
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return StructureData.Author{}, &StructureData.ErrorResponse{Message: "Author not found"}
	}
	author.ID = id
	return author, nil
}

// DeleteAuthor removes an author from the database.
func (store *PostgresAuthorStore) DeleteAuthor(id int) *StructureData.ErrorResponse {
	query := `DELETE FROM authors WHERE id=$1`
	res, err := store.db.Exec(query, id)
	if err != nil {
		return &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to delete author: %v", err)}
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return &StructureData.ErrorResponse{Message: "Author not found"}
	}
	return nil
}

// GetAllAuthors retrieves all authors from the database.
func (store *PostgresAuthorStore) GetAllAuthors() []StructureData.Author {
	authors := []StructureData.Author{}
	query := `SELECT id, first_name, last_name, bio FROM authors`
	rows, err := store.db.Query(query)
	if err != nil {
		return authors
	}
	defer rows.Close()
	for rows.Next() {
		var author StructureData.Author
		err = rows.Scan(&author.ID, &author.FirstName, &author.LastName, &author.Bio)
		if err != nil {
			continue
		}
		authors = append(authors, author)
	}
	return authors
}

// SearchAuthors filters authors based on the search criteria.
func (store *PostgresAuthorStore) SearchAuthors(criteria StructureData.AuthorSearchCriteria) ([]StructureData.Author, *StructureData.ErrorResponse) {
	allAuthors := store.GetAllAuthors()
	var result []StructureData.Author
	for _, author := range allAuthors {
		if len(criteria.IDs) > 0 {
			found := false
			for _, id := range criteria.IDs {
				if author.ID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(criteria.FirstNames) > 0 {
			found := false
			for _, fn := range criteria.FirstNames {
				if author.FirstName == fn {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(criteria.LastNames) > 0 {
			found := false
			for _, ln := range criteria.LastNames {
				if author.LastName == ln {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(criteria.Keywords) > 0 {
			matched := false
			for _, keyword := range criteria.Keywords {
				if containsIgnoreCase(author.FirstName, keyword) ||
					containsIgnoreCase(author.LastName, keyword) ||
					containsIgnoreCase(author.Bio, keyword) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		result = append(result, author)
	}
	return result, nil
}

// Helper function: case-insensitive substring match.
func containsIgnoreCase(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}
