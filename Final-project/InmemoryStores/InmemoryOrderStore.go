package InmemoryStores

import (
	"sync"
	"time"

	data "finalProject/StructureData"
	"finalProject/utils"
)

type InMemoryOrderStore struct {
	mu     sync.RWMutex
	orders map[int]data.Order
	nextID int
}

var (
	orderStoreInstance *InMemoryOrderStore
	orderOnce          sync.Once
)

// GetOrderStoreInstance returns the singleton instance of InMemoryOrderStore
func GetOrderStoreInstance() *InMemoryOrderStore {
	orderOnce.Do(func() {
		orderStoreInstance = &InMemoryOrderStore{
			orders: make(map[int]data.Order),
			nextID: 1,
		}
	})
	return orderStoreInstance
}

// CreateOrder adds a new order to the store
// CreateOrder adds a new order to the store
func (store *InMemoryOrderStore) CreateOrder(order data.Order) (data.Order, *data.ErrorResponse) {
    store.mu.Lock()
    defer store.mu.Unlock()

    // Calculate the total price of the order
    totalPrice := 0.0
    for i, item := range order.Items {
        // Ensure the book exists and fetch its details
        bookStore := GetBookStoreInstance()
        book, err := bookStore.GetBook(item.Book.ID)
        if err != nil {
            return data.Order{}, &data.ErrorResponse{Message: "Book not found for item in order"}
        }
        // Update the book details in the order
        order.Items[i].Book = book

        // Calculate price * quantity and add to total
        totalPrice += book.Price * float64(item.Quantity)
    }

    order.TotalPrice = totalPrice // Set the calculated total price
    order.ID = store.nextID
    order.CreatedAt = time.Now()
    store.nextID++
    store.orders[order.ID] = order
    return order, nil
}


// GetOrder retrieves an order by its ID
func (store *InMemoryOrderStore) GetOrder(id int) (data.Order, *data.ErrorResponse) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	order, exists := store.orders[id]
	if !exists {
		return data.Order{}, &data.ErrorResponse{Message: "Order not found"}
	}
	return order, nil
}

// UpdateOrder updates the details of an existing order
// UpdateOrder updates the details of an existing order
func (store *InMemoryOrderStore) UpdateOrder(id int, order data.Order) (data.Order, *data.ErrorResponse) {
    store.mu.Lock()
    defer store.mu.Unlock()

    _, exists := store.orders[id]
    if !exists {
        return data.Order{}, &data.ErrorResponse{Message: "Order not found"}
    }

    // Calculate the total price of the order
    totalPrice := 0.0
    for i, item := range order.Items {
        // Ensure the book exists and fetch its details
        bookStore := GetBookStoreInstance()
        book, err := bookStore.GetBook(item.Book.ID)
        if err != nil {
            return data.Order{}, &data.ErrorResponse{Message: "Book not found for item in order"}
        }
        // Update the book details in the order
        order.Items[i].Book = book

        // Calculate price * quantity and add to total
        totalPrice += book.Price * float64(item.Quantity)
    }

    order.TotalPrice = totalPrice // Set the calculated total price
    order.ID = id
    store.orders[id] = order
    return order, nil
}


// DeleteOrder removes an order from the store
func (store *InMemoryOrderStore) DeleteOrder(id int) *data.ErrorResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.orders[id]
	if !exists {
		return &data.ErrorResponse{Message: "Order not found"}
	}
	delete(store.orders, id)
	return nil
}

// GetAllOrders retrieves all orders from the store
func (store *InMemoryOrderStore) GetAllOrders() []data.Order {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var orders []data.Order
	for _, order := range store.orders {
		orders = append(orders, order)
	}
	return orders
}

// SearchOrders filters orders based on the search criteria
func (store *InMemoryOrderStore) SearchOrders(criteria data.OrderSearchCriteria) ([]data.Order, *data.ErrorResponse) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var result []data.Order
	for _, order := range store.orders {
		if len(criteria.IDs) > 0 && !utils.ContainsInt(criteria.IDs, order.ID) {
			continue
		}
		if len(criteria.CustomerIDs) > 0 && !utils.ContainsInt(criteria.CustomerIDs, order.Customer.ID) {
			continue
		}
		if criteria.MinTotalPrice > 0 && order.TotalPrice < criteria.MinTotalPrice {
			continue
		}
		if criteria.MaxTotalPrice > 0 && order.TotalPrice > criteria.MaxTotalPrice {
			continue
		}
		if !criteria.MinCreatedAt.IsZero() && order.CreatedAt.Before(criteria.MinCreatedAt) {
			continue
		}
		if !criteria.MaxCreatedAt.IsZero() && order.CreatedAt.After(criteria.MaxCreatedAt) {
			continue
		}
		
		if !matchOrderItems(order.Items, criteria.ItemCriteria) {
			continue
		}
		result = append(result, order)
	}

	return result, nil
}

// matchOrderItems filters order items based on the criteria
func matchOrderItems(items []data.OrderItem, criteria data.OrderItemSearchCriteria) bool {
	for _, item := range items {
		if criteria.MinQuantity > 0 && item.Quantity < criteria.MinQuantity {
			continue
		}
		if criteria.MaxQuantity > 0 && item.Quantity > criteria.MaxQuantity {
			continue
		}
		if !matchBookCriteria(item.Book, criteria.BookCriteria) {
			continue
		}
		return true
	}
	return false
}

// matchBookCriteria filters books based on the criteria
func matchBookCriteria(book data.Book, criteria data.BookSearchCriteria) bool {
	if len(criteria.IDs) > 0 && !utils.ContainsInt(criteria.IDs, book.ID) {
		return false
	}
	if len(criteria.Titles) > 0 && !utils.ContainsString(criteria.Titles, book.Title) {
		return false
	}
	if len(criteria.Genres) > 0 && !utils.ContainsAnyString(criteria.Genres, book.Genres) {
		return false
	}
	if !criteria.MinPublishedAt.IsZero() && book.PublishedAt.Before(criteria.MinPublishedAt) {
		return false
	}
	if !criteria.MaxPublishedAt.IsZero() && book.PublishedAt.After(criteria.MaxPublishedAt) {
		return false
	}
	if criteria.MinPrice > 0 && book.Price < criteria.MinPrice {
		return false
	}
	if criteria.MaxPrice > 0 && book.Price > criteria.MaxPrice {
		return false
	}
	if !utils.MatchAuthorCriteria(book.Author, criteria.AuthorCriteria) {
		return false
	}
	return true
}
func (store *InMemoryOrderStore) GetOrdersInTimeRange(start, end time.Time) ([]data.Order, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var filteredOrders []data.Order
	for _, order := range store.orders {
		if order.CreatedAt.After(start) && order.CreatedAt.Before(end) {
			filteredOrders = append(filteredOrders, order)
		}
	}
	return filteredOrders, nil
}
