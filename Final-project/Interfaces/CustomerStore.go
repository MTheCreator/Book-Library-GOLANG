package Interfaces

import (
	data "finalProject/StructureData"
)

type CustomerStore interface {
	CreateCustomer(customer data.Customer) (data.Customer, *data.ErrorResponse)
	GetCustomer(id int) (data.Customer, *data.ErrorResponse)
	GetAllCustomers() []data.Customer
	UpdateCustomer(id int, customer data.Customer) (data.Customer, *data.ErrorResponse)
	DeleteCustomer(id int) *data.ErrorResponse
	SearchCustomers(criteria data.CustomerSearchCriteria) ([]data.Customer, *data.ErrorResponse)
}
