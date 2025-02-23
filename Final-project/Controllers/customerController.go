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
	store := inmemoryStores.GetCustomerStoreInstance()
	customers := store.GetAllCustomers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customers)
}

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

	// Ensure required fields are provided.
	if customer.Name == "" || customer.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Name and Email required"})
		return
	}

	// Overwrite the CreatedAt field with the current time.
	customer.CreatedAt = time.Now()

	// Check if the email already exists.
	existingCustomers := pgStore.GetAllCustomers()
	for _, existingCustomer := range existingCustomers {
		if existingCustomer.Email == customer.Email {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Email exists"})
			return
		}
	}

	// Create customer in PostgreSQL.
	createdPgCustomer, pgErr := pgStore.CreateCustomer(customer)
	if pgErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Error creating customer"})
		return
	}

	// Create customer in the in-memory store.
	createdCustomer, errResp := store.CreateCustomer(createdPgCustomer)
	if errResp != nil {
		// Optionally roll back the PostgreSQL insertion.
		pgStore.DeleteCustomer(createdPgCustomer.ID)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdCustomer)
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
	store := inmemoryStores.GetCustomerStoreInstance()
	var criteria StructureData.CustomerSearchCriteria
	if err := json.NewDecoder(r.Body).Decode(&criteria); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid criteria"})
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