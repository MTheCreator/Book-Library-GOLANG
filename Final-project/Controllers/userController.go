package Controllers

import (
	"database/sql"
	"encoding/json"
	"finalProject/StructureData"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// RegisteredUser registers a new user
func RegisteredUser(db *sql.DB, w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var user StructureData.Customer

	// Decode JSON request body
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Hash the password
	if err := user.HashPassword(user.Password); err != nil {
		http.Error(w, `{"error": "failed to hash password"}`, http.StatusInternalServerError)
		return
	}

	// Insert user into the database
	query := "INSERT INTO customers (email, username, password) VALUES ($1, $2, $3) RETURNING id"
	err := db.QueryRow(query, user.Email, user.Username, user.Password).Scan(&user.ID)
	if err != nil {
		http.Error(w, `{"error": "failed to create user"}`, http.StatusInternalServerError)
		return
	}

	// Send response
	response, _ := json.Marshal(map[string]interface{}{
		"userId":   user.ID,
		"email":    user.Email,
		"username": user.Username,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(response)
}
