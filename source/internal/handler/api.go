package handler

import (
	"net/http"

	"github.com/sollie/matchmark/internal/repository"
)

func (h *Handler) createShooter(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LastName  string `json:"last_name"`
		FirstName string `json:"first_name"`
		Sex       string `json:"sex"`
		Country   string `json:"country"`
		Club      string `json:"club"`
		WAID      string `json:"waid"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	id, err := h.svc.CreateShooter(r.Context(), repository.Shooter{
		LastName: body.LastName, FirstName: body.FirstName, Sex: body.Sex,
		Country: body.Country, Club: body.Club, WAID: body.WAID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *Handler) createCompetition(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Date   string `json:"date"`
		Course string `json:"course"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	id, err := h.svc.CreateCompetition(r.Context(), repository.Competition{Name: body.Name, Date: body.Date, Course: body.Course})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *Handler) addEntry(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		ShooterID int64  `json:"shooter_id"`
		Category  string `json:"category"`
		Lane      int    `json:"lane"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	id, err := h.svc.AddEntry(r.Context(), repository.Entry{
		CompetitionID: compID, ShooterID: body.ShooterID, Category: body.Category, Lane: body.Lane,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *Handler) withdrawEntry(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.WithdrawEntry(r.Context(), compID, entryID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"withdrawn": true})
}

func (h *Handler) scoreLane(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	lane, err := pathInt(r, "lane")
	if err != nil {
		writeError(w, err)
		return
	}
	scoreLane, err := h.svc.ScoreLane(r.Context(), compID, int(lane))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"score_lane": scoreLane})
}

func (h *Handler) setRingCounts(w http.ResponseWriter, r *http.Request) {
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
	stage, err := pathInt(r, "stage")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		X     int `json:"x"`
		Ten   int `json:"10"`
		Nine  int `json:"9"`
		Eight int `json:"8"`
		Seven int `json:"7"`
		Zero  int `json:"0"`
		Miss  int `json:"miss"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	counts := repository.RingCounts{X: body.X, Ten: body.Ten, Nine: body.Nine, Eight: body.Eight, Seven: body.Seven, Zero: body.Zero, Miss: body.Miss}
	if err := h.svc.SetRingCounts(r.Context(), compID, entryID, int(stage), counts); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}
