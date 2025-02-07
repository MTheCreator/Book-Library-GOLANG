package Controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	inmemoryStores "finalProject/InmemoryStores"
	"finalProject/StructureData"
)

// JSON file path for persistence
var customerFile = "customers.json"

// InitializeCustomerFile ensures the JSON file for customers exists and loads data into the in-memory store
func InitializeCustomerFile() {
	if _, err := os.Stat(customerFile); os.IsNotExist(err) {
		// If the file doesn't exist, create an empty one
		file, _ := os.Create(customerFile)
		file.Write([]byte("[]"))
		file.Close()
	} else {
		// Load customers from the JSON file into the in-memory store
		file, err := os.Open(customerFile)
		if err != nil {
			panic("Failed to open customer file")
		}
		defer file.Close()

		var customers []StructureData.Customer
		if err := json.NewDecoder(file).Decode(&customers); err != nil {
			log.Printf("Error decoding customer file: %v. Proceeding with an empty in-memory store.", err)
			customers = []StructureData.Customer{}
		}

		// Populate the in-memory store
		store := inmemoryStores.GetCustomerStoreInstance()
		for _, customer := range customers {
			store.CreateCustomer(customer)
		}
	}
}

// GetAllCustomers handles the GET /customers request
func GetAllCustomers(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()

	// Retrieve all customers from the store
	customers := store.GetAllCustomers()

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customers)
}

// GetCustomerByID handles the GET /customers/{id} request
func GetCustomerByID(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid customer ID"})
		return
	}

	// Retrieve the customer by ID
	customer, errResp := store.GetCustomer(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Return the customer as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customer)
}

// DeleteCustomer handles the DELETE /customers/{id} request
func DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()
	orderStore := inmemoryStores.GetOrderStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid customer ID"})
		return
	}

	// Check if the customer is linked to any orders
	orders := orderStore.GetAllOrders()
	for _, order := range orders {
		if order.Customer.ID == id {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer cannot be deleted as it is linked to existing orders"})
			return
		}
	}

	// Delete the customer from the store
	errResp := store.DeleteCustomer(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist to JSON file
	if err := persistCustomersToFile(store.GetAllCustomers()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Log and return success response
	log.Printf("Customer with ID %d deleted successfully", id)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Customer deleted successfully"})
}

// CreateCustomer handles the POST /customers request
func CreateCustomer(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()

	// Decode the request body
	var customer StructureData.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate input
	if customer.Name == "" || customer.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Name and Email are required"})
		return
	}

	// Check for duplicate email
	for _, existingCustomer := range store.GetAllCustomers() {
		if existingCustomer.Email == customer.Email {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer with this email already exists"})
			return
		}
	}

	// Add the customer to the in-memory store
	createdCustomer, errResp := store.CreateCustomer(customer)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist to JSON file
	if err := persistCustomersToFile(store.GetAllCustomers()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Return the created customer as a response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdCustomer)
}

// UpdateCustomer handles the PUT /customers/{id} request
func UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()

	// Extract ID from the URL
	idStr := r.URL.Path[len("/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid customer ID"})
		return
	}

	// Decode the request body
	var customer StructureData.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate input
	if customer.Name == "" || customer.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Name and Email are required"})
		return
	}

	// Check for duplicate email (excluding the current customer)
	for _, existingCustomer := range store.GetAllCustomers() {
		if existingCustomer.Email == customer.Email && existingCustomer.ID != id {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer with this email already exists"})
			return
		}
	}

	// Update the customer in the store
	updatedCustomer, errResp := store.UpdateCustomer(id, customer)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist to JSON file
	if err := persistCustomersToFile(store.GetAllCustomers()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	// Return the updated customer as a response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedCustomer)
}

// SearchCustomers handles the POST /customers/search request
func SearchCustomers(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()

	// Decode the search criteria from the request body
	var criteria StructureData.CustomerSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
		return
	}

	// Perform the search
	searchResults, errResp := store.SearchCustomers(criteria)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Return the search results as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResults)
}

// persistCustomersToFile saves all customers to the JSON file in a pretty JSON format
func persistCustomersToFile(customers []StructureData.Customer) error {
	file, err := os.Create(customerFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use a pretty JSON encoder
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Add indentation for better readability

	if err := encoder.Encode(customers); err != nil {
		return err
	}

	return nil
}
