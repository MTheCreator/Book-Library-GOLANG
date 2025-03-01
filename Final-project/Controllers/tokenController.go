package Controllers

import (
	"database/sql"
	"encoding/json"
	"finalProject/StructureData"
	"finalProject/auth"
	postgresStores "finalProject/postgresStores"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type TokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// GenerateToken authenticates the user and generates a JWT token
func GenerateToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	store := postgresStores.GetPostgresCustomerStoreInstance()

	var request TokenRequest
	var user StructureData.Customer
	// Decode JSON request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Query user from the database
	query := "SELECT id, email, username, password FROM customers WHERE email = $1"
	row := store.DB.QueryRow(query, request.Email)
	err := row.Scan(&user.ID, &user.Email, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "user not found"}`, http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}

	// Check password
	if err := user.CheckPassword(request.Password); err != nil {
		http.Error(w, `{"error": "invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	tokenString, err := auth.GenerateJWT(user.ID, user.Email, user.Username)
	if err != nil {
		http.Error(w, `{"error": "failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	// Send response
	response, _ := json.Marshal(map[string]string{"token": tokenString})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}
