package StructureData

import "time"

type Customer struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Address   Address   `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

type CustomerSearchCriteria struct {
	IDs             []int                `json:"ids,omitempty"`          
	Names           []string             `json:"names,omitempty"`    
	Emails          []string             `json:"emails,omitempty"`     
	MinCreatedAt    time.Time            `json:"min_created_at,omitempty"` 
	MaxCreatedAt    time.Time            `json:"max_created_at,omitempty"` 
	AddressCriteria AddressSearchCriteria `json:"address_criteria,omitempty"` // Embedded address filtering criteria
}
