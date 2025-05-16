package postgresStores

import (
	"database/sql"
	"finalProject/StructureData"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// PostgresOrderStore implements the OrderStore interface using PostgreSQL.
type PostgresOrderStore struct {
	db *sql.DB
}

// Close gracefully closes the underlying DB connection.
func (store *PostgresOrderStore) Close() error {
	return store.db.Close()
}

var postgresOrderStoreInstance *PostgresOrderStore

func GetPostgresOrderStoreInstance() *PostgresOrderStore {
    if postgresOrderStoreInstance == nil {
        host := getEnv("DB_HOST", "db")
        port := getEnv("DB_PORT", "5432")
        user := getEnv("DB_USER", "postgres")
        password := getEnv("DB_PASSWORD", "root")
        dbname := getEnv("DB_NAME", "booklibrary")

        connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
            host, port, user, password, dbname)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		postgresOrderStoreInstance = &PostgresOrderStore{db: db}
	}
	return postgresOrderStoreInstance
}
// Helper function to get environment variables with defaults
func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback}

// CreateOrder inserts a new order and its items into the database.
func (store *PostgresOrderStore) CreateOrder(order StructureData.Order) (StructureData.Order, *StructureData.ErrorResponse) {
	tx, err := store.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: "Failed to begin transaction"}
	}
	defer tx.Rollback()

	var queryOrder string
	var args []interface{}
	if order.ID != 0 {
		queryOrder = `INSERT INTO orders (id, customer_id, total_price, created_at, status)
		              VALUES ($1, $2, $3, $4, $5) RETURNING id`
		args = []interface{}{
			order.ID,
			order.Customer.ID,
			order.TotalPrice,
			order.CreatedAt,
			order.Status,
		}
	} else {
		queryOrder = `INSERT INTO orders (customer_id, total_price, created_at, status)
		              VALUES ($1, $2, $3, $4) RETURNING id`
		args = []interface{}{
			order.Customer.ID,
			order.TotalPrice,
			order.CreatedAt,
			order.Status,
		}
	}
	err = tx.QueryRow(queryOrder, args...).Scan(&order.ID)
	if err != nil {
		log.Printf("Error inserting order (customer_id=%d): %v", order.Customer.ID, err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to insert order: %v", err)}
	}
	log.Printf("Inserted order with ID %d", order.ID)

	// Insert each order item.
	for _, item := range order.Items {
		queryItem := `INSERT INTO order_items (order_id, book_id, quantity) VALUES ($1, $2, $3)`
		_, err = tx.Exec(queryItem, order.ID, item.Book.ID, item.Quantity)
		if err != nil {
			log.Printf("Error inserting order item for order ID %d, book ID %d: %v", order.ID, item.Book.ID, err)
			return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to insert order item: %v", err)}
		}
		log.Printf("Inserted order item for order ID %d, book ID %d, quantity %d", order.ID, item.Book.ID, item.Quantity)
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction for order ID %d: %v", order.ID, err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to commit transaction: %v", err)}
	}
	log.Printf("Order ID %d committed successfully", order.ID)
	return order, nil
}

// (Other order methods remain unchanged.)

// GetOrder retrieves an order (including its items) by ID.
func (store *PostgresOrderStore) GetOrder(id int) (StructureData.Order, *StructureData.ErrorResponse) {
	var order StructureData.Order
	queryOrder := `SELECT id, customer_id, total_price, created_at, status FROM orders WHERE id=$1`
	row := store.db.QueryRow(queryOrder, id)
	err := row.Scan(&order.ID, &order.Customer.ID, &order.TotalPrice, &order.CreatedAt, &order.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Order with ID %d not found", id)
			return StructureData.Order{}, &StructureData.ErrorResponse{Message: "Order not found"}
		}
		log.Printf("Error fetching order with ID %d: %v", id, err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Error fetching order: %v", err)}
	}

	// Fetch order items.
	queryItems := `SELECT book_id, quantity FROM order_items WHERE order_id=$1`
	rows, err := store.db.Query(queryItems, order.ID)
	if err != nil {
		log.Printf("Error fetching order items for order ID %d: %v", order.ID, err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Error fetching order items: %v", err)}
	}
	defer rows.Close()
	for rows.Next() {
		var item StructureData.OrderItem
		err = rows.Scan(&item.Book.ID, &item.Quantity)
		if err != nil {
			log.Printf("Error scanning order item for order ID %d: %v", order.ID, err)
			return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Error scanning order item: %v", err)}
		}
		order.Items = append(order.Items, item)
	}
	log.Printf("Retrieved order ID %d with %d items", order.ID, len(order.Items))
	return order, nil
}

// UpdateOrder updates an existing order and its items.
func (store *PostgresOrderStore) UpdateOrder(id int, order StructureData.Order) (StructureData.Order, *StructureData.ErrorResponse) {
	tx, err := store.db.Begin()
	if err != nil {
		log.Printf("Failed to begin update transaction: %v", err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: "Failed to begin transaction"}
	}
	defer tx.Rollback()

	// Update order header.
	queryUpdate := `UPDATE orders SET customer_id=$1, total_price=$2, created_at=$3, status=$4 WHERE id=$5`
	_, err = tx.Exec(queryUpdate, order.Customer.ID, order.TotalPrice, order.CreatedAt, order.Status, id)
	if err != nil {
		log.Printf("Failed to update order ID %d: %v", id, err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to update order: %v", err)}
	}
	log.Printf("Updated order header for order ID %d", id)

	// Delete existing order items.
	queryDeleteItems := `DELETE FROM order_items WHERE order_id=$1`
	_, err = tx.Exec(queryDeleteItems, id)
	if err != nil {
		log.Printf("Failed to delete old order items for order ID %d: %v", id, err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to delete old order items: %v", err)}
	}

	// Insert new order items.
	for _, item := range order.Items {
		queryInsertItem := `INSERT INTO order_items (order_id, book_id, quantity) VALUES ($1, $2, $3)`
		_, err = tx.Exec(queryInsertItem, id, item.Book.ID, item.Quantity)
		if err != nil {
			log.Printf("Failed to insert order item for order ID %d, book ID %d: %v", id, item.Book.ID, err)
			return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to insert order item: %v", err)}
		}
		log.Printf("Inserted order item for order ID %d, book ID %d, quantity %d", id, item.Book.ID, item.Quantity)
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit update transaction for order ID %d: %v", id, err)
		return StructureData.Order{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to commit transaction: %v", err)}
	}
	order.ID = id
	log.Printf("Order ID %d updated successfully", id)
	return order, nil
}

// DeleteOrder removes an order and its items from the database.
func (store *PostgresOrderStore) DeleteOrder(id int) *StructureData.ErrorResponse {
	tx, err := store.db.Begin()
	if err != nil {
		log.Printf("Failed to begin delete transaction: %v", err)
		return &StructureData.ErrorResponse{Message: "Failed to begin transaction"}
	}
	defer tx.Rollback()

	// Delete order items.
	queryDeleteItems := `DELETE FROM order_items WHERE order_id=$1`
	_, err = tx.Exec(queryDeleteItems, id)
	if err != nil {
		log.Printf("Failed to delete order items for order ID %d: %v", id, err)
		return &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to delete order items: %v", err)}
	}

	// Delete order header.
	queryDeleteOrder := `DELETE FROM orders WHERE id=$1`
	res, err := tx.Exec(queryDeleteOrder, id)
	if err != nil {
		log.Printf("Failed to delete order ID %d: %v", id, err)
		return &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to delete order: %v", err)}
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("Order ID %d not found for deletion", id)
		return &StructureData.ErrorResponse{Message: "Order not found"}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit delete transaction for order ID %d: %v", id, err)
		return &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to commit transaction: %v", err)}
	}
	log.Printf("Deleted order ID %d and its items", id)
	return nil
}

// GetAllOrders retrieves all orders (and their items) from the database.
func (store *PostgresOrderStore) GetAllOrders() []StructureData.Order {
	orders := []StructureData.Order{}
	query := `SELECT id, customer_id, total_price, created_at, status FROM orders`
	rows, err := store.db.Query(query)
	if err != nil {
		log.Printf("Error querying orders: %v", err)
		return orders
	}
	defer rows.Close()

	for rows.Next() {
		var order StructureData.Order
		err = rows.Scan(&order.ID, &order.Customer.ID, &order.TotalPrice, &order.CreatedAt, &order.Status)
		if err != nil {
			log.Printf("Error scanning order: %v", err)
			continue
		}

		// Join order_items with books to get full book details for each item.
		itemQuery := `
			SELECT oi.book_id, oi.quantity, b.title, b.price, b.stock
			FROM order_items oi
			LEFT JOIN books b ON oi.book_id = b.id
			WHERE oi.order_id = $1`
		itemRows, err := store.db.Query(itemQuery, order.ID)
		if err != nil {
			log.Printf("Error querying order items for order ID %d: %v", order.ID, err)
		} else {
			for itemRows.Next() {
				var item StructureData.OrderItem
				// Scan additional book details (title, price, stock) into the Book sub-struct.
				err = itemRows.Scan(&item.Book.ID, &item.Quantity, &item.Book.Title, &item.Book.Price, &item.Book.Stock)
				if err != nil {
					log.Printf("Error scanning order item for order ID %d: %v", order.ID, err)
					continue
				}
				order.Items = append(order.Items, item)
			}
			itemRows.Close()
		}
		orders = append(orders, order)
	}
	log.Printf("Retrieved %d orders", len(orders))
	return orders
}


// SearchOrders filters orders based on the provided criteria.
func (store *PostgresOrderStore) SearchOrders(criteria StructureData.OrderSearchCriteria) ([]StructureData.Order, *StructureData.ErrorResponse) {
	allOrders := store.GetAllOrders()
	filteredOrders := []StructureData.Order{}
	for _, order := range allOrders {
		// Filter by order IDs.
		if len(criteria.IDs) > 0 {
			found := false
			for _, id := range criteria.IDs {
				if order.ID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// Filter by customer IDs.
		if len(criteria.CustomerIDs) > 0 {
			found := false
			for _, cid := range criteria.CustomerIDs {
				if order.Customer.ID == cid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// Filter by total price.
		if criteria.MinTotalPrice > 0 && order.TotalPrice < criteria.MinTotalPrice {
			continue
		}
		if criteria.MaxTotalPrice > 0 && order.TotalPrice > criteria.MaxTotalPrice {
			continue
		}
		// Filter by creation time.
		if !criteria.MinCreatedAt.IsZero() && order.CreatedAt.Before(criteria.MinCreatedAt) {
			continue
		}
		if !criteria.MaxCreatedAt.IsZero() && order.CreatedAt.After(criteria.MaxCreatedAt) {
			continue
		}
		// Filter by status.
		if criteria.Status != "" && order.Status != criteria.Status {
			continue
		}
		// Filter by order items criteria.
		if !matchOrderItems(order.Items, criteria.ItemCriteria) {
			continue
		}
		filteredOrders = append(filteredOrders, order)
	}
	log.Printf("Search returned %d orders", len(filteredOrders))
	return filteredOrders, nil
}

// Helper function to match order items based on search criteria.
func matchOrderItems(items []StructureData.OrderItem, criteria StructureData.OrderItemSearchCriteria) bool {
	// If no criteria are provided, consider it a match.
	if criteria.MinQuantity == 0 && criteria.MaxQuantity == 0 {
		return true
	}
	for _, item := range items {
		if criteria.MinQuantity > 0 && item.Quantity < criteria.MinQuantity {
			continue
		}
		if criteria.MaxQuantity > 0 && item.Quantity > criteria.MaxQuantity {
			continue
		}
		return true
	}
	return false
}

// GetOrdersInTimeRange retrieves orders created within a specified time range.
func (store *PostgresOrderStore) GetOrdersInTimeRange(start, end time.Time) ([]StructureData.Order, error) {
	orders := []StructureData.Order{}
	query := `SELECT id, customer_id, total_price, created_at, status FROM orders WHERE created_at >= $1 AND created_at <= $2`
	rows, err := store.db.Query(query, start, end)
	if err != nil {
		log.Printf("Error querying orders in time range: %v", err)
		return orders, err
	}
	defer rows.Close()
	for rows.Next() {
		var order StructureData.Order
		err = rows.Scan(&order.ID, &order.Customer.ID, &order.TotalPrice, &order.CreatedAt, &order.Status)
		if err != nil {
			log.Printf("Error scanning order in time range: %v", err)
			continue
		}
		orders = append(orders, order)
	}
	log.Printf("Retrieved %d orders in the specified time range", len(orders))
	return orders, nil
}
