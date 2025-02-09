package Controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	inmemoryStores "finalProject/InmemoryStores"
	postgresStores "finalProject/postgresStores"
	"finalProject/StructureData"
)

// JSON file paths.
var (
	orderFile       = "orders.json"
	salesReportFile = "sales_reports.json"
)

func InitializeOrderFile() {
	// Get the PostgreSQL order store and check for existing orders.
	pgStore := postgresStores.GetPostgresOrderStoreInstance()
	existingOrders := pgStore.GetAllOrders()
	if len(existingOrders) > 0 {
		log.Println("Orders already exist in PostgreSQL; skipping JSON initialization for orders.")
		return
	}

	// Check if the orders JSON file exists; if not, create an empty one.
	if _, err := os.Stat(orderFile); os.IsNotExist(err) {
		file, err := os.Create(orderFile)
		if err != nil {
			log.Fatalf("Failed to create order file: %v", err)
		}
		file.Write([]byte("[]"))
		file.Close()
		log.Println("No order file existed; created empty orders JSON file.")
		return
	}

	// Open the JSON file for reading.
	file, err := os.Open(orderFile)
	if err != nil {
		log.Fatalf("Failed to open order file: %v", err)
	}
	defer file.Close()

	// Decode the orders from JSON.
	var orders []StructureData.Order
	if err := json.NewDecoder(file).Decode(&orders); err != nil {
		log.Fatalf("Failed to decode order file: %v", err)
	}
	log.Printf("Loaded %d orders from JSON.", len(orders))

	// Get the in-memory store (and related stores) to validate order data.
	memStore := inmemoryStores.GetOrderStoreInstance()
	customerStore := inmemoryStores.GetCustomerStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()

	var failedOrders []StructureData.Order

	// Iterate over the loaded orders.
	for _, order := range orders {
		// Validate the customer.
		customer, customerErr := customerStore.GetCustomer(order.Customer.ID)
		if customerErr != nil {
			log.Printf("Skipping order ID %d: Customer with ID %d not found", order.ID, order.Customer.ID)
			failedOrders = append(failedOrders, order)
			continue
		}
		order.Customer = customer

		// Validate each order item: check that each referenced book exists.
		validOrder := true
		for i, item := range order.Items {
			book, bookErr := bookStore.GetBook(item.Book.ID)
			if bookErr != nil {
				log.Printf("Skipping order ID %d: Book with ID %d not found", order.ID, item.Book.ID)
				validOrder = false
				break
			}
			// Update the item’s book info with the one from the store.
			order.Items[i].Book = book
		}
		if !validOrder {
			failedOrders = append(failedOrders, order)
			continue
		}

		// Create the order in the in-memory store (if needed) and then persist to PostgreSQL.
		createdOrder, errResp := memStore.CreateOrder(order)
		if errResp != nil {
			log.Printf("Error creating order in memory (order ID %d): %v", order.ID, errResp.Message)
			failedOrders = append(failedOrders, order)
			continue
		}

		// Now persist the order (and its items) to PostgreSQL.
		createdOrder, errResp = pgStore.CreateOrder(createdOrder)
		if errResp != nil {
			log.Printf("Error persisting order ID %d to PostgreSQL: %v", createdOrder.ID, errResp.Message)
			failedOrders = append(failedOrders, createdOrder)
			continue
		}

		log.Printf("Order ID %d loaded from JSON and inserted into PostgreSQL.", createdOrder.ID)
	}

	if len(failedOrders) > 0 {
		log.Println("Some orders failed to load. Persisting failed orders for debugging.")
		saveFailedOrders(failedOrders)
	} else {
		log.Println("All orders loaded successfully from JSON into PostgreSQL.")
	}
}

// Helper to persist failed orders.
func saveFailedOrders(orders []StructureData.Order) {
	file, err := os.Create("failed_orders.json")
	if err != nil {
		log.Printf("Failed to create failed_orders.json: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(orders); err != nil {
		log.Printf("Failed to persist failed orders: %v", err)
	}
}

// GetAllOrders handles GET /orders.
func GetAllOrders(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetOrderStoreInstance()
	orders := store.GetAllOrders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// GetOrderByID handles GET /orders/{id}.
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
	pgStore := postgresStores.GetPostgresOrderStoreInstance() // PostgreSQL store instance

	var order StructureData.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Default status to "pending" if not provided.
	if order.Status == "" {
		order.Status = StructureData.OrderStatusPending
	}

	// Validate customer.
	customer, errResp := customerStore.GetCustomer(order.Customer.ID)
	if errResp != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer does not exist"})
		return
	}
	order.Customer = customer

	// Validate books and adjust stock.
	validItems := []StructureData.OrderItem{}
	for _, item := range order.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr != nil {
			log.Printf("Skipping book ID %d: Does not exist", item.Book.ID)
			continue
		}
		if item.Quantity > book.Stock || book.Stock == 0 {
			log.Printf("Skipping book ID %d: Insufficient stock (stock=%d)", book.ID, book.Stock)
			continue
		}
		// Deduct stock.
		book.Stock -= item.Quantity
		_, updateErr := bookStore.UpdateBook(book.ID, book)
		if updateErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Failed to update book stock"})
			return
		}
		item.Book = book
		validItems = append(validItems, item)
	}
	if len(validItems) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "No valid books available to create the order"})
		return
	}
	order.Items = validItems
	order.CreatedAt = time.Now()

	// Create order in PostgreSQL first.
	createdPgOrder, errResp := pgStore.CreateOrder(order)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Now create the order in the in-memory store using the PostgreSQL order (with synchronized ID).
	createdOrder, errResp := orderStore.CreateOrder(createdPgOrder)
	if errResp != nil {
		// Optionally rollback PostgreSQL order if needed.
		pgStore.DeleteOrder(createdPgOrder.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist orders to JSON.
	if err := persistOrdersToFile(orderStore.GetAllOrders()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving order data to JSON"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdOrder)
}


// UpdateOrder handles PUT /orders/{id}.
func UpdateOrder(w http.ResponseWriter, r *http.Request) {
	orderStore := inmemoryStores.GetOrderStoreInstance()
	customerStore := inmemoryStores.GetCustomerStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()
	pgStore := postgresStores.GetPostgresOrderStoreInstance() // PostgreSQL store instance

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

	// Prevent updates on successful orders.
	if existingOrder.Status == StructureData.OrderStatusSuccess {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Successful orders cannot be updated"})
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
		for _, existingCustomer := range customerStore.GetAllCustomers() {
			if existingCustomer.Email == updatedOrder.Customer.Email {
				customer = existingCustomer
				break
			}
		}
		if customer.ID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer does not exist"})
			return
		}
	}
	updatedOrder.Customer = customer

	// Revert stock for the existing order.
	for _, item := range existingOrder.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr == nil {
			book.Stock += item.Quantity
			_, _ = bookStore.UpdateBook(book.ID, book)
		}
	}

	validItems := []StructureData.OrderItem{}
	for _, item := range updatedOrder.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr != nil {
			log.Printf("Skipping book ID %d: Does not exist", item.Book.ID)
			continue
		}
		if item.Quantity > book.Stock || book.Stock == 0 {
			log.Printf("Skipping book ID %d: Insufficient stock (stock=%d)", book.ID, book.Stock)
			continue
		}
		book.Stock -= item.Quantity
		_, updateErr := bookStore.UpdateBook(book.ID, book)
		if updateErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Failed to update book stock"})
			return
		}
		item.Book = book
		validItems = append(validItems, item)
	}
	if len(validItems) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "No valid books available to update the order"})
		return
	}
	updatedOrder.Items = validItems
	updatedOrder.CreatedAt = existingOrder.CreatedAt

	// Update the order.
	updatedOrder, errResp = orderStore.UpdateOrder(id, updatedOrder)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if err := persistOrdersToFile(orderStore.GetAllOrders()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving order data to JSON"})
		return
	}

	_, pgErr := pgStore.UpdateOrder(id, updatedOrder)
	if pgErr != nil {
		log.Printf("Error updating order in PostgreSQL: %v", pgErr.Message)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedOrder)
}

// DeleteOrder handles DELETE /orders/{id}.
func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	orderStore := inmemoryStores.GetOrderStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()
	pgStore := postgresStores.GetPostgresOrderStoreInstance() // PostgreSQL store instance

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

	// Prevent deletion of successful orders.
	if order.Status == StructureData.OrderStatusSuccess {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Successful orders cannot be deleted"})
		return
	}

	// Restore book stock.
	for _, item := range order.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr != nil {
			log.Printf("Warning: Book with ID %d not found while deleting order %d", item.Book.ID, id)
			continue
		}
		book.Stock += item.Quantity
		_, updateErr := bookStore.UpdateBook(book.ID, book)
		if updateErr != nil {
			log.Printf("Error updating stock for book ID %d: %s", book.ID, updateErr.Message)
			continue
		}
	}

	errResp = orderStore.DeleteOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if err := persistOrdersToFile(orderStore.GetAllOrders()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving order data to JSON"})
		return
	}

	pgErr := pgStore.DeleteOrder(id)
	if pgErr != nil {
		log.Printf("Error deleting order from PostgreSQL: %v", pgErr.Message)
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchOrders handles POST /orders/search.
func SearchOrders(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetOrderStoreInstance()
	var criteria StructureData.OrderSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
		return
	}
	searchResults, errResp := store.SearchOrders(criteria)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResults)
}

func persistOrdersToFile(orders []StructureData.Order) error {
	file, err := os.Create(orderFile)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(orders)
}

func GenerateSalesReport(ctx context.Context) {
	store := inmemoryStores.GetOrderStoreInstance()
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)
	orders, err := store.GetOrdersInTimeRange(startTime, endTime)
	if err != nil {
		log.Printf("Error fetching orders for sales report: %v\n", err)
		return
	}
	if len(orders) == 0 {
		log.Println("No orders found for the sales report generation.")
		report := StructureData.SalesReport{
			Timestamp:       endTime,
			TotalRevenue:    0,
			TotalOrders:     0,
			TopSellingBooks: []StructureData.TopSellingBook{},
		}
		if err := SaveSalesReport(ctx, report); err != nil {
			log.Printf("Error saving empty sales report: %v\n", err)
		}
		return
	}
	var totalRevenue float64
	var totalOrders int
	bookSales := make(map[int]*StructureData.TopSellingBook)
	bookStore := inmemoryStores.GetBookStoreInstance()
	for _, order := range orders {
		select {
		case <-ctx.Done():
			log.Println("GenerateSalesReport was canceled.")
			return
		default:
		}
		totalRevenue += order.TotalPrice
		totalOrders++
		for _, item := range order.Items {
			select {
			case <-ctx.Done():
				log.Println("GenerateSalesReport was canceled.")
				return
			default:
			}
			book, bookErr := bookStore.GetBook(item.Book.ID)
			if bookErr != nil {
				log.Printf("Skipping order ID %d: Book ID %d not found", order.ID, item.Book.ID)
				continue
			}
			if _, exists := bookSales[book.ID]; !exists {
				bookSales[book.ID] = &StructureData.TopSellingBook{
					Book:         book,
					QuantitySold: 0,
				}
			}
			bookSales[book.ID].QuantitySold += item.Quantity
		}
	}
	topSellingBooks := make([]StructureData.TopSellingBook, 0, len(bookSales))
	for _, bookSale := range bookSales {
		topSellingBooks = append(topSellingBooks, *bookSale)
	}
	sort.Slice(topSellingBooks, func(i, j int) bool {
		revenueI := topSellingBooks[i].Book.Price * float64(topSellingBooks[i].QuantitySold)
		revenueJ := topSellingBooks[j].Book.Price * float64(topSellingBooks[j].QuantitySold)
		return revenueI > revenueJ
	})
	if len(topSellingBooks) > 5 {
		topSellingBooks = topSellingBooks[:5]
	}
	report := StructureData.SalesReport{
		Timestamp:       endTime,
		TotalRevenue:    totalRevenue,
		TotalOrders:     totalOrders,
		TopSellingBooks: topSellingBooks,
	}
	if err := SaveSalesReport(ctx, report); err != nil {
		log.Printf("Error saving sales report: %v\n", err)
	}
}

func SaveSalesReport(ctx context.Context, report StructureData.SalesReport) error {
	var reports []StructureData.SalesReport
	if _, err := os.Stat(salesReportFile); !os.IsNotExist(err) {
		file, err := os.Open(salesReportFile)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&reports); err != nil {
			return err
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	reports = append(reports, report)
	file, err := os.Create(salesReportFile)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := encoder.Encode(reports); err != nil {
		return err
	}
	log.Println("Sales report saved successfully.")
	return nil
}

func GetSalesReport(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var reports []StructureData.SalesReport
	if _, err := os.Stat(salesReportFile); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "No sales reports available"})
		return
	}
	file, err := os.Open(salesReportFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error loading sales reports file"})
		return
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&reports); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error decoding sales reports data"})
		return
	}
	select {
	case <-ctx.Done():
		w.WriteHeader(http.StatusRequestTimeout)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Request canceled by client"})
		return
	default:
	}
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	var filteredReports []StructureData.SalesReport
	if startDateStr != "" && endDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid start_date format. Use YYYY-MM-DD."})
			return
		}
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid end_date format. Use YYYY-MM-DD."})
			return
		}
		for _, report := range reports {
			if report.Timestamp.After(startDate) && report.Timestamp.Before(endDate.Add(24*time.Hour)) {
				filteredReports = append(filteredReports, report)
			}
		}
	} else {
		filteredReports = reports
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(filteredReports); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error encoding response data"})
		return
	}
}
