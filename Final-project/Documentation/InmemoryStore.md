# Project Documentation (Deprecated)

## InmemoryBookStore.go

This file implements the `BookStore` interface using an in-memory data store.

### Structures

#### InMemoryBookStore
Represents the in-memory store for books.
```go
type InMemoryBookStore struct {
    mu     sync.RWMutex
    books  map[int]data.Book
    nextID int
}
```

### Key Methods
- `GetBookStoreInstance()`: Returns a singleton instance of `InMemoryBookStore`.
- `CreateBook(book data.Book)`: Adds a new book to the store.
- `GetBook(id int)`: Retrieves a book by its ID.
- `UpdateBook(id int, book data.Book)`: Updates details of an existing book.
- `DeleteBook(id int)`: Removes a book from the store.
- `GetAllBooks()`: Retrieves all books in the store.
- `SearchBooks(criteria data.BookSearchCriteria)`: Filters books based on search criteria.
- `AddBookDirectly(book data.Book)`: Adds a book with a specific ID, ensuring no ID collisions.

---

## InmemoryCustomerStore.go

This file implements the `CustomerStore` interface using an in-memory data store.

### Structures

#### InMemoryCustomerStore
Represents the in-memory store for customers.
```go
type InMemoryCustomerStore struct {
    mu        sync.RWMutex
    customers map[int]data.Customer
    nextID    int
}
```

### Key Methods
- `GetCustomerStoreInstance()`: Returns a singleton instance of `InMemoryCustomerStore`.
- `CreateCustomer(customer data.Customer)`: Adds a new customer to the store.
- `GetCustomer(id int)`: Retrieves a customer by its ID.
- `GetAllCustomers()`: Retrieves all customers in the store.
- `UpdateCustomer(id int, customer data.Customer)`: Updates details of an existing customer.
- `DeleteCustomer(id int)`: Removes a customer from the store.
- `SearchCustomers(criteria data.CustomerSearchCriteria)`: Filters customers based on search criteria.

---

## InmemoryOrderStore.go

This file implements the `OrderStore` interface using an in-memory data store.

### Structures

#### InMemoryOrderStore
Represents the in-memory store for orders.
```go
type InMemoryOrderStore struct {
    mu     sync.RWMutex
    orders map[int]data.Order
    nextID int
}
```

### Key Methods
- `GetOrderStoreInstance()`: Returns a singleton instance of `InMemoryOrderStore`.
- `CreateOrder(order data.Order)`: Adds a new order to the store, calculates the total price, and updates item details.
- `GetOrder(id int)`: Retrieves an order by its ID.
- `UpdateOrder(id int, order data.Order)`: Updates an existing order's details, including recalculating the total price.
- `DeleteOrder(id int)`: Removes an order from the store.
- `GetAllOrders()`: Retrieves all orders in the store.
- `SearchOrders(criteria data.OrderSearchCriteria)`: Filters orders based on search criteria.
- `GetOrdersInTimeRange(start, end time.Time)`: Retrieves orders within a specific time range.

---

## InmemoryAuthorStore.go

This file implements the `AuthorStore` interface using an in-memory data store.

### Structures

#### InMemoryAuthorStore
Represents the in-memory store for authors.
```go
type InMemoryAuthorStore struct {
    mu      sync.RWMutex
    authors map[int]data.Author
    nextID  int
}
```

### Key Methods
- `GetAuthorStoreInstance()`: Returns a singleton instance of `InMemoryAuthorStore`.
- `CreateAuthor(author data.Author)`: Adds a new author to the store.
- `GetAuthor(id int)`: Retrieves an author by its ID.
- `GetAllAuthors()`: Retrieves all authors in the store.
- `UpdateAuthor(id int, author data.Author)`: Updates an author's details.
- `DeleteAuthor(id int)`: Removes an author from the store.
- `SearchAuthors(criteria data.AuthorSearchCriteria)`: Filters authors based on search criteria.
