package handler

import (
	"net/http"

	"github.com/sollie/matchmark/internal/repository"
)

type competitionBody struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (b competitionBody) competition(id int64) repository.Competition {
	return repository.Competition{ID: id, Name: b.Name, StartDate: b.StartDate, EndDate: b.EndDate}
}

func competitionJSON(c repository.Competition) map[string]any {
	return map[string]any{"id": c.ID, "name": c.Name, "start_date": c.StartDate, "end_date": c.EndDate, "match_count": c.MatchCount}
}

func (h *Handler) listCompetitions(r *http.Request) (int, any, error) {
	cs, err := h.repo.ListCompetitions(r.Context())
	if err != nil {
		return fail(err)
	}
	out := make([]map[string]any, 0, len(cs))
	for _, c := range cs {
		out = append(out, competitionJSON(c))
	}
	return okJSON(map[string]any{"competitions": out})
}

func (h *Handler) createCompetition(r *http.Request) (int, any, error) {
	req := in(r)
	var body competitionBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	id, err := h.svc.CreateCompetition(r.Context(), body.competition(0))
	if err != nil {
		return fail(err)
	}
	return http.StatusCreated, map[string]int64{"id": id}, nil
}

func (h *Handler) getCompetition(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("competitionID")
	if req.err != nil {
		return fail(req.err)
	}
	c, err := h.repo.GetCompetition(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(competitionJSON(c))
}

func (h *Handler) updateCompetition(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("competitionID")
	var body competitionBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.UpdateCompetition(r.Context(), body.competition(id)); err != nil {
		return fail(err)
	}
	c, err := h.repo.GetCompetition(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(competitionJSON(c))
}

func (h *Handler) deleteCompetition(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("competitionID")
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.DeleteCompetition(r.Context(), id); err != nil {
		return fail(err)
	}
	return okJSON(map[string]bool{"deleted": true})
}

func (h *Handler) competitionDownload(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	file, err := h.svc.CompetitionDownload(r.Context(), id, r.URL.Query().Get("format"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeRendered(w, "application/zip", "zip", file, true)
}

func (h *Handler) competitionWorkbook1500(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	file, err := h.svc.CompetitionWorkbook1500(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRendered(w, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx", file, true)
}
