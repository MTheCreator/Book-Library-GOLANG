package InmemoryStores

import (
	"sync"
	"time"

	interfaces "finalProject/Interfaces"
	data "finalProject/StructureData"
	"finalProject/utils"
)

type InMemoryCustomerStore struct {
	mu        sync.RWMutex
	customers map[int]data.Customer
	nextID    int
}

var (
	customerStoreInstance *InMemoryCustomerStore
	customerOnce          sync.Once
)

// GetCustomerStoreInstance returns the singleton instance of InMemoryCustomerStore
func GetCustomerStoreInstance() interfaces.CustomerStore {
	customerOnce.Do(func() {
		customerStoreInstance = &InMemoryCustomerStore{
			customers: make(map[int]data.Customer),
			nextID:    1,
		}
	})
	return customerStoreInstance
}

// CreateCustomer adds a new customer to the store
func (store *InMemoryCustomerStore) CreateCustomer(customer data.Customer) (data.Customer, *data.ErrorResponse) {
	store.mu.Lock()
	defer store.mu.Unlock()

	// Set the creation time.
	customer.CreatedAt = time.Now()

	if customer.ID != 0 {
		// Use the provided ID (for example, when coming from PostgreSQL)
		if _, exists := store.customers[customer.ID]; exists {
			return data.Customer{}, &data.ErrorResponse{Message: "Customer ID already exists"}
		}
		store.customers[customer.ID] = customer
		if customer.ID >= store.nextID {
			store.nextID = customer.ID + 1
		}
	} else {
		// Otherwise assign a new ID.
		customer.ID = store.nextID
		store.nextID++
		store.customers[customer.ID] = customer
	}
	return customer, nil
}

// GetCustomer retrieves a customer by its ID
func (store *InMemoryCustomerStore) GetCustomer(id int) (data.Customer, *data.ErrorResponse) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	customer, exists := store.customers[id]
	if !exists {
		return data.Customer{}, &data.ErrorResponse{Message: "Customer not found"}
	}
	return customer, nil
}

// GetAllCustomers retrieves all customers
func (store *InMemoryCustomerStore) GetAllCustomers() []data.Customer {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var customers []data.Customer
	for _, customer := range store.customers {
		customers = append(customers, customer)
	}
	return customers
}

// UpdateCustomer updates the details of an existing customer
func (store *InMemoryCustomerStore) UpdateCustomer(id int, customer data.Customer) (data.Customer, *data.ErrorResponse) {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.customers[id]
	if !exists {
		return data.Customer{}, &data.ErrorResponse{Message: "Customer not found"}
	}
	customer.ID = id
	store.customers[id] = customer
	return customer, nil
}

// DeleteCustomer removes a customer from the store
func (store *InMemoryCustomerStore) DeleteCustomer(id int) *data.ErrorResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.customers[id]
	if !exists {
		return &data.ErrorResponse{Message: "Customer not found"}
	}
	delete(store.customers, id)
	return nil
}

// SearchCustomers filters customers based on the search criteria
func (store *InMemoryCustomerStore) SearchCustomers(criteria data.CustomerSearchCriteria) ([]data.Customer, *data.ErrorResponse) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var result []data.Customer
	for _, customer := range store.customers {
		if len(criteria.IDs) > 0 && !utils.ContainsInt(criteria.IDs, customer.ID) {
			continue
		}
		if len(criteria.Names) > 0 && !utils.ContainsString(criteria.Names, customer.Name) {
			continue
		}
		if len(criteria.Emails) > 0 && !utils.ContainsString(criteria.Emails, customer.Email) {
			continue
		}
		if !matchAddressCriteria(customer.Address, criteria.AddressCriteria) {
			continue
		}
		if !criteria.MinCreatedAt.IsZero() && customer.CreatedAt.Before(criteria.MinCreatedAt) {
			continue
		}
		if !criteria.MaxCreatedAt.IsZero() && customer.CreatedAt.After(criteria.MaxCreatedAt) {
			continue
		}
		result = append(result, customer)
	}
	return result, nil
}

// matchAddressCriteria matches a customer's address with the specified address search criteria
func matchAddressCriteria(address data.Address, criteria data.AddressSearchCriteria) bool {
	if len(criteria.Streets) > 0 && !utils.ContainsString(criteria.Streets, address.Street) {
		return false
	}
	if len(criteria.Cities) > 0 && !utils.ContainsString(criteria.Cities, address.City) {
		return false
	}
	if len(criteria.States) > 0 && !utils.ContainsString(criteria.States, address.State) {
		return false
	}
	if len(criteria.PostalCodes) > 0 && !utils.ContainsString(criteria.PostalCodes, address.PostalCode) {
		return false
	}
	if len(criteria.Countries) > 0 && !utils.ContainsString(criteria.Countries, address.Country) {
		return false
	}
	return true
}
