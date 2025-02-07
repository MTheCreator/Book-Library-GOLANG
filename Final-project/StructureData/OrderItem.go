package StructureData

type OrderItem struct {
	Book Book `json:"book"`
	Quantity int `json:"quantity"`
   }
   
type OrderItemSearchCriteria struct {
	BookCriteria BookSearchCriteria `json:"book_criteria,omitempty"`
	MinQuantity  int                `json:"min_quantity,omitempty"`
	MaxQuantity  int                `json:"max_quantity,omitempty"`
}
