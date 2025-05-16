# Project Documentation (Deprecated)

## bookController.go

This file provides HTTP handlers for managing books, interacting with an in-memory book store and author store, and persisting data to a JSON file.

### Key Endpoints

- **`GET /books`**: Retrieves all books.
- **`GET /books/{id}`**: Retrieves a specific book by ID.
- **`POST /books`**: Creates a new book. If the associated author does not exist, it creates the author as well.
- **`PUT /books/{id}`**: Updates an existing book by ID.
- **`DELETE /books/{id}`**: Deletes a book by ID. Prevents deletion if the book is linked to any orders.
- **`POST /books/search`**: Searches for books based on criteria.

### Utility Functions

- **`InitializeBookFile`**: Ensures the JSON file for books exists and loads data into the in-memory store.
- **`persistBooksToFile`**: Saves all books to a JSON file in a formatted manner.

---

## customerController.go

This file provides HTTP handlers for managing customers, interacting with an in-memory customer store, and persisting data to a JSON file.

### Key Endpoints

- **`GET /customers`**: Retrieves all customers.
- **`GET /customers/{id}`**: Retrieves a specific customer by ID.
- **`POST /customers`**: Creates a new customer.
- **`PUT /customers/{id}`**: Updates an existing customer by ID.
- **`DELETE /customers/{id}`**: Deletes a customer by ID. Prevents deletion if the customer is linked to any orders.
- **`POST /customers/search`**: Searches for customers based on criteria.

### Utility Functions

- **`InitializeCustomerFile`**: Ensures the JSON file for customers exists and loads data into the in-memory store.
- **`persistCustomersToFile`**: Saves all customers to a JSON file in a formatted manner.

---

## orderController.go

This file provides HTTP handlers for managing orders, interacting with in-memory order, book, and customer stores, and persisting data to JSON files.

### Key Endpoints

- **`GET /orders`**: Retrieves all orders.
- **`GET /orders/{id}`**: Retrieves a specific order by ID.
- **`POST /orders`**: Creates a new order, validates stock availability, and updates book inventory.
- **`PUT /orders/{id}`**: Updates an existing order by ID, including inventory adjustments.
- **`DELETE /orders/{id}`**: Deletes an order by ID and adjusts book stock accordingly.
- **`POST /orders/search`**: Searches for orders based on criteria.
- **`GET /sales-report`**: Retrieves sales reports, optionally filtered by a date range.

### Utility Functions

- **`InitializeOrderFile`**: Ensures the JSON file for orders exists and loads data into the in-memory store.
- **`persistOrdersToFile`**: Saves all orders to a JSON file in a formatted manner.
- **`GenerateSalesReport`**: Generates a sales report for the last 24 hours.
- **`SaveSalesReport`**: Saves a sales report to a JSON file.

---

## authorController.go

This file provides HTTP handlers for managing authors, interacting with an in-memory author store, and persisting data to a JSON file.

### Key Endpoints

- **`GET /authors`**: Retrieves all authors.
- **`GET /authors/{id}`**: Retrieves a specific author by ID.
- **`POST /authors`**: Creates a new author.
- **`PUT /authors/{id}`**: Updates an existing author by ID.
- **`DELETE /authors/{id}`**: Deletes an author by ID. Deletes associated books unless they are part of an order.
- **`POST /authors/search`**: Searches for authors based on criteria.

### Utility Functions

- **`InitializeAuthorFile`**: Ensures the JSON file for authors exists and loads data into the in-memory store.
- **`persistAuthorsToFile`**: Saves all authors to a JSON file in a formatted manner.
