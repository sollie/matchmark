package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) adminRoutes(r chi.Router) {
	r.Post("/shooters", h.createShooter)
	r.Get("/shooters", h.listShooters)
	r.Get("/shooters/{shooterID}", h.getShooter)
	r.Put("/shooters/{shooterID}", h.updateShooter)
	r.Delete("/shooters/{shooterID}", h.deleteShooter)
	r.Post("/competitions", h.createCompetition)
	r.Get("/competitions", h.listCompetitions)
	r.Get("/competitions/{competitionID}", h.getCompetition)
	r.Put("/competitions/{competitionID}", h.updateCompetition)
	r.Delete("/competitions/{competitionID}", h.deleteCompetition)
	r.Post("/competitions/{competitionID}/status", h.setCompetitionStatus)
	r.Get("/competitions/{competitionID}/entries", h.listEntries)
	r.Post("/competitions/{competitionID}/entries", h.addEntry)
	r.Put("/competitions/{competitionID}/entries/{entryID}", h.updateEntry)
	r.Delete("/competitions/{competitionID}/entries/{entryID}", h.deleteEntry)
	r.Post("/competitions/{competitionID}/entries/{entryID}/withdraw", h.withdrawEntry)
	r.Get("/competitions/{competitionID}/lanes/{lane}/score-lane", h.scoreLane)
	r.Put("/competitions/{competitionID}/entries/{entryID}/stages/{stage}", h.setRingCounts)
	r.Get("/competitions/{competitionID}/results.xlsx", h.resultsWorkbook)
	r.Get("/competitions/{competitionID}/results.csv", h.resultsCSV)
	r.Post("/competitions/{competitionID}/entries/{entryID}/stages/{stage}/lock", h.lockStage)
	r.Post("/competitions/{competitionID}/entries/{entryID}/stages/{stage}/unlock", h.unlockStage)
	r.Post("/competitions/{competitionID}/pairings", h.createPairing)
	r.Get("/competitions/{competitionID}/pairings", h.listPairings)
	r.Delete("/competitions/{competitionID}/pairings/{pairingID}", h.revokePairing)
}

type shooterBody struct {
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	Sex       string `json:"sex"`
	Country   string `json:"country"`
	Club      string `json:"club"`
	WAID      string `json:"waid"`
}

func (b shooterBody) shooter(id int64) repository.Shooter {
	return repository.Shooter{ID: id, LastName: b.LastName, FirstName: b.FirstName, Sex: b.Sex, Country: b.Country, Club: b.Club, WAID: b.WAID}
}

func shooterJSON(s repository.Shooter) map[string]any {
	return map[string]any{
		"id": s.ID, "last_name": s.LastName, "first_name": s.FirstName,
		"sex": s.Sex, "country": s.Country, "club": s.Club, "waid": s.WAID,
	}
}

func competitionJSON(c repository.Competition) map[string]any {
	return map[string]any{"id": c.ID, "name": c.Name, "date": c.Date, "course": c.Course, "status": c.Status}
}

func entryJSON(e repository.Entry) map[string]any {
	return map[string]any{
		"id": e.ID, "competition_id": e.CompetitionID, "shooter_id": e.ShooterID,
		"category": e.Category, "lane": e.Lane, "withdrawn": e.Withdrawn,
	}
}

func (h *Handler) listShooters(w http.ResponseWriter, r *http.Request) {
	shooters, err := h.svc.ListShooters(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(shooters))
	for _, s := range shooters {
		out = append(out, shooterJSON(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"shooters": out})
}

func (h *Handler) getShooter(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "shooterID")
	if err != nil {
		writeError(w, err)
		return
	}
	s, err := h.svc.GetShooter(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shooterJSON(s))
}

func (h *Handler) updateShooter(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "shooterID")
	if err != nil {
		writeError(w, err)
		return
	}
	var body shooterBody
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	s := body.shooter(id)
	if err := h.svc.UpdateShooter(r.Context(), s); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shooterJSON(s))
}

func (h *Handler) deleteShooter(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "shooterID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.DeleteShooter(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *Handler) listCompetitions(w http.ResponseWriter, r *http.Request) {
	comps, err := h.svc.ListCompetitions(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(comps))
	for _, c := range comps {
		out = append(out, competitionJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"competitions": out})
}

func (h *Handler) getCompetition(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	c, err := h.svc.GetCompetition(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, competitionJSON(c))
}

func (h *Handler) updateCompetition(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		Name   string `json:"name"`
		Date   string `json:"date"`
		Course string `json:"course"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.UpdateCompetition(r.Context(), repository.Competition{ID: id, Name: body.Name, Date: body.Date, Course: body.Course}); err != nil {
		writeError(w, err)
		return
	}
	c, err := h.svc.GetCompetition(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, competitionJSON(c))
}

func (h *Handler) deleteCompetition(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.DeleteCompetition(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *Handler) setCompetitionStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.SetCompetitionStatus(r.Context(), id, service.CompetitionStatus(body.Status)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": body.Status})
}

func (h *Handler) listEntries(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	entries, err := h.svc.ListEntries(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		m := entryJSON(e.Entry)
		stages := make([]map[string]any, 0, len(e.Stages))
		for _, s := range e.Stages {
			stages = append(stages, map[string]any{"stage_number": s.Number, "expected_shots": s.ExpectedShots, "locked": s.Locked, "scorer_lane": s.ScorerLane, "scorer_name": s.ScorerName})
		}
		m["stages"] = stages
		out = append(out, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}

func (h *Handler) updateEntry(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	entryID, err := pathInt(r, "entryID")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		Category string `json:"category"`
		Lane     int    `json:"lane"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	e, err := h.svc.UpdateEntry(r.Context(), compID, entryID, body.Category, body.Lane)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entryJSON(e))
}

func (h *Handler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	entryID, err := pathInt(r, "entryID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.DeleteEntry(r.Context(), compID, entryID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
