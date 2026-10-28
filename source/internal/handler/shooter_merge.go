package handler

import (
	"errors"
	"net/http"

	"github.com/sollie/matchmark/internal/service"
)

func mergePreviewJSON(p service.ShooterMergePreview) map[string]any {
	entries := make([]map[string]any, 0, len(p.Entries))
	for _, e := range p.Entries {
		entries = append(entries, map[string]any{
			"entry_id": e.EntryID, "match_id": e.MatchID, "match_name": e.MatchName,
			"match_date": e.MatchDate, "match_status": e.MatchStatus,
		})
	}
	conflicts := make([]map[string]any, 0, len(p.Conflicts))
	for _, c := range p.Conflicts {
		conflicts = append(conflicts, map[string]any{"match_id": c.MatchID, "match_name": c.MatchName, "match_date": c.MatchDate})
	}
	return map[string]any{
		"source": shooterJSON(p.Source), "target": shooterJSON(p.Target),
		"entries": entries, "conflicts": conflicts,
		"waid_conflict": p.WAIDConflict, "resulting_waid": p.ResultingWAID, "can_merge": p.CanMerge(),
	}
}

func (h *Handler) previewShooterMerge(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("shooterID")
	if req.err != nil {
		return fail(req.err)
	}
	target, err := queryInt(r, "target_id", 0)
	if err != nil || target <= 0 {
		return fail(service.ErrInvalid)
	}
	preview, err := h.svc.PreviewShooterMerge(r.Context(), id, int64(target))
	if err != nil {
		return fail(err)
	}
	return okJSON(mergePreviewJSON(preview))
}

func (h *Handler) mergeShooter(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("shooterID")
	var body struct {
		TargetID int64 `json:"target_id"`
	}
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	preview, err := h.svc.MergeShooters(r.Context(), id, body.TargetID)
	var blocked *service.MergeBlockedError
	if errors.As(err, &blocked) {
		out := mergePreviewJSON(blocked.Preview)
		out["error"] = "merge_blocked"
		return http.StatusConflict, out, nil
	}
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]any{"target_id": body.TargetID, "moved_entries": len(preview.Entries)})
}
