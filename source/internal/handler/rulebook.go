package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type rulebookStageBody struct {
	StageNumber   int    `json:"stage_number"`
	Distance      string `json:"distance"`
	ExpectedShots int    `json:"expected_shots"`
	TimeSeconds   int    `json:"time_seconds"`
	Position      string `json:"position"`
}

type rulebookCourseBody struct {
	Key        string              `json:"key"`
	Label      string              `json:"label"`
	Discipline string              `json:"discipline"`
	TotalShots int                 `json:"total_shots"`
	Categories []string            `json:"categories"`
	Stages     []rulebookStageBody `json:"stages"`
}

type rulebookRingBody struct {
	Ring   string `json:"ring"`
	Points int    `json:"points"`
}

type rulebookBody struct {
	Courses       []rulebookCourseBody `json:"courses"`
	WeaponClasses map[string][]string  `json:"weapon_classes"`
	Rings         []rulebookRingBody   `json:"rings"`
}

type versionBody struct {
	Version string `json:"version"`
}

func (h *Handler) getVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, versionBody{Version: h.version})
}

func (h *Handler) rulebookRoutes(r chi.Router) {
	r.Get("/rulebook", h.getRulebook)
}

func (h *Handler) getRulebook(w http.ResponseWriter, _ *http.Request) {
	view, err := h.svc.Rulebook()
	if err != nil {
		writeError(w, err)
		return
	}
	body := rulebookBody{
		Courses:       make([]rulebookCourseBody, 0, len(view.Courses)),
		WeaponClasses: view.WeaponClasses,
		Rings:         make([]rulebookRingBody, 0, len(view.Rings)),
	}
	for _, c := range view.Courses {
		course := rulebookCourseBody{Key: c.Key, Label: c.Label, Discipline: c.Discipline, TotalShots: c.TotalShots, Categories: c.Categories, Stages: make([]rulebookStageBody, 0, len(c.Stages))}
		for _, st := range c.Stages {
			course.Stages = append(course.Stages, rulebookStageBody(st))
		}
		body.Courses = append(body.Courses, course)
	}
	for _, ring := range view.Rings {
		body.Rings = append(body.Rings, rulebookRingBody(ring))
	}
	writeJSON(w, http.StatusOK, body)
}
