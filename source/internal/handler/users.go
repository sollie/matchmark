package handler

import (
	"net/http"

	"github.com/sollie/matchmark/internal/service"
)

type userResponse struct {
	ID                 int    `json:"id"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	Active             bool   `json:"active"`
	CanDownloadBackup  bool   `json:"can_download_backup"`
	MustChangePassword bool   `json:"must_change_password"`
}

func toUserResponse(u service.User) userResponse {
	return userResponse{
		ID:                 u.ID,
		Username:           u.Username,
		Role:               u.Role,
		Active:             u.Active,
		CanDownloadBackup:  u.CanDownloadBackup,
		MustChangePassword: u.MustChangePassword,
	}
}

func (h *Handler) listUsers(w http.ResponseWriter, _ *http.Request) {
	users, err := h.auth.ListUsers()
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]userResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toUserResponse(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	u, err := h.auth.CreateUser(body.Username, body.Role, body.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(u))
}
