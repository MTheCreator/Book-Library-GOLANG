package postgresStores

import (
	"database/sql"
	"fmt"
	"log"

	"finalProject/StructureData"

	_ "github.com/lib/pq"
)

// PostgresSalesReportStore implements persistence for sales reports in PostgreSQL.
type PostgresSalesReportStore struct {
	db *sql.DB
}

var postgresSalesReportStoreInstance *PostgresSalesReportStore

// GetPostgresSalesReportStoreInstance returns a singleton instance.
func GetPostgresSalesReportStoreInstance() *PostgresSalesReportStore {
	if postgresSalesReportStoreInstance == nil {
		connStr := "user=postgres password=root dbname=booklibrary sslmode=disable"
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			panic(fmt.Sprintf("Failed to connect to Postgres for sales reports: %v", err))
		}
		if err := db.Ping(); err != nil {
			panic(fmt.Sprintf("Failed to ping Postgres for sales reports: %v", err))
		}
		postgresSalesReportStoreInstance = &PostgresSalesReportStore{db: db}
		log.Println("Connected to Postgres for sales reports.")
	}
	return postgresSalesReportStoreInstance
}

// Close gracefully closes the underlying DB connection.
func (store *PostgresSalesReportStore) Close() error {
	return store.db.Close()
}

// SaveSalesReport inserts a new sales report and its top selling books into PostgreSQL.
func (store *PostgresSalesReportStore) SaveSalesReport(report StructureData.SalesReport) (*StructureData.SalesReport, *StructureData.ErrorResponse) {
	tx, err := store.db.Begin()
	if err != nil {
		return nil, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to begin transaction: %v", err)}
	}
	defer tx.Rollback()

	// Insert the sales report header.
	reportQuery := `
		INSERT INTO sales_reports (timestamp, total_revenue, total_orders, successful_orders, pending_orders)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var reportID int
	err = tx.QueryRow(reportQuery,
		report.Timestamp,
		report.TotalRevenue,
		report.TotalOrders,
		report.SuccessfulOrders,
		report.PendingOrders,
	).Scan(&reportID)
	if err != nil {
		log.Printf("Error inserting sales report: %v", err)
		return nil, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to insert sales report: %v", err)}
	}
	log.Printf("Inserted sales report with generated ID %d", reportID)

	// Insert each top selling book using the desired column order:
	// (sales_report_id, book_id, quantity_sold, total_revenue, book_title, book_price)
	bookQuery := `
		INSERT INTO top_selling_books 
			(sales_report_id, book_id, quantity_sold, total_revenue, book_title, book_price)
		VALUES ($1, $2, $3, $4, $5, $6)`
	for _, tsb := range report.TopSellingBooks {
		_, err = tx.Exec(bookQuery,
			reportID,
			tsb.Book.ID,
			tsb.QuantitySold,
			tsb.TotalRevenue,
			tsb.Book.Title,
			tsb.Book.Price,
		)
		if err != nil {
			return nil, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to insert top selling book: %v", err)}
		}
		log.Printf("Inserted top selling book for report ID %d, book ID %d", reportID, tsb.Book.ID)
	}

	if err = tx.Commit(); err != nil {
		return nil, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to commit transaction: %v", err)}
	}

	return &report, nil
}

// GetAllSalesReports retrieves all sales reports and their top selling books from PostgreSQL.
func (store *PostgresSalesReportStore) GetAllSalesReports() ([]StructureData.SalesReport, *StructureData.ErrorResponse) {
	const mainQuery = `
		SELECT id, timestamp, total_revenue, total_orders, successful_orders, pending_orders 
		FROM sales_reports`
	rows, err := store.db.Query(mainQuery)
	if err != nil {
		return nil, &StructureData.ErrorResponse{Message: fmt.Sprintf("Failed to fetch sales reports: %v", err)}
	}
	defer rows.Close()

	var reports []StructureData.SalesReport
	for rows.Next() {
		var report StructureData.SalesReport
		var reportID int
		err := rows.Scan(&reportID, &report.Timestamp, &report.TotalRevenue, &report.TotalOrders, &report.SuccessfulOrders, &report.PendingOrders)
		if err != nil {
			log.Printf("Error scanning sales report: %v", err)
			continue
		}

		// Query the top selling books for this report.
		tsbQuery := `
			SELECT book_id, quantity_sold, total_revenue, book_title, book_price
			FROM top_selling_books
			WHERE sales_report_id = $1
			ORDER BY total_revenue DESC`
		tsbRows, err := store.db.Query(tsbQuery, reportID)
		if err != nil {
			log.Printf("Error fetching top selling books for report ID %d: %v", reportID, err)
			reports = append(reports, report)
			continue
		}
		var topSellingBooks []StructureData.TopSellingBook
		for tsbRows.Next() {
			var tsb StructureData.TopSellingBook
			var bookID int
			var quantitySold int
			var totalRevenue float64
			var bookTitle string
			var bookPrice float64

			err = tsbRows.Scan(&bookID, &quantitySold, &totalRevenue, &bookTitle, &bookPrice)
			if err != nil {
				log.Printf("Error scanning top selling book for report ID %d: %v", reportID, err)
				continue
			}

			tsb.QuantitySold = quantitySold
			tsb.TotalRevenue = totalRevenue
			tsb.Book = StructureData.Book{
				ID:    bookID,
				Title: bookTitle,
				Price: bookPrice,
			}
			topSellingBooks = append(topSellingBooks, tsb)
		}
		tsbRows.Close()

		report.TopSellingBooks = topSellingBooks
		reports = append(reports, report)
	}

	return reports, nil
}
