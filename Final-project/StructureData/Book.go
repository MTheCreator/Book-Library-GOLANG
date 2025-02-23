package StructureData

import ("time")



type Book struct {
	ID          int                   `json:"id"`
	Title       string                `json:"title"`
	Author      Author                `json:"author"`
	Genres      []string              `json:"genres,omitempty"` // Optional list of genres.
	PublishedAt time.Time             `json:"published_at"`
	Price       float64               `json:"price"`
	Stock       int                   `json:"stock"`
	CreatedAt   time.Time             `json:"created_at"`
	ReviewStats *BookReviewAggregate  `json:"review_stats,omitempty"` // Optional review summary.
}
// BookSearchCriteria allows filtering of books based on various fields.
type BookSearchCriteria struct {
	IDs              []int                `json:"ids,omitempty"`
	Titles           []string             `json:"titles,omitempty"`
	Genres           []string             `json:"genres,omitempty"`
	MinPublishedAt   time.Time            `json:"min_published_at,omitempty"`
	MaxPublishedAt   time.Time            `json:"max_published_at,omitempty"`
	MinPrice         float64              `json:"min_price,omitempty"`
	MaxPrice         float64              `json:"max_price,omitempty"`
	MinStock         int                  `json:"min_stock,omitempty"`
	MaxStock         int                  `json:"max_stock,omitempty"`
	AuthorCriteria   AuthorSearchCriteria `json:"author_criteria,omitempty"`
	// Optional criteria to filter by review statistics:
	MinAverageRating float64 `json:"min_average_rating,omitempty"`
	MaxAverageRating float64 `json:"max_average_rating,omitempty"`
	MinReviewCount   int     `json:"min_review_count,omitempty"`
	MaxReviewCount   int     `json:"max_review_count,omitempty"`
}