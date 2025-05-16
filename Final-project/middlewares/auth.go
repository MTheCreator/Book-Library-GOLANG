package middlewares

import (
	"context"
	"finalProject/auth"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

// UserContext is the key used to store and retrieve user information from the request context
type UserContext string

const UserContextKey UserContext = "user"

// UserClaims holds the user information extracted from JWT
type UserClaims struct {
	UserID   int
	Email    string
	Username string
	Role     string
}

// Auth middleware validates the JWT token and adds user claims to the request context
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

		// Extract claims from token
		claims, err := extractClaims(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		// Add user information to request context
		userClaims := UserClaims{
			UserID:   claims.ID,
			Email:    claims.Email,
			Username: claims.Username,
			Role:     claims.Role,
		}

		// Create a new context with user information
		ctx := context.WithValue(r.Context(), UserContextKey, userClaims)

		// Call next handler with updated context
		next(w, r.WithContext(ctx))
	}
}

// extractClaims parses the token and returns the JWT claims
func extractClaims(tokenString string) (*auth.JWTClaim, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&auth.JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(auth.GetJWTKey()), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*auth.JWTClaim)
	if !ok {
		return nil, fmt.Errorf("couldn't parse claims")
	}

	// Validate token expiration
	if err := auth.ValidateToken(tokenString); err != nil {
		return nil, err
	}

	return claims, nil
}

// GetUserFromContext extracts user claims from the request context
func GetUserFromContext(ctx context.Context) (UserClaims, error) {
	user, ok := ctx.Value(UserContextKey).(UserClaims)
	if !ok {
		return UserClaims{}, fmt.Errorf("user information not found in context")
	}
	return user, nil
}

// AuthorizeAdmin ensures only users with role "admin" can access the route
func AuthorizeAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := GetUserFromContext(r.Context())
		if err != nil {
			http.Error(w, `{"error": "user not found in context"}`, http.StatusUnauthorized)
			return
		}

		if user.Role != "admin" {
			http.Error(w, `{"error": "forbidden: admin access required"}`, http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
