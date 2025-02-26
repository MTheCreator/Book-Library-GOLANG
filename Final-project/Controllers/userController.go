package Controllers

import (
	"encoding/json"
	"finalProject/StructureData"
	"finalProject/database"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func RegisteredUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var user StructureData.Customer
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := user.HashPassword(user.Password); err != nil {
		http.Error(w, `{"error": "failed to hash password"}`, http.StatusInternalServerError)
		return
	}

	record := database.Instance.Create(&user)
	if record.Error != nil {
		http.Error(w, `{"error": "failed to create user"}`, http.StatusInternalServerError)
		return
	}

	response, _ := json.Marshal(map[string]interface{}{
		"userId":   user.ID,
		"email":    user.Email,
		"username": user.Username,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(response)
}
