package handler

import (
	"net/http"

	"github.com/sollie/matchmark/internal/repository"
)

func rangeJSON(g repository.Range) map[string]any {
	return map[string]any{"id": g.ID, "name": g.Name, "archived": g.Archived}
}

func (h *Handler) listRanges(r *http.Request) (int, any, error) {
	rs, err := h.svc.ListRanges(r.Context())
	if err != nil {
		return fail(err)
	}
	out := make([]map[string]any, 0, len(rs))
	for _, g := range rs {
		out = append(out, rangeJSON(g))
	}
	return okJSON(map[string]any{"ranges": out})
}

func (h *Handler) createRange(r *http.Request) (int, any, error) {
	req := in(r)
	var body struct {
		Name string `json:"name"`
	}
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	id, err := h.svc.CreateRange(r.Context(), body.Name)
	if err != nil {
		return fail(err)
	}
	return http.StatusCreated, map[string]int64{"id": id}, nil
}

func (h *Handler) updateRange(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("rangeID")
	var body struct {
		Name string `json:"name"`
	}
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.UpdateRange(r.Context(), id, body.Name); err != nil {
		return fail(err)
	}
	return h.rangeResponse(r, id)
}

func (h *Handler) archiveRange(r *http.Request) (int, any, error) {
	return h.setRangeArchived(r, true)
}

func (h *Handler) unarchiveRange(r *http.Request) (int, any, error) {
	return h.setRangeArchived(r, false)
}

func (h *Handler) setRangeArchived(r *http.Request, archived bool) (int, any, error) {
	req := in(r)
	id := req.path("rangeID")
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.SetRangeArchived(r.Context(), id, archived); err != nil {
		return fail(err)
	}
	return h.rangeResponse(r, id)
}

func (h *Handler) deleteRange(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("rangeID")
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.DeleteRange(r.Context(), id); err != nil {
		return fail(err)
	}
	return okJSON(map[string]bool{"deleted": true})
}

func (h *Handler) rangeResponse(r *http.Request, id int64) (int, any, error) {
	g, err := h.svc.GetRange(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(rangeJSON(g))
}
