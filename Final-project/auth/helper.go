package auth

import (
	"context"
	"errors"
	"net/http"
)

var (
	// ErrUnauthorized is returned when a user doesn't have permission to access a resource
	ErrUnauthorized = errors.New("unauthorized: you don't have permission to access this resource")
)

// CanAccessCustomer checks if the user from context can access the specified customer
func CanAccessCustomer(r *http.Request, customerID int) error {
	userClaims, err := GetUserFromContext(r.Context())
	if err != nil {
		return err
	}

	// Users can only access their own data
	if userClaims.ID != customerID {
		return ErrUnauthorized
	}

	return nil
}

// RespondWithUnauthorizedError writes an unauthorized error response
func RespondWithUnauthorizedError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error": "You don't have permission to access this resource"}`))
}

var ErrUserNotFound = errors.New("user not found in context")

// GetUserFromContext retrieves user claims from the context

func GetUserFromContext(ctx context.Context) (*JWTClaim, error) {

	userClaims, ok := ctx.Value("userClaims").(*JWTClaim)

	if !ok {

		return nil, ErrUserNotFound

	}

	return userClaims, nil

}
