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
	"finalProject/StructureData"
)

// JSON file path for order persistence
var (
	orderFile       = "orders.json"
	salesReportFile = "sales_reports.json"
)

func InitializeOrderFile() {
	if _, err := os.Stat(orderFile); os.IsNotExist(err) {
		file, _ := os.Create(orderFile)
		file.Write([]byte("[]"))
		file.Close()
	} else {
		file, err := os.Open(orderFile)
		if err != nil {
			panic("Failed to open order file")
		}
		defer file.Close()

		var orders []StructureData.Order
		if err := json.NewDecoder(file).Decode(&orders); err != nil {
			panic("Failed to decode order file")
		}

		store := inmemoryStores.GetOrderStoreInstance()
		customerStore := inmemoryStores.GetCustomerStoreInstance()
		bookStore := inmemoryStores.GetBookStoreInstance()

		var failedOrders []StructureData.Order

		for _, order := range orders {
			customer, customerErr := customerStore.GetCustomer(order.Customer.ID)
			if customerErr != nil {
				log.Printf("Skipping order ID %d: Customer with ID %d not found", order.ID, order.Customer.ID)
				failedOrders = append(failedOrders, order)
				continue
			}
			order.Customer = customer

			validOrder := true
			for i, item := range order.Items {
				book, bookErr := bookStore.GetBook(item.Book.ID)
				if bookErr != nil {
					log.Printf("Skipping order ID %d: Book ID %d not found", order.ID, item.Book.ID)
					validOrder = false
					break
				}
				order.Items[i].Book = book
			}

			if !validOrder {
				failedOrders = append(failedOrders, order)
				continue
			}

			if _, errResp := store.CreateOrder(order); errResp != nil {
				log.Printf("Error creating order ID %d: %v", order.ID, errResp)
				failedOrders = append(failedOrders, order)
			}
		}

		if len(failedOrders) > 0 {
			log.Println("Persisting failed orders for debugging.")
			saveFailedOrders(failedOrders)
		}
	}
}

// Helper function to persist failed orders
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

// GetAllOrders handles the GET /orders request
func GetAllOrders(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetOrderStoreInstance()

	// Retrieve all orders
	orders := store.GetAllOrders()

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// GetOrderByID handles the GET /orders/{id} request
func GetOrderByID(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetOrderStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid order ID"})
		return
	}

	// Retrieve the order by ID
	order, errResp := store.GetOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func CreateOrder(w http.ResponseWriter, r *http.Request) {
	orderStore := inmemoryStores.GetOrderStoreInstance()
	customerStore := inmemoryStores.GetCustomerStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()

	// Decode the request body
	var order StructureData.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate customer
	customer, errResp := customerStore.GetCustomer(order.Customer.ID)
	if errResp != nil {
		// If ID is not found, try validating by email
		for _, existingCustomer := range customerStore.GetAllCustomers() {
			if existingCustomer.Email == order.Customer.Email {
				customer = existingCustomer
				break
			}
		}
		// If still not found
		if customer.ID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer does not exist"})
			return
		}
	}
	// Fill customer details in the order
	order.Customer = customer

	// Validate books in the order
	validItems := []StructureData.OrderItem{} // Store valid items
	for _, item := range order.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr != nil {
			log.Printf("Skipping book ID %d: Does not exist", item.Book.ID)
			continue
		}

		// Check if the stock is sufficient
		if item.Quantity > book.Stock || book.Stock == 0 {
			log.Printf("Skipping book ID %d: Insufficient stock (stock=%d)", book.ID, book.Stock)
			continue
		}

		// Deduct stock
		book.Stock -= item.Quantity

		// Update the book in the store
		_, updateErr := bookStore.UpdateBook(book.ID, book)
		if updateErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Failed to update book stock"})
			return
		}

		// Add the item to the valid items list
		item.Book = book // Ensure all fields in the book are updated
		validItems = append(validItems, item)
	}

	// If no valid items are present, return an error
	if len(validItems) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "No valid books available to create the order"})
		return
	}

	// Update the order with valid items
	order.Items = validItems

	// Create the order in the store
	createdOrder, errResp := orderStore.CreateOrder(order)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist to JSON file
	if err := persistOrdersToFile(orderStore.GetAllOrders()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Persist updated books to the JSON file
	if err := persistBooksToFile(bookStore.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving updated book data"})
		return
	}

	// Return the created order
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdOrder)
}


func UpdateOrder(w http.ResponseWriter, r *http.Request) {
	orderStore := inmemoryStores.GetOrderStoreInstance()
	customerStore := inmemoryStores.GetCustomerStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid order ID"})
		return
	}

	// Retrieve the existing order
	existingOrder, errResp := orderStore.GetOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Decode the request body
	var updatedOrder StructureData.Order
	if err := json.NewDecoder(r.Body).Decode(&updatedOrder); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate customer
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

	// Adjust stock based on changes to the order
	for _, item := range existingOrder.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr == nil {
			book.Stock += item.Quantity                // Revert the stock changes from the old order
			_, _ = bookStore.UpdateBook(book.ID, book) // Update silently
		}
	}

	// Validate and update books for the new order
	validItems := []StructureData.OrderItem{} // Store valid items
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

		item.Book = book // Ensure all fields in the book are updated
		validItems = append(validItems, item)
	}

	// If no valid items are present, return an error
	if len(validItems) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "No valid books available to update the order"})
		return
	}

	// Update the order with valid items
	updatedOrder.Items = validItems

	// Update the order in the store
	updatedOrder, errResp = orderStore.UpdateOrder(id, updatedOrder)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist the updated order and books
	if err := persistOrdersToFile(orderStore.GetAllOrders()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving order data"})
		return
	}

	if err := persistBooksToFile(bookStore.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving updated book data"})
		return
	}

	// Return the updated order
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedOrder)
}


func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	orderStore := inmemoryStores.GetOrderStoreInstance()
	bookStore := inmemoryStores.GetBookStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid order ID"})
		return
	}

	// Retrieve the order to be deleted
	order, errResp := orderStore.GetOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Adjust the stock for the books in the order
	for _, item := range order.Items {
		book, bookErr := bookStore.GetBook(item.Book.ID)
		if bookErr != nil {
			log.Printf("Warning: Book with ID %d not found while deleting order %d", item.Book.ID, id)
			continue
		}

		// Increase the stock by the quantity in the order
		book.Stock += item.Quantity

		// Update the book in the store
		_, updateErr := bookStore.UpdateBook(book.ID, book)
		if updateErr != nil {
			log.Printf("Error updating stock for book ID %d: %s", book.ID, updateErr.Message)
			continue
		}
	}

	// Delete the order from the store
	errResp = orderStore.DeleteOrder(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist the updated orders to the JSON file
	if err := persistOrdersToFile(orderStore.GetAllOrders()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving order data"})
		return
	}

	// Persist the updated books to the JSON file
	if err := persistBooksToFile(bookStore.GetAllBooks()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving book data"})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusNoContent)
}

// SearchOrders handles the POST /orders/search request
func SearchOrders(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetOrderStoreInstance()

	// Decode the search criteria from the request body
	var criteria StructureData.OrderSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
		return
	}

	// Perform the search
	searchResults, errResp := store.SearchOrders(criteria)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Return the search results as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResults)
}

// persistOrdersToFile saves all orders to the JSON file in a pretty JSON format
func persistOrdersToFile(orders []StructureData.Order) error {
	file, err := os.Create(orderFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use a pretty JSON encoder
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Add indentation for better readability

	if err := encoder.Encode(orders); err != nil {
		return err
	}

	return nil
}

// generateSalesReport generates a sales report for the last 24 hours using a context.
func GenerateSalesReport(ctx context.Context) {
	store := inmemoryStores.GetOrderStoreInstance()

	// Define the time range for the report
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	// Retrieve orders within the time range
	orders, err := store.GetOrdersInTimeRange(startTime, endTime)
	if err != nil {
		log.Printf("Error fetching orders for sales report: %v\n", err)
		return
	}

	// Ensure there are orders to process
	if len(orders) == 0 {
		log.Println("No orders found for the sales report generation.")

		// Create an empty sales report with just the timestamp
		report := StructureData.SalesReport{
			Timestamp:       endTime,
			TotalRevenue:    0,
			TotalOrders:     0,
			TopSellingBooks: []StructureData.TopSellingBook{},
		}

		// Save the empty sales report to the file
		if err := SaveSalesReport(ctx, report); err != nil {
			log.Printf("Error saving empty sales report: %v\n", err)
		}

		return
	}

	// Calculate sales report details
	var totalRevenue float64
	var totalOrders int
	bookSales := make(map[int]*StructureData.TopSellingBook) // book ID -> TopSellingBook

	bookStore := inmemoryStores.GetBookStoreInstance()

	for _, order := range orders {
		select {
		case <-ctx.Done(): // Check for cancellation
			log.Println("GenerateSalesReport was canceled.")
			return
		default:
		}

		totalRevenue += order.TotalPrice
		totalOrders++
		for _, item := range order.Items {
			select {
			case <-ctx.Done(): // Check for cancellation
				log.Println("GenerateSalesReport was canceled.")
				return
			default:
			}

			// Ensure book exists in the store
			book, bookErr := bookStore.GetBook(item.Book.ID)
			if bookErr != nil {
				log.Printf("Skipping order ID %d: Book ID %d not found in the in-memory store. Verify book loading.", order.ID, item.Book.ID)
				continue
			}

			// Initialize if not exists
			if _, exists := bookSales[book.ID]; !exists {
				bookSales[book.ID] = &StructureData.TopSellingBook{
					Book:         book,
					QuantitySold: 0,
				}
			}
			// Increment quantity sold
			bookSales[book.ID].QuantitySold += item.Quantity
		}
	}

	// Convert bookSales map to slice for sorting
	topSellingBooks := make([]StructureData.TopSellingBook, 0, len(bookSales))
	for _, bookSale := range bookSales {
		topSellingBooks = append(topSellingBooks, *bookSale)
	}

	// Sort Top-Selling Books by Revenue (Price * Quantity Sold)
	sort.Slice(topSellingBooks, func(i, j int) bool {
		revenueI := topSellingBooks[i].Book.Price * float64(topSellingBooks[i].QuantitySold)
		revenueJ := topSellingBooks[j].Book.Price * float64(topSellingBooks[j].QuantitySold)
		return revenueI > revenueJ
	})

	// Limit to Top 5 Books
	if len(topSellingBooks) > 5 {
		topSellingBooks = topSellingBooks[:5]
	}

	// Create the sales report
	report := StructureData.SalesReport{
		Timestamp:       endTime,
		TotalRevenue:    totalRevenue,
		TotalOrders:     totalOrders,
		TopSellingBooks: topSellingBooks,
	}

	// Save the sales report to the file
	if err := SaveSalesReport(ctx, report); err != nil {
		log.Printf("Error saving sales report: %v\n", err)
	}
}

// SaveSalesReport saves the sales report to the sales_reports.json file with a pretty JSON format
func SaveSalesReport(ctx context.Context, report StructureData.SalesReport) error {
	var reports []StructureData.SalesReport

	// Load existing reports
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
	case <-ctx.Done(): // Check for cancellation
		return ctx.Err()
	default:
	}

	// Append the new report and save it
	reports = append(reports, report)
	file, err := os.Create(salesReportFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use a pretty JSON encoder
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Add indentation for better readability

	select {
	case <-ctx.Done(): // Check for cancellation
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
	// Load sales reports from the file
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
	case <-ctx.Done(): // Check for cancellation
		w.WriteHeader(http.StatusRequestTimeout)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Request canceled by client"})
		return
	default:
	}

	// Extract query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var filteredReports []StructureData.SalesReport

	// Parse the dates if provided
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

		// Filter the reports by date range
		for _, report := range reports {
			if report.Timestamp.After(startDate) && report.Timestamp.Before(endDate.Add(24*time.Hour)) {
				filteredReports = append(filteredReports, report)
			}
		}
	} else {
		// If no dates provided, return all reports
		filteredReports = reports
	}

	// Use a pretty JSON encoder for the response
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ") // Add indentation for better readability

	if err := encoder.Encode(filteredReports); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error encoding response data"})
		return
	}
}
