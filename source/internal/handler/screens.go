package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) screenPublicRoutes(r chi.Router) {
	r.Post("/spectator/screens", h.registerScreen)
	r.Delete("/spectator/screens/self", h.leaveScreen)
}

func (h *Handler) screenAdminRoutes(r chi.Router) {
	r.Get("/screens", h.listScreens)
	r.Post("/screens/approve", h.approveScreen)
	r.Put("/screens/assignment", h.assignAllScreens)
	r.Get("/screens/assignable-matches", h.assignableMatches)
	r.Put("/screens/{screenID}", h.updateScreen)
	r.Delete("/screens/{screenID}", h.removeScreen)
}

func screenJSON(s service.Screen) map[string]any {
	return map[string]any{
		"id": s.ID, "name": s.Name, "match_id": s.MatchID, "online": s.Online,
		"last_seen_at": s.LastSeenAt.UTC().Format(time.RFC3339),
	}
}

func (h *Handler) registerScreen(w http.ResponseWriter, r *http.Request) {
	reg, err := h.svc.RegisterScreen(r.Context(), clientKey(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"token": reg.Token, "code": reg.Code})
}

func (h *Handler) leaveScreen(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.LeaveScreen(r.Context(), bearerToken(r)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listScreens(w http.ResponseWriter, r *http.Request) {
	screens, err := h.svc.ListScreens(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(screens))
	for _, s := range screens {
		out = append(out, screenJSON(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"screens": out})
}

func (h *Handler) approveScreen(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	s, err := h.svc.ApproveScreen(r.Context(), body.Code, body.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, screenJSON(s))
}

func (h *Handler) updateScreen(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "screenID")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		Name    *string         `json:"name"`
		MatchID json.RawMessage `json:"match_id"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	update := service.ScreenUpdate{Name: body.Name}
	if body.MatchID != nil {
		update.SetMatch = true
		if err := json.Unmarshal(body.MatchID, &update.MatchID); err != nil {
			writeError(w, service.ErrInvalid)
			return
		}
	}
	s, err := h.svc.UpdateScreen(r.Context(), id, update)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, screenJSON(s))
}

func (h *Handler) assignAllScreens(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MatchID int64 `json:"match_id"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	n, err := h.svc.AssignAllScreens(r.Context(), body.MatchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"assigned": n})
}

func (h *Handler) removeScreen(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "screenID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.RemoveScreen(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) assignableMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := h.svc.AssignableMatches(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		out = append(out, map[string]any{"id": m.ID, "name": m.Name, "date": m.Date, "course": m.Course, "status": m.Status})
	}
	writeJSON(w, http.StatusOK, map[string]any{"matches": out})
}
