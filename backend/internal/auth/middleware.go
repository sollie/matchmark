package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/sollie/matchmark/internal/repository"
)

type contextKey string

const (
	contextKeyUserID contextKey = "userID"
	contextKeyRole   contextKey = "role"
)

func RequireAuth(jwtSecret string, users *repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"missing or invalid authorization header"}`))
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			claims, err := VerifyAccessToken(jwtSecret, tokenStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid or expired token"}`))
				return
			}

			user, err := users.GetByID(r.Context(), claims.UserID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				if errors.Is(err, repository.ErrNotFound) {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"invalid or expired token"}`))
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"authentication check failed"}`))
				return
			}
			if user.Role != claims.Role || user.AuthVersion != claims.AuthVer {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid or expired token"}`))
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, contextKeyRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(contextKeyRole).(string)
		if role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(contextKeyUserID).(int64)
	return id
}

func RoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(contextKeyRole).(string)
	return role
}
