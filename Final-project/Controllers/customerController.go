package Controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	inmemoryStores "finalProject/InmemoryStores"
	"finalProject/StructureData"
	"finalProject/auth"
	postgresStores "finalProject/postgresStores"
)

func InitializeCustomerFile() {
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()
	memStore := inmemoryStores.GetCustomerStoreInstance()

	pgCustomers := pgStore.GetAllCustomers()

	// Only initialize if memory store is empty
	if len(memStore.GetAllCustomers()) == 0 {
		for _, customer := range pgCustomers {
			_, err := memStore.CreateCustomer(customer)
			if err != nil {
				log.Printf("Error loading customer %d into memory: %v", customer.ID, err.Message)
			}
		}
		log.Printf("Loaded %d customers from PostgreSQL into memory", len(pgCustomers))
	}
}
func GetAllCustomers(w http.ResponseWriter, r *http.Request) {

	pgStore := postgresStores.GetPostgresCustomerStoreInstance()

	// Fetch latest customers from PostgreSQL
	pgCustomers := pgStore.GetAllCustomers()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pgCustomers)
}

func GetCustomerByID(w http.ResponseWriter, r *http.Request) {
	memStore := inmemoryStores.GetCustomerStoreInstance()
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()

	idStr := r.URL.Path[len("/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid customer ID"})
		return
	}

	// Check in-memory store first
	customer, errResp := memStore.GetCustomer(id)
	if errResp == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(customer)
		return
	}

	// Fallback to PostgreSQL if not found in-memory
	pgCustomer, pgErr := pgStore.GetCustomer(id)
	if pgErr != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer not found"})
		return
	}

	// Add to in-memory store for future requests
	memStore.CreateCustomer(pgCustomer)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pgCustomer)
}

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
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Customer linked to orders"})
			return
		}
	}

	errResp := store.DeleteCustomer(id)
	if errResp != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	pgErr := pgStore.DeleteCustomer(id)
	if pgErr != nil {
		log.Printf("Error deleting customer: %v", pgErr.Message)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Customer deleted"})
}

func CreateCustomer(w http.ResponseWriter, r *http.Request) {
	store := inmemoryStores.GetCustomerStoreInstance()
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()

	var customer StructureData.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Ensure required fields are provided
	if customer.Name == "" || customer.Email == "" || customer.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Name, Email, and Password are required"})
		return
	}

	// Hash the password before storing
	err := customer.HashPassword(customer.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error hashing password"})
		return
	}

	// Check if the email already exists
	existingCustomers := pgStore.GetAllCustomers()
	for _, existingCustomer := range existingCustomers {
		if existingCustomer.Email == customer.Email {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Email already exists"})
			return
		}
	}

	// Set the CreatedAt field
	customer.CreatedAt = time.Now()

	// Save to PostgreSQL
	createdPgCustomer, pgErr := pgStore.CreateCustomer(customer)
	if pgErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error creating customer in PostgreSQL"})
		return
	}

	// Save to in-memory store
	_, errResp := store.CreateCustomer(createdPgCustomer)
	if errResp != nil {
		// Roll back PostgreSQL insert if needed
		pgStore.DeleteCustomer(createdPgCustomer.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Generate JWT token for the newly created user
	token, jwtErr := auth.GenerateJWT(createdPgCustomer.ID, createdPgCustomer.Email, createdPgCustomer.Username, createdPgCustomer.Role)
	if jwtErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error generating JWT token"})
		return
	}

	// Return the customer info + token
	response := map[string]interface{}{
		"message": "Customer created successfully",
		"token":   token,
		"customer": map[string]interface{}{
			"id":         createdPgCustomer.ID,
			"name":       createdPgCustomer.Name,
			"email":      createdPgCustomer.Email,
			"username":   createdPgCustomer.Username,
			"created_at": createdPgCustomer.CreatedAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	// Retrieve store instances.
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()
	memStore := inmemoryStores.GetCustomerStoreInstance()

	// Extract customer ID from the URL.
	idStr := r.URL.Path[len("/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid customer ID"})
		return
	}

	// Decode the incoming customer update.
	var customer StructureData.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid input"})
		return
	}

	// Validate required fields.
	if customer.Name == "" || customer.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Name and Email required"})
		return
	}

	// First, update the customer in PostgreSQL.
	updatedPgCustomer, pgErr := pgStore.UpdateCustomer(id, customer)
	if pgErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: fmt.Sprintf("Error updating customer in PostgreSQL: %v", pgErr.Message)})
		return
	}

	// Then, update the customer in the in-memory store.
	updatedMemCustomer, memErrResp := memStore.UpdateCustomer(id, updatedPgCustomer)
	if memErrResp != nil {
		// Optionally: you could attempt to revert the PG update here.
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error updating in-memory customer; data may be inconsistent"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMemCustomer)
}
func SearchCustomers(w http.ResponseWriter, r *http.Request) {
	pgStore := postgresStores.GetPostgresCustomerStoreInstance()
	memStore := inmemoryStores.GetCustomerStoreInstance()

	var criteria StructureData.CustomerSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid criteria"})
		return
	}

	// Search in PostgreSQL for accurate results
	pgResults, pgErr := pgStore.SearchCustomers(criteria)
	if pgErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(pgErr)
		return
	}

	// Update in-memory store with the found customers
	for _, customer := range pgResults {
		existing, err := memStore.GetCustomer(customer.ID)
		if err != nil {
			memStore.CreateCustomer(customer)
		} else if !customersEqual(existing, customer) {
			memStore.UpdateCustomer(customer.ID, customer)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pgResults)
}

// Helper function to check customer equality
func customersEqual(a, b StructureData.Customer) bool {
	return a.Name == b.Name && a.Email == b.Email && a.Username == b.Username
}
