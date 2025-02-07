package Interfaces

import (
	data "finalProject/StructureData"
)

type AuthorStore interface {
	CreateAuthor(author data.Author) (data.Author, *data.ErrorResponse)
	GetAuthor(id int) (data.Author, *data.ErrorResponse)
	UpdateAuthor(id int, author data.Author) (data.Author, *data.ErrorResponse)
	DeleteAuthor(id int) *data.ErrorResponse
	SearchAuthors(criteria data.AuthorSearchCriteria) ([]data.Author, *data.ErrorResponse)
	GetAllAuthors() []data.Author 
}
