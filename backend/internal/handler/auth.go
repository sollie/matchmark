package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/sollie/matchmark/internal/service"
)

const refreshCookieName = "mm_rt"

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type loginUserResp struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type loginResp struct {
	AccessToken string        `json:"access_token"`
	ExpiresIn   int           `json:"expires_in"`
	User        loginUserResp `json:"user"`
}

type refreshResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func setRefreshCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Username == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	result, err := h.authSvc.Login(r.Context(), body.Username, body.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	setRefreshCookie(w, result.RefreshToken, result.ExpiresIn)
	writeJSON(w, http.StatusOK, loginResp{
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
		User: loginUserResp{
			ID:       result.User.ID,
			Username: result.User.Username,
			Role:     result.User.Role,
		},
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	result, err := h.authSvc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrTokenExpired) {
			setRefreshCookie(w, "", -1)
			writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		writeError(w, http.StatusInternalServerError, "refresh failed")
		return
	}

	setRefreshCookie(w, result.RefreshToken, result.ExpiresIn)
	writeJSON(w, http.StatusOK, refreshResp{
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.authSvc.Logout(r.Context(), cookie.Value); err != nil {
		setRefreshCookie(w, "", -1)
		writeError(w, http.StatusInternalServerError, "logout failed")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}
