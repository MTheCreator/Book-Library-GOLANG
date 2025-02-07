Here is a detailed markdown documentation for the provided project files:

---

# Project Documentation

## Address.go

This file defines the `Address` structure and associated search criteria.

### Structures

#### Address
Represents an address with fields for street, city, state, postal code, and country.
```go
type Address struct {
    Street     string `json:"street"`
    City       string `json:"city"`
    State      string `json:"state"`
    PostalCode string `json:"postal_code"`
    Country    string `json:"country"`
}
```

#### AddressSearchCriteria
Facilitates filtering of addresses based on various fields like street, city, and state.
```go
type AddressSearchCriteria struct {
    Streets     []string `json:"streets,omitempty"`
    Cities      []string `json:"cities,omitempty"`
    States      []string `json:"states,omitempty"`
    PostalCodes []string `json:"postal_codes,omitempty"`
    Countries   []string `json:"countries,omitempty"`
}
```

---

## Author.go

Defines the `Author` structure for representing authors and their search criteria.

### Structures

#### Author
Represents an author with attributes such as ID, first name, last name, and bio.
```go
type Author struct {
    ID        int    `json:"id"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
    Bio       string `json:"bio"`
}
```

#### AuthorSearchCriteria
Supports searching for authors by ID, name, or keywords in their bio.
```go
type AuthorSearchCriteria struct {
    IDs         []int    `json:"ids,omitempty"`
    FirstNames  []string `json:"first_names,omitempty"`
    LastNames   []string `json:"last_names,omitempty"`
    Keywords    []string `json:"keywords,omitempty"`
}
```

---

## Book.go

Defines the `Book` structure and search criteria for managing books.

### Structures

#### Book
Represents a book with fields for ID, title, author, genres, publication date, price, and stock.
```go
type Book struct {
    ID          int       `json:"id"`
    Title       string    `json:"title"`
    Author      Author    `json:"author"`
    Genres      []string  `json:"genres"`
    PublishedAt time.Time `json:"published_at"`
    Price       float64   `json:"price"`
    Stock       int       `json:"stock"`
}
```

#### BookSearchCriteria
Facilitates filtering of books based on IDs, titles, genres, price, stock, and publication dates.
```go
type BookSearchCriteria struct {
    IDs            []int       `json:"ids,omitempty"`
    Titles         []string    `json:"titles,omitempty"`
    Genres         []string    `json:"genres,omitempty"`
    MinPublishedAt time.Time   `json:"min_published_at,omitempty"`
    MaxPublishedAt time.Time   `json:"max_published_at,omitempty"`
    MinPrice       float64     `json:"min_price,omitempty"`
    MaxPrice       float64     `json:"max_price,omitempty"`
    MinStock       int         `json:"min_stock,omitempty"`
    MaxStock       int         `json:"max_stock,omitempty"`
    AuthorCriteria AuthorSearchCriteria `json:"author_criteria,omitempty"`
}
```

---

## Error.go

Defines the `ErrorResponse` structure for handling errors.

### Structures

#### ErrorResponse
Represents an error message for API responses.
```go
type ErrorResponse struct {
    Message string `json:"error"`
}
```

Provides an `Error()` method for error string representation:
```go
func (e *ErrorResponse) Error() string {
    return e.Message
}
```

---

## Order.go

Defines the `Order` structure and related search criteria for managing customer orders.

### Structures

#### Order
Represents an order with customer details, items, total price, and creation date.
```go
type Order struct {
    ID         int         `json:"id"`
    Customer   Customer    `json:"customer"`
    Items      []OrderItem `json:"items"`
    TotalPrice float64     `json:"total_price"`
    CreatedAt  time.Time   `json:"created_at"`
}
```

#### OrderSearchCriteria
Supports filtering orders by ID, customer, price, and creation date.
```go
type OrderSearchCriteria struct {
    IDs             []int                 `json:"ids,omitempty"`
    CustomerIDs     []int                 `json:"customer_ids,omitempty"`
    MinTotalPrice   float64               `json:"min_total_price,omitempty"`
    MaxTotalPrice   float64               `json:"max_total_price,omitempty"`
    MinCreatedAt    time.Time             `json:"min_created_at,omitempty"`
    MaxCreatedAt    time.Time             `json:"max_created_at,omitempty"`
    ItemCriteria    OrderItemSearchCriteria `json:"item_criteria,omitempty"`
}
```

---

## OrderItem.go

Defines the `OrderItem` structure and search criteria for individual order items.

### Structures

#### OrderItem
Represents a book and its quantity in an order.
```go
type OrderItem struct {
    Book     Book `json:"book"`
    Quantity int  `json:"quantity"`
}
```

#### OrderItemSearchCriteria
Facilitates filtering of order items based on book criteria or quantity.
```go
type OrderItemSearchCriteria struct {
    BookCriteria BookSearchCriteria `json:"book_criteria,omitempty"`
    MinQuantity  int                `json:"min_quantity,omitempty"`
    MaxQuantity  int                `json:"max_quantity,omitempty"`
}
```

---

## SalesReport.go

Defines structures for generating and searching sales reports.

### Structures

#### SalesReport
Represents a sales report with details about revenue, orders, and top-selling books.
```go
type SalesReport struct {
    Timestamp       time.Time        `json:"timestamp"`
    TotalRevenue    float64          `json:"total_revenue"`
    TotalOrders     int              `json:"total_orders"`
    TopSellingBooks []TopSellingBook `json:"top_selling_books"`
}
```

#### TopSellingBook
Contains details of a top-selling book and its sold quantity.
```go
type TopSellingBook struct {
    Book         Book `json:"book"`
    QuantitySold int  `json:"quantity_sold"`
}
```

#### SalesReportSearchCriteria
Enables filtering of sales reports based on timestamp, revenue, and orders.
```go
type SalesReportSearchCriteria struct {
    MinTimestamp     time.Time               `json:"min_timestamp,omitempty"`
    MaxTimestamp     time.Time               `json:"max_timestamp,omitempty"`
    MinRevenue       float64                 `json:"min_revenue,omitempty"`
    MaxRevenue       float64                 `json:"max_revenue,omitempty"`
    MinOrders        int                     `json:"min_orders,omitempty"`
    MaxOrders        int                     `json:"max_orders,omitempty"`
    TopBooksCriteria BookSalesSearchCriteria `json:"top_books_criteria,omitempty"`
}
```

---

## Customer.go

Defines the `Customer` structure and associated search criteria.

### Structures

#### Customer
Represents a customer with fields for ID, name, email, address, and creation date.
```go
type Customer struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Address   Address   `json:"address"`
    CreatedAt time.Time `json:"created_at"`
}
```

#### CustomerSearchCriteria
Supports filtering customers based on ID, name, email, and creation date.
```go
type CustomerSearchCriteria struct {
    IDs             []int                `json:"ids,omitempty"`
    Names           []string             `json:"names,omitempty"`
    Emails          []string             `json:"emails,omitempty"`
    MinCreatedAt    time.Time            `json:"min_created_at,omitempty"`
    MaxCreatedAt    time.Time            `json:"max_created_at,omitempty"`
    AddressCriteria AddressSearchCriteria `json:"address_criteria,omitempty"`
}
```

--- 

This documentation provides a clear and structured overview of the project's core components. 