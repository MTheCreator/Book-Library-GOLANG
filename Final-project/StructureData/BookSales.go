package StructureData

type BookSales struct {
	Book Book `json:"book"`
	Quantity int `json:"quantity_sold"`
   }

type BookSalesSearchCriteria struct {
	BookCriteria BookSearchCriteria `json:"book_criteria,omitempty"`
	MinQuantity  int                `json:"min_quantity,omitempty"`
	MaxQuantity  int                `json:"max_quantity,omitempty"`
}
