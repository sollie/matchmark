package handler

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sollie/matchmark/internal/service"
)

const sessionCookie = "mm_session"

type Option func(*Handler)

func WithAdminAuth(auth *service.AdminAuth, baseURL string) Option {
	return func(h *Handler) {
		h.auth = auth
		h.secureCookie = strings.HasPrefix(strings.ToLower(baseURL), "https://")
	}
}

func (h *Handler) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			writeUnauthorized(w)
			return
		}
		if err := h.auth.Authenticate(c.Value); err != nil {
			if errors.Is(err, service.ErrUnauthenticated) {
				writeUnauthorized(w)
				return
			}
			writeError(w, err)
			return
		}
		next.ServeHTTP(w, r)
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
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	token, expires, err := h.auth.Login(clientKey(r), body.Password)
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

func (h *Handler) adminSession(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
