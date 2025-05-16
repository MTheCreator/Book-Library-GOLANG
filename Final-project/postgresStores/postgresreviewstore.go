package postgresStores

import (
	"database/sql"
	"finalProject/StructureData"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// PostgresReviewStore implements the review storage using PostgreSQL.
type PostgresReviewStore struct {
	db *sql.DB
}

var postgresReviewStoreInstance *PostgresReviewStore

// GetPostgresReviewStoreInstance returns a singleton instance of PostgresReviewStore.
func GetPostgresReviewStoreInstance() *PostgresReviewStore {
	if postgresReviewStoreInstance == nil {
		host := getEnvR("DB_HOST", "db")
		port := getEnvR("DB_PORT", "5432")
		user := getEnvR("DB_USER", "postgres")
		password := getEnvR("DB_PASSWORD", "root")
		dbname := getEnvR("DB_NAME", "booklibrary")
		
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
		
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			panic(fmt.Sprintf("Failed to connect to Postgres for reviews: %v", err))
		}
		if err := db.Ping(); err != nil {
			panic(fmt.Sprintf("Failed to ping Postgres for reviews: %v", err))
		}
		postgresReviewStoreInstance = &PostgresReviewStore{db: db}
		log.Println("Connected to Postgres for reviews.")
	}
	return postgresReviewStoreInstance
}

// Helper function to get environment variables with defaults
func getEnvR(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
// Close gracefully closes the database connection.
func (store *PostgresReviewStore) Close() error {
	return store.db.Close()
}

// CreateReview inserts a new review into the reviews table.
func (store *PostgresReviewStore) CreateReview(review StructureData.Review) (StructureData.Review, *StructureData.ErrorResponse) {
	query := `
		INSERT INTO reviews (book_id, customer_id, rating, review_text, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	err := store.db.QueryRow(query, review.BookID, review.CustomerID, review.Rating, review.ReviewText, review.CreatedAt).Scan(&review.ID)
	if err != nil {
		return StructureData.Review{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to create review: %v", err)}
	}
	go updateBookReviewStats(review.BookID)
    return review, nil
}

// GetReviewsByBookID retrieves all reviews for a given book, ordered by creation time (most recent first).
func (store *PostgresReviewStore) GetReviewsByBookID(bookID int) ([]StructureData.Review, *StructureData.ErrorResponse) {
	query := `
		SELECT id, book_id, customer_id, rating, review_text, created_at
		FROM reviews
		WHERE book_id = $1
		ORDER BY created_at DESC`
	rows, err := store.db.Query(query, bookID)
	if err != nil {
		return nil, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to fetch reviews: %v", err)}
	}
	defer rows.Close()

	var reviews []StructureData.Review
	for rows.Next() {
		var r StructureData.Review
		err := rows.Scan(&r.ID, &r.BookID, &r.CustomerID, &r.Rating, &r.ReviewText, &r.CreatedAt)
		if err != nil {
			log.Printf("Error scanning review: %v", err)
			continue
		}
		reviews = append(reviews, r)
	}
	return reviews, nil
}

func updateBookReviewStats(bookID int) {
    stats, err := GetPostgresReviewStoreInstance().GetBookReviewStats(bookID)
    if err != nil {
        log.Printf("Failed to update review stats for book %d: %v", bookID, err)
        return
    }

    bookStore := GetPostgresBookStoreInstance()
    book, errResp := bookStore.GetBook(bookID)
    if errResp != nil {
        log.Printf("Book %d not found for stats update: %v", bookID, errResp)
        return
    }

    book.ReviewStats = &stats
    _, errResp = bookStore.UpdateBook(bookID, book)
    if errResp != nil {
        log.Printf("Failed to update book %d review stats: %v", bookID, errResp)
    }
}

// DeleteReview removes a review from the database.
func (store *PostgresReviewStore) DeleteReview(id int) *StructureData.ErrorResponse {
	query := `DELETE FROM reviews WHERE id = $1`
	res, err := store.db.Exec(query, id)
	if err != nil {
		return &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to delete review: %v", err)}
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return &StructureData.ErrorResponse{Message: "Review not found"}
	}
	return nil
}
// Add to postgresStores/reviewStore.go
func (store *PostgresReviewStore) GetBookReviewStats(bookID int) (StructureData.BookReviewAggregate, error) {
    query := `
        SELECT 
            COALESCE(AVG(rating), 0), 
            COUNT(*) 
        FROM reviews 
        WHERE book_id = $1`
    
    var stats StructureData.BookReviewAggregate
    err := store.db.QueryRow(query, bookID).Scan(
        &stats.AverageRating,
        &stats.ReviewCount,
    )
    
    if err != nil {
        return StructureData.BookReviewAggregate{}, &StructureData.ErrorResponse{
            Message: "Failed to calculate review stats",
        }
    }
    return stats, nil
}
