package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	controllers "finalProject/Controllers"
	"finalProject/postgresStores" // Ensure this import path matches your project structure

	"github.com/julienschmidt/httprouter"
)

func initConfig() {
	// In production, load your credentials from environment variables.
	if os.Getenv("DB_USER") == "" || os.Getenv("DB_PASSWORD") == "" || os.Getenv("DB_NAME") == "" || os.Getenv("DB_SSLMODE") == "" {
		log.Println("WARNING: DB configuration not set via environment variables. Falling back to hardcoded connection string.")
	}
}

func closePostgresConnections() {
	// Use the public Close() method from each store.
	if store := postgresStores.GetPostgresCustomerStoreInstance(); store != nil {
		if err := store.Close(); err != nil {
			log.Printf("Error closing customer Postgres connection: %v", err)
		}
	}
	if store := postgresStores.GetPostgresAuthorStoreInstance(); store != nil {
		if err := store.Close(); err != nil {
			log.Printf("Error closing author Postgres connection: %v", err)
		}
	}
	if store := postgresStores.GetPostgresBookStoreInstance(); store != nil {
		if err := store.Close(); err != nil {
			log.Printf("Error closing book Postgres connection: %v", err)
		}
	}
	if store := postgresStores.GetPostgresOrderStoreInstance(); store != nil {
		if err := store.Close(); err != nil {
			log.Printf("Error closing order Postgres connection: %v", err)
		}
	}
}

func main() {
	// Load configuration.
	initConfig()

	// Initialize JSON files and load data into in-memory and PostgreSQL stores.
	controllers.InitializeCustomerFile()
	controllers.InitializeAuthorFile()
	controllers.InitializeBookFile()
	controllers.InitializeOrderFile()

	// Start periodic sales report generation.
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("Generating periodic sales report...")
				controllers.GenerateSalesReport(ctx)
			case <-ctx.Done():
				log.Println("Stopped periodic sales report generation.")
				return
			}
		}
	}()

	// Create a new router.
	router := httprouter.New()

	// Customer Routes
	router.GET("/customers", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.GetAllCustomers(w, r)
	})
	router.GET("/customers/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/customers/" + ps.ByName("id")
		controllers.GetCustomerByID(w, r)
	})
	router.POST("/customers", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.CreateCustomer(w, r)
	})
	router.PUT("/customers/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/customers/" + ps.ByName("id")
		controllers.UpdateCustomer(w, r)
	})
	router.DELETE("/customers/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/customers/" + ps.ByName("id")
		controllers.DeleteCustomer(w, r)
	})
	router.POST("/customers/search", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.SearchCustomers(w, r)
	})

	// Author Routes
	router.GET("/authors", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.GetAllAuthors(w, r)
	})
	router.GET("/authors/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/authors/" + ps.ByName("id")
		controllers.GetAuthorByID(w, r)
	})
	router.POST("/authors", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.CreateAuthor(w, r)
	})
	router.PUT("/authors/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/authors/" + ps.ByName("id")
		controllers.UpdateAuthor(w, r)
	})
	router.DELETE("/authors/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/authors/" + ps.ByName("id")
		controllers.DeleteAuthor(w, r)
	})
	router.POST("/authors/search", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.SearchAuthors(w, r)
	})

	// Book Routes
	router.GET("/books", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.GetAllBooks(w, r)
	})
	router.GET("/books/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/books/" + ps.ByName("id")
		controllers.GetBookByID(w, r)
	})
	router.POST("/books", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.CreateBook(w, r)
	})
	router.PUT("/books/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/books/" + ps.ByName("id")
		controllers.UpdateBook(w, r)
	})
	router.DELETE("/books/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/books/" + ps.ByName("id")
		controllers.DeleteBook(w, r)
	})
	router.POST("/books/search", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.SearchBooks(w, r)
	})

	// Order Routes
	router.GET("/orders", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.GetAllOrders(w, r)
	})
	router.GET("/orders/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/orders/" + ps.ByName("id")
		controllers.GetOrderByID(w, r)
	})
	router.POST("/orders", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.CreateOrder(w, r)
	})
	router.PUT("/orders/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/orders/" + ps.ByName("id")
		controllers.UpdateOrder(w, r)
	})
	router.DELETE("/orders/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Path = "/orders/" + ps.ByName("id")
		controllers.DeleteOrder(w, r)
	})
	router.POST("/orders/search", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		controllers.SearchOrders(w, r)
	})

	// Reports Routes
	router.GET("/reports/sales", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		controllers.GetSalesReport(ctx, w, r)
	})
	router.POST("/reports/sales/generate", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		controllers.GenerateSalesReport(ctx)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Sales report generated successfully"))
	})

	// Create and start the HTTP server.
	server := &http.Server{Addr: ":8080", Handler: router}
	go func() {
		log.Println("Starting server on :8080...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for termination signal to gracefully shut down.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	// Close PostgreSQL connections gracefully.
	closePostgresConnections()

	log.Println("Server exited gracefully.")
}
