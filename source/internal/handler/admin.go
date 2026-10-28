package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) adminRoutes(r chi.Router) {
	r.Post("/shooters", serve(h.createShooter))
	r.Post("/shooters/import", h.importShooter)
	r.Get("/shooters", serve(h.listShooters))
	r.Get("/shooters/{shooterID}", serve(h.getShooter))
	r.Put("/shooters/{shooterID}", serve(h.updateShooter))
	r.Delete("/shooters/{shooterID}", serve(h.deleteShooter))
	r.Post("/shooters/{shooterID}/archive", serve(h.archiveShooter))
	r.Post("/shooters/{shooterID}/unarchive", serve(h.unarchiveShooter))
	r.Get("/shooters/{shooterID}/merge-preview", serve(h.previewShooterMerge))
	r.Post("/shooters/{shooterID}/merge", serve(h.mergeShooter))
	r.Get("/competitions", serve(h.listCompetitions))
	r.Post("/competitions", serve(h.createCompetition))
	r.Get("/competitions/{competitionID}", serve(h.getCompetition))
	r.Put("/competitions/{competitionID}", serve(h.updateCompetition))
	r.Delete("/competitions/{competitionID}", serve(h.deleteCompetition))
	r.With(h.render.MiddlewareUncoalesced(ResourceByPath)).Get("/competitions/{competitionID}/results.zip", h.competitionDownload)
	r.Post("/matches", serve(h.createMatch))
	r.Get("/matches", serve(h.listMatches))
	r.Get("/matches/{matchID}", serve(h.getMatch))
	r.Put("/matches/{matchID}", serve(h.updateMatch))
	r.Delete("/matches/{matchID}", serve(h.deleteMatch))
	r.Post("/matches/{matchID}/status", serve(h.setMatchStatus))
	r.Post("/matches/{matchID}/next-stage", serve(h.startNextStage))
	r.Get("/matches/{matchID}/control-room", serve(h.controlRoom))
	r.Post("/matches/{matchID}/accept-all", serve(h.acceptAll))
	r.Get("/matches/{matchID}/stage-events", serve(h.stageEvents))
	r.Get("/matches/{matchID}/entries", serve(h.listEntries))
	r.Post("/matches/{matchID}/entries", serve(h.addEntry))
	r.Put("/matches/{matchID}/entries/{entryID}", serve(h.updateEntry))
	r.Delete("/matches/{matchID}/entries/{entryID}", serve(h.deleteEntry))
	r.Post("/matches/{matchID}/entries/{entryID}/withdraw", serve(h.withdrawEntry))
	r.Get("/matches/{matchID}/lanes/{lane}/score-lane", serve(h.scoreLane))
	r.Put("/matches/{matchID}/entries/{entryID}/stages/{stage}", serve(h.setRingCounts))
	r.Get("/matches/{matchID}/results.xlsx", h.resultsWorkbook)
	r.Get("/matches/{matchID}/results.csv", h.resultsCSV)
	r.With(h.render.MiddlewareUncoalesced(ResourceByPath)).Get("/admin/matches/{matchID}/results", h.adminResultsPDF)
	r.Post("/matches/{matchID}/entries/{entryID}/stages/{stage}/lock", serve(h.lockStage))
	r.Post("/matches/{matchID}/entries/{entryID}/stages/{stage}/accept", serve(h.acceptStage))
	r.Post("/matches/{matchID}/entries/{entryID}/stages/{stage}/unlock", serve(h.unlockStage))
	r.Post("/matches/{matchID}/pairings", serve(h.createPairing))
	r.Get("/matches/{matchID}/pairings", serve(h.listPairings))
	r.Delete("/matches/{matchID}/pairings/{pairingID}", serve(h.revokePairing))
	r.Get("/matches/{matchID}/stream", h.adminStream)
}

type matchBody struct {
	Name          string           `json:"name"`
	Date          string           `json:"date"`
	Course        string           `json:"course"`
	CompetitionID int64            `json:"competition_id"`
	Competition   *competitionBody `json:"competition"`
}

func matchesJSON(comps []repository.Match) []map[string]any {
	out := make([]map[string]any, 0, len(comps))
	for _, c := range comps {
		out = append(out, matchJSON(c))
	}
	return out
}

type shooterBody struct {
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	Country   string `json:"country"`
	Club      string `json:"club"`
	WAID      string `json:"waid"`
}

func (b shooterBody) shooter(id int64) repository.Shooter {
	return repository.Shooter{ID: id, LastName: b.LastName, FirstName: b.FirstName, Country: b.Country, Club: b.Club, WAID: b.WAID}
}

func shooterJSON(s repository.Shooter) map[string]any {
	return map[string]any{
		"id": s.ID, "last_name": s.LastName, "first_name": s.FirstName,
		"country": s.Country, "club": s.Club, "waid": s.WAID, "archived": s.Archived,
	}
}

func matchJSON(c repository.Match) map[string]any {
	return map[string]any{"id": c.ID, "competition_id": c.CompetitionID, "name": c.Name, "date": c.Date, "course": c.Course, "status": c.Status, "current_stage": c.CurrentStage}
}

func entryJSON(e repository.Entry) map[string]any {
	return map[string]any{
		"id": e.ID, "match_id": e.MatchID, "shooter_id": e.ShooterID,
		"category": e.Category, "lane": e.Lane, "withdrawn": e.Withdrawn,
	}
}

func (h *Handler) listShooters(r *http.Request) (int, any, error) {
	limit, err := queryInt(r, "limit", 0)
	if err != nil {
		return fail(err)
	}
	shooters, err := h.svc.ListShooters(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("include_archived") == "true", limit)
	if err != nil {
		return fail(err)
	}
	counts, err := h.svc.ShooterEntryCounts(r.Context())
	if err != nil {
		return fail(err)
	}
	out := make([]map[string]any, 0, len(shooters))
	for _, s := range shooters {
		j := shooterJSON(s)
		j["entry_count"] = counts[s.ID]
		out = append(out, j)
	}
	return okJSON(map[string]any{"shooters": out})
}

func (h *Handler) getShooter(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("shooterID")
	if req.err != nil {
		return fail(req.err)
	}
	s, usage, err := h.svc.ShooterUsage(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	matches := make([]map[string]any, 0, len(usage.Matches))
	for _, m := range usage.Matches {
		matches = append(matches, map[string]any{"id": m.ID, "name": m.Name, "date": m.Date, "status": m.Status})
	}
	j := shooterJSON(s)
	j["entry_count"] = usage.EntryCount
	j["matches"] = matches
	return okJSON(j)
}

func (h *Handler) updateShooter(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("shooterID")
	var body shooterBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	s := body.shooter(id)
	if err := h.svc.UpdateShooter(r.Context(), s); err != nil {
		return fail(err)
	}
	stored, _, err := h.svc.ShooterUsage(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(shooterJSON(stored))
}

func (h *Handler) archiveShooter(r *http.Request) (int, any, error) {
	return h.setShooterArchived(r, h.svc.ArchiveShooter)
}

func (h *Handler) unarchiveShooter(r *http.Request) (int, any, error) {
	return h.setShooterArchived(r, h.svc.UnarchiveShooter)
}

func (h *Handler) setShooterArchived(r *http.Request, set func(context.Context, int64) (repository.Shooter, error)) (int, any, error) {
	req := in(r)
	id := req.path("shooterID")
	if req.err != nil {
		return fail(req.err)
	}
	s, err := set(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(shooterJSON(s))
}

func (h *Handler) deleteShooter(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("shooterID")
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.DeleteShooter(r.Context(), id); err != nil {
		return fail(err)
	}
	return okJSON(map[string]bool{"deleted": true})
}

func (h *Handler) listMatches(r *http.Request) (int, any, error) {
	comps, err := h.repo.ListMatches(r.Context())
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]any{"matches": matchesJSON(comps)})
}

func (h *Handler) getMatch(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	if req.err != nil {
		return fail(req.err)
	}
	c, err := h.repo.GetMatch(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(matchJSON(c))
}

func (h *Handler) updateMatch(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	var body matchBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.UpdateMatch(r.Context(), repository.Match{ID: id, Name: body.Name, Date: body.Date, Course: body.Course}); err != nil {
		return fail(err)
	}
	c, err := h.repo.GetMatch(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(matchJSON(c))
}

func (h *Handler) deleteMatch(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.DeleteMatch(r.Context(), id); err != nil {
		return fail(err)
	}
	return okJSON(map[string]bool{"deleted": true})
}

func (h *Handler) setMatchStatus(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	var body struct {
		Status string `json:"status"`
	}
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.SetMatchStatus(r.Context(), id, service.MatchStatus(body.Status)); err != nil {
		return fail(err)
	}
	return okJSON(map[string]string{"status": body.Status})
}

func (h *Handler) startNextStage(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	if req.err != nil {
		return fail(req.err)
	}
	stage, err := h.svc.StartNextStage(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]int{"current_stage": stage})
}

func (h *Handler) listEntries(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	if req.err != nil {
		return fail(req.err)
	}
	entries, err := h.svc.ListEntries(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		m := entryJSON(e.Entry)
		stages := make([]map[string]any, 0, len(e.Stages))
		for _, s := range e.Stages {
			stages = append(stages, map[string]any{"stage_number": s.Number, "expected_shots": s.ExpectedShots, "status": s.Status, "scorer_lane": s.ScorerLane, "scorer_name": s.ScorerName, "score": s.Score, "shots_entered": s.ShotsEntered})
		}
		m["stages"] = stages
		out = append(out, m)
	}
	return okJSON(map[string]any{"entries": out})
}

func (h *Handler) updateEntry(r *http.Request) (int, any, error) {
	req := in(r)
	compID, entryID := req.path("matchID"), req.path("entryID")
	var body struct {
		Category string `json:"category"`
		Lane     int    `json:"lane"`
	}
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	e, err := h.svc.UpdateEntry(r.Context(), compID, entryID, body.Category, body.Lane)
	if err != nil {
		return fail(err)
	}
	return okJSON(entryJSON(e))
}

func (h *Handler) deleteEntry(r *http.Request) (int, any, error) {
	req := in(r)
	compID, entryID := req.path("matchID"), req.path("entryID")
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.DeleteEntry(r.Context(), compID, entryID); err != nil {
		return fail(err)
	}
	return okJSON(map[string]bool{"deleted": true})
}

func (h *Handler) adminStream(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "matchID")
	if err != nil {
		writeError(w, err)
		return
	}
	h.serveStream(w, r, roleAdmin, func(ctx context.Context) error { return h.svc.CheckAdminStream(ctx, id) }, func() *service.Subscription { return h.svc.Subscribe(id) }, func(ctx context.Context) (streamEvent, error) {
		if err := h.svc.CheckAdminStream(ctx, id); err != nil {
			return streamEvent{}, err
		}
		return streamEvent{Name: "changed", Data: struct{}{}}, nil
	})
}

func controlRoomJSON(room service.ControlRoom) map[string]any {
	lanes := make([]map[string]any, 0, len(room.Lanes))
	for _, l := range room.Lanes {
		lanes = append(lanes, map[string]any{
			"lane": l.Lane, "entry_id": l.EntryID, "shooter_name": l.ShooterName, "scorer_name": l.ScorerName,
			"status": l.Status, "disputed": l.Disputed, "disputes": l.Disputes,
			"paired": l.Paired, "connected": l.Connected,
		})
	}
	return map[string]any{
		"current_stage": room.CurrentStage, "can_start_next": room.CanStartNext,
		"can_accept_all": room.CanAcceptAll, "lanes": lanes,
	}
}

func (h *Handler) controlRoom(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	if req.err != nil {
		return fail(req.err)
	}
	room, err := h.svc.ControlRoom(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(controlRoomJSON(room))
}

func (h *Handler) acceptAll(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	if req.err != nil {
		return fail(req.err)
	}
	n, err := h.svc.AcceptAll(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]int{"accepted": n})
}

func (h *Handler) stageEvents(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("matchID")
	if req.err != nil {
		return fail(req.err)
	}
	stage, err := queryInt(r, "stage", 0)
	if err != nil || stage < 0 {
		return fail(service.ErrInvalid)
	}
	q := r.URL.Query()
	history, err := h.svc.StageHistory(r.Context(), id, service.StageHistoryFilter{
		Scorer: q.Get("scorer"), Shooter: q.Get("shooter"), Stage: stage,
	})
	if err != nil {
		return fail(err)
	}
	events := make([]map[string]any, 0, len(history.Events))
	for _, e := range history.Events {
		events = append(events, map[string]any{
			"id": e.ID, "time": e.CreatedAt, "stage": e.StageNumber, "entry_id": e.EntryID, "event": e.Event,
			"actor_type": e.ActorType, "actor_name": e.ActorName,
			"shooter_name": e.ShooterName, "scorer_name": e.ScorerName,
		})
	}
	scorers := make([]map[string]any, 0, len(history.Scorers))
	for _, c := range history.Scorers {
		scorers = append(scorers, map[string]any{"scorer_name": c.ScorerName, "locks": c.Locks, "disputes": c.Disputes})
	}
	return okJSON(map[string]any{"events": events, "scorers": scorers})
}
