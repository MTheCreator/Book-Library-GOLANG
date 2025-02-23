package StructureData

import "time"

// Review represents a single review for a book.
type Review struct {
	ID         int       `json:"id"`                    // Unique review ID
	BookID     int       `json:"book_id"`               // The ID of the reviewed book
	CustomerID int       `json:"customer_id,omitempty"` // The ID of the customer who wrote the review (optional)
	Rating     int       `json:"rating"`                // Rating value (e.g., 1 to 5)
	ReviewText string    `json:"review_text"`           // The review content
	CreatedAt  time.Time `json:"created_at"`            // When the review was submitted
}

// ReviewSearchCriteria allows filtering of reviews based on various fields.
type ReviewSearchCriteria struct {
	BookIDs      []int     `json:"book_ids,omitempty"`       // Filter reviews for these book IDs
	CustomerIDs  []int     `json:"customer_ids,omitempty"`   // Filter reviews by these customer IDs
	MinRating    int       `json:"min_rating,omitempty"`     // Minimum rating value
	MaxRating    int       `json:"max_rating,omitempty"`     // Maximum rating value
	MinCreatedAt time.Time `json:"min_created_at,omitempty"` // Earliest review creation time
	MaxCreatedAt time.Time `json:"max_created_at,omitempty"` // Latest review creation time
}

// BookReviewAggregate provides a summary of reviews for a given book.
type BookReviewAggregate struct {
	AverageRating float64 `json:"average_rating"` // Average rating computed from all reviews
	ReviewCount   int     `json:"review_count"`   // Total number of reviews
}
