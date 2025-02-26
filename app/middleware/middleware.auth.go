package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userID"

// AuthMiddleware checks for the Authorization header, validates it,
// and injects the user identifier into the request context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized: no token provided", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token (for demo purposes, use a dummy validation)
		// TODO: validate the token and decode the user id from it.
		userID, err := validateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateToken simulates token validation.
func validateToken(token string) (string, error) {
	// TODO: validate token here
	if token == "secret-token" {
		return "a-user-uuid", nil
	}
	return "", http.ErrNoCookie
}
