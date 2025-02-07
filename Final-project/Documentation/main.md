Here is the markdown documentation for the provided `main.go` file:

---

# Project Documentation

## main.go

This file serves as the entry point of the application, configuring and starting an HTTP server with various routes for managing customers, authors, books, orders, and sales reports.

### Key Features

1. **Data Initialization**:
   - Initializes JSON files for persistence:
     - Customers
     - Authors
     - Books
     - Orders
   - Ensures data is loaded into in-memory stores at startup.

2. **Sales Report Generation**:
   - Periodically generates sales reports every 24 hours.
   - Provides an endpoint to trigger report generation manually.

3. **Router Setup**:
   - Configures routes for managing resources such as customers, authors, books, and orders using the `httprouter` package.

4. **Graceful Shutdown**:
   - Handles termination signals (e.g., `SIGTERM`) to allow the server to shut down gracefully.

---

### Routes and Endpoints

#### **Customer Routes**
- `GET /customers`: Retrieve all customers.
- `GET /customers/:id`: Retrieve a specific customer by ID.
- `POST /customers`: Create a new customer.
- `PUT /customers/:id`: Update a specific customer by ID.
- `DELETE /customers/:id`: Delete a specific customer by ID.
- `POST /customers/search`: Search for customers based on criteria.

#### **Author Routes**
- `GET /authors`: Retrieve all authors.
- `GET /authors/:id`: Retrieve a specific author by ID.
- `POST /authors`: Create a new author.
- `PUT /authors/:id`: Update a specific author by ID.
- `DELETE /authors/:id`: Delete a specific author by ID.
- `POST /authors/search`: Search for authors based on criteria.

#### **Book Routes**
- `GET /books`: Retrieve all books.
- `GET /books/:id`: Retrieve a specific book by ID.
- `POST /books`: Create a new book.
- `PUT /books/:id`: Update a specific book by ID.
- `DELETE /books/:id`: Delete a specific book by ID.
- `POST /books/search`: Search for books based on criteria.

#### **Order Routes**
- `GET /orders`: Retrieve all orders.
- `GET /orders/:id`: Retrieve a specific order by ID.
- `POST /orders`: Create a new order.
- `PUT /orders/:id`: Update a specific order by ID.
- `DELETE /orders/:id`: Delete a specific order by ID.
- `POST /orders/search`: Search for orders based on criteria.

#### **Report Routes**
- `GET /reports/sales`: Retrieve sales reports.
- `POST /reports/sales/generate`: Manually generate a sales report.

---

### Server Configuration

- **Address**: `:8080`
- **Router**: Configured with `httprouter`.
- **Graceful Shutdown**: Waits for termination signals and allows the server to shut down within a 10-second timeout.

---

### Periodic Tasks

- **Sales Report Generation**:
  - Runs every 24 hours.
  - Logs and generates sales reports based on recent orders.

---

This documentation outlines the structure and functionality of the `main.go` file, detailing the endpoints and server lifecycle.