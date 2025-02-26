package postgresStores

import (
	"database/sql"
	"finalProject/StructureData"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresCustomerStore implements the customer store using PostgreSQL.
type PostgresCustomerStore struct {
	db *sql.DB
}

var postgresCustomerStoreInstance *PostgresCustomerStore

func (store *PostgresCustomerStore) Close() error {
	return store.db.Close()
}

// GetPostgresCustomerStoreInstance returns a singleton instance.
func GetPostgresCustomerStoreInstance() *PostgresCustomerStore {
	if postgresCustomerStoreInstance == nil {
		connStr := "user=postgres password=root dbname=booklibrary sslmode=disable"
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			panic(fmt.Sprintf("Failed to connect to Postgres: %v", err))
		}
		if err := db.Ping(); err != nil {
			panic(fmt.Sprintf("Failed to ping Postgres: %v", err))
		}
		postgresCustomerStoreInstance = &PostgresCustomerStore{db: db}
	}
	return postgresCustomerStoreInstance
}

// CreateCustomer inserts a new customer into PostgreSQL.
// If customer.ID is nonzero, it will be inserted explicitly.
func (store *PostgresCustomerStore) CreateCustomer(customer StructureData.Customer) (StructureData.Customer, *StructureData.ErrorResponse) {
	var query string
	var args []interface{}

	if customer.ID != 0 {
		query = `INSERT INTO customers (id, name, username, email, password, street, city, state, postal_code, country, created_at)
		          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`
		args = []interface{}{
			customer.ID,
			customer.Name,
			customer.Username,
			customer.Email,
			customer.Password,
			customer.Address.Street,
			customer.Address.City,
			customer.Address.State,
			customer.Address.PostalCode,
			customer.Address.Country,
			customer.CreatedAt,
		}
	} else {
		query = `INSERT INTO customers (name, username, email, password, street, city, state, postal_code, country, created_at)
		          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
		args = []interface{}{
			customer.Name,
			customer.Username,
			customer.Email,
			customer.Password,
			customer.Address.Street,
			customer.Address.City,
			customer.Address.State,
			customer.Address.PostalCode,
			customer.Address.Country,
			customer.CreatedAt,
		}
	}

	err := store.db.QueryRow(query, args...).Scan(&customer.ID)
	if err != nil {
		return StructureData.Customer{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to insert customer: %v", err)}
	}
	return customer, nil
}

// GetCustomer retrieves a customer by its ID.
func (store *PostgresCustomerStore) GetCustomer(id int) (StructureData.Customer, *StructureData.ErrorResponse) {
	var customer StructureData.Customer
	var street, city, state, postalCode, country string
	query := `SELECT id, name, username, email, street, city, state, postal_code, country, created_at FROM customers WHERE id=$1`
	row := store.db.QueryRow(query, id)
	err := row.Scan(&customer.ID, &customer.Name, &customer.Email,
		&street, &city, &state, &postalCode, &country, &customer.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return StructureData.Customer{}, &StructureData.ErrorResponse{Message: "Customer not found"}
		}
		return StructureData.Customer{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Error fetching customer: %v", err)}
	}
	customer.Address = StructureData.Address{
		Street:     street,
		City:       city,
		State:      state,
		PostalCode: postalCode,
		Country:    country,
	}
	return customer, nil
}

// GetAllCustomers, UpdateCustomer, DeleteCustomer, and SearchCustomers remain unchanged.

// GetAllCustomers retrieves all customers from the database.
func (store *PostgresCustomerStore) GetAllCustomers() []StructureData.Customer {
	customers := []StructureData.Customer{}
	query := `SELECT id, name, username, email, street, city, state, postal_code, country, created_at FROM customers`
	rows, err := store.db.Query(query)
	if err != nil {
		return customers
	}
	defer rows.Close()
	for rows.Next() {
		var customer StructureData.Customer
		var street, city, state, postalCode, country string
		err := rows.Scan(&customer.ID, &customer.Name, &customer.Email,
			&street, &city, &state, &postalCode, &country, &customer.CreatedAt)
		if err != nil {
			continue
		}
		customer.Address = StructureData.Address{
			Street:     street,
			City:       city,
			State:      state,
			PostalCode: postalCode,
			Country:    country,
		}
		customers = append(customers, customer)
	}
	return customers
}

// UpdateCustomer updates an existing customer in the database.
func (store *PostgresCustomerStore) UpdateCustomer(id int, customer StructureData.Customer) (StructureData.Customer, *StructureData.ErrorResponse) {
	query := `UPDATE customers SET name=$1, username=$2, email=$3, street=$4, city=$5, state=$6, postal_code=$7, country=$8 WHERE id=$9`
	res, err := store.db.Exec(query,
		customer.Name,
		customer.Username,
		customer.Email,
		customer.Address.Street,
		customer.Address.City,
		customer.Address.State,
		customer.Address.PostalCode,
		customer.Address.Country,
		id,
	)
	if err != nil {
		return StructureData.Customer{}, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to update customer: %v", err)}
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return StructureData.Customer{}, &StructureData.ErrorResponse{Message: "Customer not found"}
	}
	customer.ID = id
	return customer, nil
}

// DeleteCustomer removes a customer from the database.
func (store *PostgresCustomerStore) DeleteCustomer(id int) *StructureData.ErrorResponse {
	query := `DELETE FROM customers WHERE id=$1`
	res, err := store.db.Exec(query, id)
	if err != nil {
		return &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to delete customer: %v", err)}
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return &StructureData.ErrorResponse{Message: "Customer not found"}
	}
	return nil
}

// SearchCustomers filters customers based on the search criteria.
// For simplicity, this implementation fetches all customers and then applies in-memory filtering.
func (store *PostgresCustomerStore) SearchCustomers(criteria StructureData.CustomerSearchCriteria) ([]StructureData.Customer, *StructureData.ErrorResponse) {
	allCustomers := store.GetAllCustomers()
	var result []StructureData.Customer
	for _, customer := range allCustomers {
		// Filter by IDs.
		if len(criteria.IDs) > 0 {
			match := false
			for _, id := range criteria.IDs {
				if customer.ID == id {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		// Filter by Names.
		if len(criteria.Names) > 0 {
			match := false
			for _, name := range criteria.Names {
				if customer.Name == name {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		// Filter by Emails.
		if len(criteria.Emails) > 0 {
			match := false
			for _, email := range criteria.Emails {
				if customer.Email == email {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		// Filter by creation time.
		if !criteria.MinCreatedAt.IsZero() && customer.CreatedAt.Before(criteria.MinCreatedAt) {
			continue
		}
		if !criteria.MaxCreatedAt.IsZero() && customer.CreatedAt.After(criteria.MaxCreatedAt) {
			continue
		}
		// Filter by address criteria.
		if !matchAddressCriteria(customer.Address, criteria.AddressCriteria) {
			continue
		}
		result = append(result, customer)
	}
	return result, nil
}

// matchAddressCriteria matches a customer's address with the specified search criteria.
func matchAddressCriteria(address StructureData.Address, criteria StructureData.AddressSearchCriteria) bool {
	// If no criteria are provided, assume a match.
	if len(criteria.Streets) == 0 && len(criteria.Cities) == 0 && len(criteria.States) == 0 &&
		len(criteria.PostalCodes) == 0 && len(criteria.Countries) == 0 {
		return true
	}
	if len(criteria.Streets) > 0 {
		found := false
		for _, s := range criteria.Streets {
			if address.Street == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(criteria.Cities) > 0 {
		found := false
		for _, city := range criteria.Cities {
			if address.City == city {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(criteria.States) > 0 {
		found := false
		for _, state := range criteria.States {
			if address.State == state {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(criteria.PostalCodes) > 0 {
		found := false
		for _, p := range criteria.PostalCodes {
			if address.PostalCode == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(criteria.Countries) > 0 {
		found := false
		for _, c := range criteria.Countries {
			if address.Country == c {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
