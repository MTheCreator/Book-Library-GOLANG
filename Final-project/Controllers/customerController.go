package Controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	inmemoryStores "finalProject/InmemoryStores"
	postgresStores "finalProject/postgresStores"
	"finalProject/StructureData"
)

// JSON file path for persistence.
var customerFile = "customers.json"

// InitializeCustomerFile ensures the JSON file for customers exists and loads data into both the in-memory and PostgreSQL stores.
func InitializeCustomerFile() {
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()
    existingCustomers := pgStore.GetAllCustomers()
    if len(existingCustomers) > 0 {
        log.Println("Customers already exist in PostgreSQL; skipping JSON initialization for Customers.")
        return
    }
	if _, err := os.Stat(customerFile); os.IsNotExist(err) {
		// Create an empty file.
		file, _ := os.Create(customerFile)
		file.Write([]byte("[]"))
		file.Close()
	} else {
		// Load customers from JSON.
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

		store := inmemoryStores.GetCustomerStoreInstance()
		pgStore := postgresStores.GetPostgresCustomerStoreInstance()
		for _, customer := range customers {
			// Create in PostgreSQL first (this uses the customer ID from JSON if available).
			createdPgCustomer, errResp := pgStore.CreateCustomer(customer)
			if errResp != nil {
				log.Printf("Error creating customer in PostgreSQL for customer ID %d: %v", customer.ID, errResp.Message)
				continue
			}
			// Then create in the in-memory store.
			_, errResp = store.CreateCustomer(createdPgCustomer)
			if errResp != nil {
				log.Printf("Error creating customer in in-memory store for customer ID %d: %v", createdPgCustomer.ID, errResp.Message)
				// Optionally, delete from PostgreSQL here.
				continue
			}
		}
	}
}

// GetAllCustomers handles the GET /customers request.
func GetAllCustomers(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()
	customers := store.GetAllCustomers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customers)
}

// GetCustomerByID handles the GET /customers/{id} request.
func GetCustomerByID(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()

	idStr := r.URL.Path[len("/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid customer ID"})
		return
	}

	customer, errResp := store.GetCustomer(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customer)
}

// DeleteCustomer handles the DELETE /customers/{id} request.
func DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()
	orderStore := inmemoryStores.GetOrderStoreInstance()
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()

	idStr := r.URL.Path[len("/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid customer ID"})
		return
	}

	orders := orderStore.GetAllOrders()
	for _, order := range orders {
		if order.Customer.ID == id {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer cannot be deleted as it is linked to existing orders"})
			return
		}
	}

	errResp := store.DeleteCustomer(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if err := persistCustomersToFile(store.GetAllCustomers()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	pgErr := pgStore.DeleteCustomer(id)
	if pgErr != nil {
		log.Printf("Error deleting customer from PostgreSQL: %v", pgErr.Message)
	}

	log.Printf("Customer with ID %d deleted successfully", id)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Customer deleted successfully"})
}

// CreateCustomer handles the POST /customers request.
func CreateCustomer(w http.ResponseWriter, r *http.Request) {
	// Get store instances.
	store := inmemoryStores.GetCustomerStoreInstance()
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()

	// Decode the request.
	var customer StructureData.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate input.
	if customer.Name == "" || customer.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Name and Email are required"})
		return
	}

	// Check for duplicate email in PostgreSQL.
	existingCustomers := pgStore.GetAllCustomers()
	for _, existingCustomer := range existingCustomers {
		if existingCustomer.Email == customer.Email {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer with this email already exists"})
			return
		}
	}

	// Create the customer in PostgreSQL first.
	createdPgCustomer, pgErr := pgStore.CreateCustomer(customer)
	if pgErr != nil {
		log.Printf("Error persisting customer to PostgreSQL: %v", pgErr.Message)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error creating customer"})
		return
	}

	// Now create in the in-memory store using the PostgreSQL record.
	createdCustomer, errResp := store.CreateCustomer(createdPgCustomer)
	if errResp != nil {
		// Optionally, roll back the PostgreSQL insert.
		pgStore.DeleteCustomer(createdPgCustomer.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Persist to JSON.
	if err := persistCustomersToFile(store.GetAllCustomers()); err != nil {
		// Roll back both stores if needed.
		store.DeleteCustomer(createdCustomer.ID)
		pgStore.DeleteCustomer(createdPgCustomer.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdCustomer)
}

// UpdateCustomer handles the PUT /customers/{id} request.
func UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()

	idStr := r.URL.Path[len("/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid customer ID"})
		return
	}

	var customer StructureData.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	if customer.Name == "" || customer.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Name and Email are required"})
		return
	}

	// Check for duplicate email (excluding current customer).
	for _, existingCustomer := range store.GetAllCustomers() {
		if existingCustomer.Email == customer.Email && existingCustomer.ID != id {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer with this email already exists"})
			return
		}
	}

	updatedCustomer, errResp := store.UpdateCustomer(id, customer)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if err := persistCustomersToFile(store.GetAllCustomers()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error saving data"})
		return
	}

	_, pgErr := pgStore.UpdateCustomer(id, updatedCustomer)
	if pgErr != nil {
		log.Printf("Error updating customer in PostgreSQL: %v", pgErr.Message)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedCustomer)
}

// SearchCustomers handles the POST /customers/search request.
func SearchCustomers(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()

	var criteria StructureData.CustomerSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid search criteria"})
		return
	}

	searchResults, errResp := store.SearchCustomers(criteria)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResults)
}

// persistCustomersToFile saves all customers to the JSON file.
func persistCustomersToFile(customers []StructureData.Customer) error {
	file, err := os.Create(customerFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(customers)
}
