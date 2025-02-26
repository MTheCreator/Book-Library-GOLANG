package Controllers

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func Ping(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	response, _ := json.Marshal(map[string]string{"message": "pong"})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}
