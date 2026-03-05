package handler

import (
	"errors"
	"net/http"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

type UserHandler struct {
	userSvc *service.UserService
}

type userResp struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func toUserResp(u *model.User) userResp {
	return userResp{
		ID:        u.ID,
		Username:  u.Username,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	id := auth.UserIDFromContext(r.Context())
	user, err := h.userSvc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	writeJSON(w, http.StatusOK, toUserResp(user))
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.userSvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	out := make([]userResp, 0, len(users))
	for i := range users {
		out = append(out, toUserResp(&users[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Username == "" || body.Password == "" || body.Role == "" {
		writeError(w, http.StatusBadRequest, "username, password, and role are required")
		return
	}
	if len(body.Username) > maxUsernameLen {
		writeError(w, http.StatusBadRequest, "username too long")
		return
	}

	user, err := h.userSvc.Create(r.Context(), body.Username, body.Password, body.Role)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRole):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, auth.ErrPasswordTooShort):
			writeError(w, http.StatusBadRequest, err.Error())
		case repository.IsUniqueConstraintErr(err):
			writeError(w, http.StatusConflict, "username already taken")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toUserResp(user))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var body struct {
		Username *string `json:"username"`
		Role     *string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Username != nil && len(*body.Username) > maxUsernameLen {
		writeError(w, http.StatusBadRequest, "username too long")
		return
	}

	user, err := h.userSvc.Update(r.Context(), id, service.UpdateUserParams{
		Username: body.Username,
		Role:     body.Role,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, service.ErrInvalidRole):
			writeError(w, http.StatusBadRequest, err.Error())
		case repository.IsUniqueConstraintErr(err):
			writeError(w, http.StatusConflict, "username already taken")
		default:
			writeError(w, http.StatusInternalServerError, "failed to update user")
		}
		return
	}

	writeJSON(w, http.StatusOK, toUserResp(user))
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())
	if err := h.userSvc.Delete(r.Context(), id, callerID); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, service.ErrCannotDeleteSelf):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete user")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var body struct {
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	if err := h.userSvc.ResetPassword(r.Context(), id, body.NewPassword); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, auth.ErrPasswordTooShort):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to reset password")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
