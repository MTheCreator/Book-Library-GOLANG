package StructureData

import ("time")



type Book struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Author      Author    `json:"author"`
	Genres      []string  `json:"genres"`
	PublishedAt time.Time `json:"published_at"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
}
type BookSearchCriteria struct {
	IDs            []int       `json:"ids,omitempty"`
	Titles         []string    `json:"titles,omitempty"`
	Genres         []string    `json:"genres,omitempty"`
	MinPublishedAt time.Time   `json:"min_published_at,omitempty"`
	MaxPublishedAt time.Time   `json:"max_published_at,omitempty"`
	MinPrice       float64     `json:"min_price,omitempty"`
	MaxPrice       float64     `json:"max_price,omitempty"`
	MinStock       int         `json:"min_stock,omitempty"`
	MaxStock       int         `json:"max_stock,omitempty"`
	AuthorCriteria AuthorSearchCriteria `json:"author_criteria,omitempty"`
}