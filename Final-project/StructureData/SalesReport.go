package StructureData

import "time"





type SalesReport struct {
    Timestamp        time.Time        `json:"timestamp"`
    TotalRevenue     float64          `json:"total_revenue"`
    TotalOrders      int              `json:"total_orders"`
    SuccessfulOrders int              `json:"successful_orders"`
    PendingOrders    int              `json:"pending_orders"`
    TopSellingBooks  []TopSellingBook `json:"top_selling_books"`
}

type TopSellingBook struct {
    Book          Book    `json:"book"`
    QuantitySold  int     `json:"quantity_sold"`
    TotalRevenue  float64 `json:"total_revenue"`
}
type SalesReportSearchCriteria struct {
	MinTimestamp     time.Time               `json:"min_timestamp,omitempty"`
	MaxTimestamp     time.Time               `json:"max_timestamp,omitempty"`
	MinRevenue       float64                 `json:"min_revenue,omitempty"`
	MaxRevenue       float64                 `json:"max_revenue,omitempty"`
	MinOrders        int                     `json:"min_orders,omitempty"`
	MaxOrders        int                     `json:"max_orders,omitempty"`
	TopBooksCriteria BookSalesSearchCriteria `json:"top_books_criteria,omitempty"`
}
