package Interfaces

import (
	data "finalProject/StructureData"
)

type BookStore interface {
	CreateBook(book data.Book) (data.Book, *data.ErrorResponse)
	GetBook(id int) (data.Book, *data.ErrorResponse)
	UpdateBook(id int, book data.Book) (data.Book, *data.ErrorResponse)
	DeleteBook(id int) *data.ErrorResponse
	GetAllBooks() []data.Book
	AddBookDirectly(book data.Book)
	SearchBooks(criteria data.BookSearchCriteria) ([]data.Book, *data.ErrorResponse)
}
