package StructureData

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Customer struct {
	gorm.Model
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username" gorm:"unique"`
	Email     string    `json:"email" gorm:"unique"`
	Password  string    `json:"password"`
	Address   Address   `json:"address"`
	CreatedAt time.Time `json:"created_at"`
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
