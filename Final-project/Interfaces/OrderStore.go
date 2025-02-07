package Interfaces

import (
	data "finalProject/StructureData"
)

type OrderStore interface {
	CreateOrder(order data.Order) (data.Order, *data.ErrorResponse)
	GetOrder(id int) (data.Order, *data.ErrorResponse)
	UpdateOrder(id int, order data.Order) (data.Order, *data.ErrorResponse)
	DeleteOrder(id int) *data.ErrorResponse
	GetAllOrders() []data.Order
	SearchOrders(criteria data.OrderSearchCriteria) ([]data.Order, *data.ErrorResponse)
}
