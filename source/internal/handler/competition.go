package handler

import (
	"context"
	"net/http"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

type competitionBody struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`

	RangeID service.OptionalID `json:"range_id"`

	RefereeWAIDs      *[]string `json:"referee_waids"`
	RangeOfficerWAIDs *[]string `json:"range_officer_waids"`
}

func (b competitionBody) competition(id int64, current repository.Competition, defaultReferee string) repository.Competition {
	c := repository.Competition{ID: id, Name: b.Name, StartDate: b.StartDate, EndDate: b.EndDate,
		Referees: current.Referees, RangeOfficers: current.RangeOfficers}
	if b.RefereeWAIDs != nil {
		c.Referees = officialsFromWAIDs(*b.RefereeWAIDs)
	} else if defaultReferee != "" {
		c.Referees = officialsFromWAIDs([]string{defaultReferee})
	}
	if b.RangeOfficerWAIDs != nil {
		c.RangeOfficers = officialsFromWAIDs(*b.RangeOfficerWAIDs)
	}
	return c
}

func competitionJSON(c repository.Competition) map[string]any {
	return map[string]any{"id": c.ID, "name": c.Name, "start_date": c.StartDate, "end_date": c.EndDate, "match_count": c.MatchCount, "range_id": c.RangeID, "range_name": nullable(c.RangeName),
		"referees": officialsJSON(c.Referees), "range_officers": officialsJSON(c.RangeOfficers)}
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
	id, err := h.svc.CreateCompetition(r.Context(), body.competition(0, repository.Competition{}, currentUser(r).WAID), body.RangeID)
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
	current, err := h.repo.GetCompetition(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	if err := h.svc.UpdateCompetition(r.Context(), body.competition(id, current, ""), body.RangeID); err != nil {
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
	h.competitionFile(w, r, "zip", "application/zip", func(ctx context.Context, id int64) (service.Download, error) {
		return h.svc.CompetitionDownload(ctx, id, r.URL.Query().Get("format"))
	})
}

func (h *Handler) competitionWorkbook1500(w http.ResponseWriter, r *http.Request) {
	h.competitionFile(w, r, "xlsx", xlsxContentType, h.svc.CompetitionWorkbook1500)
}

func (h *Handler) competitionFile(w http.ResponseWriter, r *http.Request, ext, contentType string, build func(ctx context.Context, id int64) (service.Download, error)) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	file, err := build(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRendered(w, contentType, ext, file, true)
}
