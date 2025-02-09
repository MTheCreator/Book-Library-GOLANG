package StructureData

import "time"
const (
	OrderStatusPending = "pending"
	OrderStatusSuccess = "success"
)
type Order struct {
	ID         int         `json:"id"`
	Customer   Customer    `json:"customer"`
	Items      []OrderItem `json:"items"`
	TotalPrice float64     `json:"total_price"`
	CreatedAt  time.Time   `json:"created_at"`
	Status     string      `json:"status"` // Either "pending" or "success"

}

// OrderSearchCriteria is used for filtering orders, including a status filter.
type OrderSearchCriteria struct {
	IDs           []int                   `json:"ids,omitempty"`
	CustomerIDs   []int                   `json:"customer_ids,omitempty"`
	MinTotalPrice float64                 `json:"min_total_price,omitempty"`
	MaxTotalPrice float64                 `json:"max_total_price,omitempty"`
	MinCreatedAt  time.Time               `json:"min_created_at,omitempty"`
	MaxCreatedAt  time.Time               `json:"max_created_at,omitempty"`
	Status        string                  `json:"status,omitempty"`
	ItemCriteria  OrderItemSearchCriteria `json:"item_criteria,omitempty"`
}