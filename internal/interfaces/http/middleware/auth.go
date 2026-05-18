package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/lidar-platform/backend/internal/application/usecases"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(auth *usecases.AuthUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}

			userID, err := auth.Validate(r.Context(), parts[1])
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
