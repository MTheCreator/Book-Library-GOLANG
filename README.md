# Final Project: Bookstore Management System  

## Project Description  

This project is a **Bookstore Management System** built using **Go**. It provides a **REST API** for managing customers, authors, books, orders, and sales. The system supports **CRUD operations** and offers **sales reports** generation. It ensures **robust error handling** and **data integrity** through various business logic constraints.  

## Collaborators  
- **Mounia Baddou**:
  email: Mounia.Baddou@um6p.ma,
  github: MTheCreator   
- **Kawtar Labzae**
  email: Kawtar.Labzae@um6p.ma,
  github: kawtarlabzae  

---

## Key Features  

1. **Customer Management**  
   - Create, update, retrieve, delete, and search customers.  
   - Customers cannot be deleted if linked to an existing order.  

2. **Author Management**  
   - Create, update, retrieve, delete, and search authors.  
   - When an author is deleted, their books are also deleted unless the books are part of an order.  

3. **Book Management**  
   - Create, update, retrieve, delete, and search books.  
   - Books with `stock = 0` cannot be used in new orders.  

4. **Order Management**  
   - Create, update, retrieve, delete, and search orders.  
   - Orders must reference existing customers and books. Deleting an order restores the stock of the associated books.  

5. **Sales Reports**  
   - Generate and retrieve sales reports for specific date ranges or instantly for the last 24 hours.  
   - Reports summarize total sales and revenue.  

---

## Technical Requirements  

- **Programming Language**: Go  
- **API**: REST API with Swagger Documentation  
- **Data Storage**: In-memory JSON-based storage for testing purposes  
- **Tools**: Swagger for API documentation, Go modules for dependency management  

---

## Folder Structure (As of last update)
```
Final-project/
├── Auth/                # Authorization related codes
├── Controllers/         # Logic for handling requests and routing
├── Documentation/       # Project documentation files
├── InmemoryStores/      # In-memory data storage (for testing)
├── Interfaces/          # Interface definitions for abstractions
├── StructureData/       # Data structures (e.g., structs for Customers, Books, etc.)
├── swaggerfiles/        # Swagger API definitions
├── utils/               # Utility functions or helpers
├── authors.json         # Sample data for authors
├── books.json           # Sample data for books
├── customers.json       # Sample data for customers
├── go.mod               # Go module file for dependencies
├── go.sum               # Go dependency checksum file
├── main.go              # Main entry point for the application
├── users.json           # Sample users for testing
└── sales_reports.json   # Sample sales report data
```


## Steps to Run the Project  

1. **Clone the Repository**  
   ```bash
   git clone https://github.com/MTheCreator/Book-Library-GOLANG
   ```

2. **Navigate to the Project Folder**  
   ```bash
   cd Final-project
   ```

3. **Run the Main File**  
   ```bash
   go run main.go
   ```

---

## How to Use the System  

### 1. Customer Management  
Create, update, retrieve, delete, and search customers.  

Example JSON for creating a customer:  
```json
{
  "name": "John Doe",
  "email": "john.doe@example.com",
  "password": "/* Enter your user's password */"
  "address": {
    "street": "123 Elm St",
    "city": "Springfield",
    "state": "IL",
    "postal_code": "62701"
  },
  "role" : /* Should be either 'admin' for administrator or 'user' for regular users*/
}
```

### 2. Author Management  
Create authors or manage existing ones. When deleting an author, books will be deleted unless they are part of an order.

### 3. Book Management  
Add books for an author or let the system automatically create an author when adding a book.  

Example JSON for creating a book:  
```json
{
  "title": "The Great Adventure",
  "price": 19.99,
  "stock": 100,
  "author": {
    "first_name": "Jane",
    "last_name": "Doe",
    "bio": "Famous author of adventurous tales."
  }
}
```

### 4. Order Management  
Create orders with existing customers and books.  

Example JSON for creating an order:  
```json
{
  "customer": {
    "id": 1
  },
  "items": [
    {
      "book": {
        "id": 1
      },
      "quantity": 2
    }
  ]
}
```

### 5. Sales Reports  
Generate and retrieve sales reports for a specific date range.  

Example request for retrieving sales reports:  
```http
GET /reports/sales?start_date=2025-01-01&end_date=2025-01-31
```
---

## API Endpoints

### Authentication & Authorization
- Most customer endpoints are protected. Only **Admins** can access certain routes (e.g., get all customers)

### Customer Routes

| Method | Endpoint             | Description                                     |
|--------|----------------------|-------------------------------------------------|
| GET    | /customers           | Get a list of all customers (Admin only).       |
| GET    | /customers/:id       | Get details of a specific customer by ID.       |
| POST   | /customers           | Create a new customer.                          |
| PUT    | /customers/:id       | Update a customer’s information.                |
| DELETE | /customers/:id       | Delete a customer by ID.                        |
| POST   | /customers/search    | Search customers based on filter criteria.      |


### Author Routes

| Method | Endpoint           | Description                                     |
|--------|--------------------|-------------------------------------------------|
| GET    | /authors           | Get a list of all authors.                      |
| GET    | /authors/:id       | Get details of a specific author by ID.         |
| POST   | /authors           | Create a new author.                            |
| PUT    | /authors/:id       | Update author information.                      |
| DELETE | /authors/:id       | Delete an author by ID.                         |
| POST   | /authors/search    | Search authors based on filter criteria.        |


### Book Routes

| Method | Endpoint         | Description                                     |
|--------|------------------|-------------------------------------------------|
| GET    | /books           | Get a list of all books.                        |
| GET    | /books/:id       | Get details of a specific book by ID.          |
| POST   | /books           | Create a new book.                              |
| PUT    | /books/:id       | Update book information.                        |
| DELETE | /books/:id       | Delete a book by ID.                            |
| POST   | /books/search    | Search books based on filter criteria.         |


### Order Routes

| Method | Endpoint          | Description                                     |
|--------|-------------------|-------------------------------------------------|
| GET    | /orders           | Get a list of all orders.                       |
| GET    | /orders/:id       | Get details of a specific order by ID.         |
| POST   | /orders           | Create a new order.                             |
| PUT    | /orders/:id       | Update an order by ID.                          |
| DELETE | /orders/:id       | Delete an order by ID.                          |
| POST   | /orders/search    | Search orders based on filter criteria.        |


### Report Routes

| Method | Endpoint                   | Description                                     |
|--------|----------------------------|-------------------------------------------------|
| GET    | /reports/sales             | Retrieve sales report summary.                  |
| POST   | /reports/sales/generate    | Generate a new sales report.                    |


### Review Routes

| Method | Endpoint          | Description                                     |
|--------|-------------------|-------------------------------------------------|
| POST   | /reviews          | Add a new review to a book.                     |
| GET    | /reviews          | Get all reviews for a specific book (via `book_id` query). |
| DELETE | /reviews/:id      | Delete a review by ID.                          |

---

## API Documentation  

API documentation is available in the **swaggerfiles/** directory. To view it:  
1. Run the project.  
2. Visit `http://localhost:8080/swagger/index.html` in your browser.
 
##### (This swagger file is deprecated since there was much more that was added to the code!)
---

## Contributing Guidelines  

We welcome contributions! Please follow these steps:  
1. **Fork** the repository.  
2. Create a new branch:  
   ```bash
   git checkout -b feature/your-feature-name
   ```  
3. **Commit** your changes and push the branch to your fork.  
4. Open a **Pull Request**.  

---
## Development Status

### Authentication & Authorization
- Basic signup/login endpoints implemented with JWT issuance.
- Role-based authorization checks in place for protected routes.
- Core modules compile and run without errors.
- No automated tests yet—potential bugs remain unverified.

### Testing
- Unit and integration test suites are pending.
- CI/CD pipeline and coverage metrics are not configured.

### Dockerization
- Initial Dockerfile and `docker-compose.yml` scaffolds created.
- Local container build and run attempts completed but not production-ready.
- Further work and expertise required to finalize a reliable, optimized Docker setup.

## License
---
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


