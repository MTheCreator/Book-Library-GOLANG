package database

import (
	"finalProject/StructureData"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Instance *gorm.DB
var dbError error

func Connect(connectionString string) {
	dsn := "host=localhost user=postgres password=root dbname=booklibrary"
	Instance, dbError = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if dbError != nil {
		log.Fatal(dbError)
		panic("Cannot Connect to DB")
	}
	log.Println("Connected to Database!")
}

func Migrate() {
	Instance.AutoMigrate(&StructureData.Customer{})
	log.Println("Database Migration Completed!")
}
