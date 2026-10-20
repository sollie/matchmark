package handler

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sollie/matchmark/internal/service"
)

const sessionCookie = "mm_session"

type Option func(*Handler)

func WithAuth(auth *service.Auth, baseURL string) Option {
	return func(h *Handler) {
		h.auth = auth
		h.secureCookie = strings.HasPrefix(strings.ToLower(baseURL), "https://")
	}
}

func WithAudit(a *service.Audit) Option {
	return func(h *Handler) { h.audit = a }
}

type userContextKey struct{}

func (h *Handler) RequireUser(next http.Handler) http.Handler {
	return h.requireUser(next, false)
}

func (h *Handler) RequireAdmin(next http.Handler) http.Handler {
	return h.requireUser(next, true)
}

func currentUser(r *http.Request) service.User {
	u, _ := r.Context().Value(userContextKey{}).(service.User)
	return u
}

func (h *Handler) requireUser(next http.Handler, adminOnly bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			writeUnauthorized(w)
			return
		}
		user, err := h.auth.Authenticate(c.Value)
		if err != nil {
			if errors.Is(err, service.ErrUnauthenticated) {
				writeUnauthorized(w)
				return
			}
			writeError(w, err)
			return
		}
		if user.MustChangePassword {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "password_change_required"})
			return
		}
		if adminOnly && user.Role != service.RoleAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey{}, user)
		ctx = service.WithActor(ctx, service.Actor{UserID: user.ID, Username: user.Username, IP: clientKey(r)})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}

func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, value string, expires time.Time, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) adminLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	token, expires, err := h.auth.Login(clientKey(r), body.Username, body.Password)
	switch {
	case errors.Is(err, service.ErrTooManyAttempts):
		w.Header().Set("Retry-After", "900")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many login attempts"})
		return
	case errors.Is(err, service.ErrInvalidCredentials):
		writeUnauthorized(w)
		return
	case err != nil:
		writeError(w, err)
		return
	}
	h.setSessionCookie(w, token, expires, int(service.SessionTTL.Seconds()))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := h.auth.Logout(c.Value); err != nil {
			writeError(w, err)
			return
		}
	}
	h.setSessionCookie(w, "", time.Unix(0, 0), -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminSession(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	writeJSON(w, http.StatusOK, map[string]any{"username": u.Username, "role": u.Role, "can_download_backup": u.CanDownloadBackup})
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		writeUnauthorized(w)
		return
	}
	err = h.auth.ChangePassword(service.WithActor(r.Context(), service.Actor{IP: clientKey(r)}), c.Value, body.CurrentPassword, body.NewPassword)
	switch {
	case errors.Is(err, service.ErrTooManyAttempts):
		w.Header().Set("Retry-After", "900")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts"})
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrUnauthenticated):
		writeUnauthorized(w)
	case errors.Is(err, service.ErrWeakPassword):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": service.ErrWeakPassword.Error()})
	case err != nil:
		writeError(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
