package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	contextKeyUserID   contextKey = "userID"
	contextKeyUsername contextKey = "username"
	contextKeyRole     contextKey = "role"
)

func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			claims, err := VerifyAccessToken(jwtSecret, tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, contextKeyUsername, claims.Username)
			ctx = context.WithValue(ctx, contextKeyRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(contextKeyRole).(string)
		if role != "admin" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(contextKeyUserID).(int64)
	return id
}

func UsernameFromContext(ctx context.Context) string {
	name, _ := ctx.Value(contextKeyUsername).(string)
	return name
}

func RoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(contextKeyRole).(string)
	return role
}
