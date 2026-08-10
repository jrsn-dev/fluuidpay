package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// RequireAuth creates a middleware that validates JWTs against a JWKS endpoint.
func RequireAuth(jwksURL string) (func(http.Handler) http.Handler, error) {
	if jwksURL == "" {
		// If no URL is provided, return a mock middleware for testing
		// that bypasses validation and injects a dummy user ID.
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), UserIDKey, "123e4567-e89b-12d3-a456-426614174000")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}, nil
	}

	// Initialize the keyfunc
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid Authorization header", r)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse and validate the token
			token, err := jwt.Parse(tokenStr, jwks.Keyfunc)
			if err != nil || !token.Valid {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token", r)
				return
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid token claims", r)
				return
			}

			// Extract subject (user_id)
			sub, err := claims.GetSubject()
			if err != nil || sub == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Token missing subject claim", r)
				return
			}

			// Add user_id to context
			ctx := context.WithValue(r.Context(), UserIDKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}
