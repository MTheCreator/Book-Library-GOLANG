package Controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"finalProject/StructureData"
	"finalProject/postgresStores"
)

// CreateReview handles POST /reviews.
// It decodes the review input, sets CreatedAt to the current time,
// then creates the review using the PostgreSQL ReviewStore.
func CreateReview(w http.ResponseWriter, r *http.Request) {
	reviewStore := postgresStores.GetPostgresReviewStoreInstance()

	var review StructureData.Review
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid review input"})
		return
	}

	// Overwrite CreatedAt with current time
	review.CreatedAt = time.Now()

	createdReview, errResp := reviewStore.CreateReview(review)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdReview)
}

// GetReviewsByBook handles GET /reviews?book_id=1.
// It retrieves all reviews for a given book.
func GetReviewsByBook(w http.ResponseWriter, r *http.Request) {
	reviewStore := postgresStores.GetPostgresReviewStoreInstance()

	bookIDStr := r.URL.Query().Get("book_id")
	if bookIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "book_id is required"})
		return
	}
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid book_id"})
		return
	}

	reviews, errResp := reviewStore.GetReviewsByBookID(bookID)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}



// DeleteReview handles DELETE /reviews/:id.
// It deletes the review by ID.
func DeleteReview(w http.ResponseWriter, r *http.Request) {
	reviewStore := postgresStores.GetPostgresReviewStoreInstance()

	idStr := r.URL.Path[len("/reviews/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(StructureData.ErrorResponse{Message: "Invalid review ID"})
		return
	}

	errResp := reviewStore.DeleteReview(id)
	if errResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
