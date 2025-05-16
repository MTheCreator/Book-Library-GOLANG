package Controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	inmemoryStores "finalProject/InmemoryStores"
	"finalProject/StructureData"
	postgresStores "finalProject/postgresStores"
)

func InitializeOrderFile() {
	pgStore := postgresStores.GetPostgresOrderStoreInstance()
	memStore := inmemoryStores.GetOrderStoreInstance()

	// Load PostgreSQL orders into memory
	pgOrders := pgStore.GetAllOrders()

	// Only initialize if memory store is empty
	if len(memStore.GetAllOrders()) == 0 {
		for _, order := range pgOrders {
			_, err := memStore.CreateOrder(order)
			if err != nil {
				log.Printf("Error loading order %d into memory: %v", order.ID, err.Message)
			}
		}
		log.Printf("Loaded %d orders from PostgreSQL into memory", len(pgOrders))
	}
}

func GetAllOrders(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetOrderStoreInstance()
	orders := store.GetAllOrders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func GetOrderByID(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetOrderStoreInstance()
	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid order ID"})
		return
	}
	order, errResp := store.GetOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}


func CreateOrder(w http.ResponseWriter, r *http.Request) {
	orderStore := inmemoryStores.GetOrderStoreInstance()
	customerStore := inmemoryStores.GetCustomerStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()
	pgStore := postgresStores.GetPostgresOrderStoreInstance()
	pgBookStore := postgresStores.GetPostgresBookStoreInstance()

	var order StructureData.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	if order.Status == "" {
		order.Status = StructureData.OrderStatusPending
	}

	// Validate customer exists in memory.
	customer, errResp := customerStore.GetCustomer(order.Customer.ID)
	if errResp != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer does not exist"})
		return
	}
	order.Customer = customer

	validItems := []StructureData.OrderItem{}
	for _, item := range order.Items {
		// Attempt to fetch the book from the in-memory store.
		book, bookErr := bookStore.GetBook(item.Book.ID)
		log.Println(book)
		// If not found in memory, try PostgreSQL.
		if bookErr != nil {
			pgBook, pgErrResp := pgBookStore.GetBook(item.Book.ID)
			log.Println(pgBook)
			if pgErrResp != nil {
				// If not found in PostgreSQL either, skip this item.
				continue
			}
			// Add the book from PostgreSQL into the in-memory store.
			_, memErr := bookStore.CreateBook(pgBook)
			if memErr != nil {
				continue
			}
			book = pgBook
		}
		// Check if the requested quantity is available.
		if item.Quantity > book.Stock || book.Stock == 0 {
			continue
		}
		// Deduct stock.
		book.Stock -= item.Quantity

		// Update the book in both stores.
		_, updateErr := bookStore.UpdateBook(book.ID, book)
		if updateErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Failed to update book stock in memory"})
			return
		}
		if _, pgUpdateErr := pgBookStore.UpdateBook(book.ID, book); pgUpdateErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Failed to update book stock in PostgreSQL"})
			return
		}
		item.Book = book
		validItems = append(validItems, item)
	}
	if len(validItems) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "No valid books available"})
		return
	}
	order.Items = validItems
	order.CreatedAt = time.Now()

	createdPgOrder, errResp := pgStore.CreateOrder(order)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	createdOrder, errResp := orderStore.CreateOrder(createdPgOrder)
	if errResp != nil {
		pgStore.DeleteOrder(createdPgOrder.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdOrder)
}


func UpdateOrder(w http.ResponseWriter, r *http.Request) {
	orderStore := inmemoryStores.GetOrderStoreInstance()
	customerStore := inmemoryStores.GetCustomerStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()
	pgStore := postgresStores.GetPostgresOrderStoreInstance()
	pgBookStore := postgresStores.GetPostgresBookStoreInstance()

	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid order ID"})
		return
	}

	existingOrder, errResp := orderStore.GetOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if existingOrder.Status == StructureData.OrderStatusSuccess {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Cannot update successful orders"})
		return
	}

	var updatedOrder StructureData.Order
	if err := json.NewDecoder(r.Body).Decode(&updatedOrder); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	customer, errResp := customerStore.GetCustomer(updatedOrder.Customer.ID)
	if errResp != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer does not exist"})
		return
	}
	updatedOrder.Customer = customer

	// Revert stock for existing order items.
	for _, item := range existingOrder.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr == nil {
			book.Stock += item.Quantity
			// Update in-memory.
			_, _ = bookStore.UpdateBook(book.ID, book)
			// Also update in PostgreSQL.
			_, _ = pgBookStore.UpdateBook(book.ID, book)
		}
	}

	validItems := []StructureData.OrderItem{}
	for _, item := range updatedOrder.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr != nil {
			continue
		}
		if item.Quantity > book.Stock || book.Stock == 0 {
			continue
		}
		book.Stock -= item.Quantity
		// Update in-memory.
		_, updateErr := bookStore.UpdateBook(book.ID, book)
		if updateErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Failed to update book stock in memory"})
			return
		}
		// Update in PostgreSQL.
		if _, pgUpdateErr := pgBookStore.UpdateBook(book.ID, book); pgUpdateErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Failed to update book stock in PostgreSQL"})
			return
		}
		item.Book = book
		validItems = append(validItems, item)
	}
	if len(validItems) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "No valid books available"})
		return
	}
	updatedOrder.Items = validItems
	updatedOrder.CreatedAt = existingOrder.CreatedAt

	updatedOrder, errResp = orderStore.UpdateOrder(id, updatedOrder)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	_, pgErr := pgStore.UpdateOrder(id, updatedOrder)
	if pgErr != nil {
		log.Printf("Error updating order in PostgreSQL: %v", pgErr.Message)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedOrder)
}

func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	orderStore := inmemoryStores.GetOrderStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()
	pgStore := postgresStores.GetPostgresOrderStoreInstance()
	pgBookStore := postgresStores.GetPostgresBookStoreInstance()

	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid order ID"})
		return
	}

	order, errResp := orderStore.GetOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if order.Status == StructureData.OrderStatusSuccess {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Cannot delete successful orders"})
		return
	}

	for _, item := range order.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr != nil {
			continue
		}
		book.Stock += item.Quantity
		_, _ = bookStore.UpdateBook(book.ID, book)
		_, _ = pgBookStore.UpdateBook(book.ID, book)
	}

	errResp = orderStore.DeleteOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	pgErr := pgStore.DeleteOrder(id)
	if pgErr != nil {
		log.Printf("Error deleting order from PostgreSQL: %v", pgErr.Message)
	}

	w.WriteHeader(http.StatusNoContent)
}

func SearchOrders(w http.ResponseWriter, r *http.Request) {
    pgStore := postgresStores.GetPostgresOrderStoreInstance() // Use PostgreSQL
    var criteria StructureData.OrderSearchCriteria
    if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
        return
    }
    searchResults, errResp := pgStore.SearchOrders(criteria) // Query PostgreSQL
    if errResp != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(errResp)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(searchResults)
}

func GenerateSalesReport(ctx context.Context) {
	// Use the PostgreSQL order and sales report stores.
	orderStore := postgresStores.GetPostgresOrderStoreInstance()
	reportStore := postgresStores.GetPostgresSalesReportStoreInstance()

	// Use local time for the report window.
	nowLocal := time.Now() // local time
	endTime := nowLocal
	startTime := endTime.Add(-24 * time.Hour)

	var report StructureData.SalesReport
	report.Timestamp = endTime
	// Initialize a map to accumulate revenue and quantity per book.
	bookRevenueMap := make(map[int]*StructureData.TopSellingBook)

	orders := orderStore.GetAllOrders()
	
	log.Printf("[Report] Local Time Range: %s to %s",
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
	)

	for _, order := range orders {
		// Convert order.CreatedAt to local time.
		orderTimeLocal := order.CreatedAt.Local().Add(-1 * time.Hour)

		log.Printf("[Order %d] Created (Local): %s | Status: %s | Total: $%.2f",
			order.ID,
			orderTimeLocal.Format(time.RFC3339),
			order.Status,
			order.TotalPrice,
		)

		// Check if order is within the local time window.
		if orderTimeLocal.Before(startTime) || orderTimeLocal.After(endTime) {
			log.Printf("[Excluded] Order %d - Reason: %s",
				order.ID,
				getExclusionReason(orderTimeLocal, startTime, endTime),
			)
			continue
		}

		log.Printf("[Included] Order %d - Within time window", order.ID)
		report.TotalOrders++
		report.TotalRevenue += order.TotalPrice

		// Count order status if needed.
		switch order.Status {
		case StructureData.OrderStatusSuccess:
			report.SuccessfulOrders++
		case StructureData.OrderStatusPending:
			report.PendingOrders++
		}
		log.Println(order.Items)
		// Process each order item.
		for _, item := range order.Items {
			if item.Book.ID == 0 {
				log.Printf("[Warning] Invalid book in order %d", order.ID)
				continue
			}

			// Calculate revenue for this item.
			revenue := item.Book.Price * float64(item.Quantity)
			// If we've already seen this book, update its totals.
			if existing, exists := bookRevenueMap[item.Book.ID]; exists {
				existing.QuantitySold += item.Quantity
				existing.TotalRevenue += revenue
			} else {
				// Otherwise, create a new TopSellingBook entry.
				bookRevenueMap[item.Book.ID] = &StructureData.TopSellingBook{
					Book: item.Book,
					QuantitySold: item.Quantity,
					TotalRevenue: revenue,
				}
			}
		}
	}

	// Convert the map to a slice.
	var topSellers []StructureData.TopSellingBook
	for _, tsb := range bookRevenueMap {
		topSellers = append(topSellers, *tsb)
	}

	// Sort the top sellers by descending total revenue.
	sort.Slice(topSellers, func(i, j int) bool {
		return topSellers[i].TotalRevenue > topSellers[j].TotalRevenue
	})

	// Limit the list to at most five top-selling books.
	if len(topSellers) > 5 {
		report.TopSellingBooks = topSellers[:5]
	} else {
		report.TopSellingBooks = topSellers
	}

	log.Printf("[Report] Final Result: (Timestamp=%s, TotalRevenue=%.2f, TotalOrders=%d, PendingOrders=%d, SuccessfulOrders=%d, TopSellingBooks=%+v)",
		report.Timestamp.Format(time.RFC3339),
		report.TotalRevenue,
		report.TotalOrders,
		report.PendingOrders,
		report.SuccessfulOrders,
		report.TopSellingBooks,
	)

	// Save the report to PostgreSQL.
	if _, err := reportStore.SaveSalesReport(report); err != nil {
		log.Printf("[Error] Failed to save sales report: %v", err)
	}
}

func getExclusionReason(orderTime, startTime, endTime time.Time) string {
	if orderTime.Before(startTime) {
		return fmt.Sprintf("Too old (Order: %s < Start: %s)",
			orderTime.Format(time.RFC3339),
			startTime.Format(time.RFC3339),
		)
	}
	return fmt.Sprintf("Too new (Order: %s > End: %s)",
		orderTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
	)
}


// GetSalesReport handles GET /reports/sales by retrieving sales reports from PostgreSQL.
func GetSalesReport(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	reportStore := postgresStores.GetPostgresSalesReportStoreInstance()

	reports, err := reportStore.GetAllSalesReports()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{
			Message: "Failed to retrieve sales reports",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(reports)
}
