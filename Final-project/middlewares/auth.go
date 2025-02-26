package middlewares

import (
	"finalProject/auth"
	"fmt"
	"net/http"
	"strings"
)

func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, `{"error": "request does not contain an access token"}`, http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix if it exists
		tokenParts := strings.Split(tokenString, " ")
		if len(tokenParts) == 2 {
			tokenString = tokenParts[1]
		}

		err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		next(w, r) // Call the next handler
	}
}
