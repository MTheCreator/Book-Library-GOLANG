package StructureData

import (
	"encoding/json"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Customer struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Address   Address   `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	Role      string    `json:"role"`
}

type CustomerSearchCriteria struct {
	IDs             []int                 `json:"ids,omitempty"`
	Names           []string              `json:"names,omitempty"`
	Emails          []string              `json:"emails,omitempty"`
	MinCreatedAt    time.Time             `json:"min_created_at,omitempty"`
	MaxCreatedAt    time.Time             `json:"max_created_at,omitempty"`
	AddressCriteria AddressSearchCriteria `json:"address_criteria,omitempty"` // Embedded address filtering criteria
}

func (user *Customer) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	user.Password = string(bytes)
	return nil
}

func (user *Customer) CheckPassword(providedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(providedPassword))
	if err != nil {
		return err
	}
	return nil
}

// Override JSON marshaling to mask the password
func (c Customer) MarshalJSON() ([]byte, error) {
	// Create a temporary struct to avoid recursion
	type Alias Customer
	return json.Marshal(&struct {
		Password string `json:"password"` // Override the Password field
		*Alias
	}{
		Password: "...", // Always set to "..."
		Alias:    (*Alias)(&c),
	})
}

// IsValidRole checks if the customer's role is either "admin" or "user"
func (c *Customer) IsValidRole() bool {
	return c.Role == "admin" || c.Role == "user"
}
