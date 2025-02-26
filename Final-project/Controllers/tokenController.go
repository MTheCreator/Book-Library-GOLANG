package Controllers

import (
	"encoding/json"
	"finalProject/StructureData"
	"finalProject/auth"
	"finalProject/database"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type TokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func GenerateToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request TokenRequest
	var user StructureData.Customer

	// Decode JSON request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Find user in database
	record := database.Instance.Where("email = ?", request.Email).First(&user)
	if record.Error != nil {
		http.Error(w, `{"error": "user not found"}`, http.StatusUnauthorized)
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
