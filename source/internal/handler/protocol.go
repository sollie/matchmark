package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
	"github.com/sollie/matchmark/internal/wa1500"
)

type endpoint func(r *http.Request) (status int, body any, err error)

func serve(e endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, body, err := e(r)
		if err != nil {
			writeError(w, err)
			return
		}
		if body == nil {
			w.WriteHeader(status)
			return
		}
		writeJSON(w, status, body)
	}
}

func okJSON(body any) (int, any, error) {
	return http.StatusOK, body, nil
}

func fail(err error) (int, any, error) {
	return 0, nil, err
}

type input struct {
	r   *http.Request
	err error
}

func in(r *http.Request) *input {
	return &input{r: r}
}

func (i *input) path(name string) int64 {
	if i.err != nil {
		return 0
	}
	n, err := pathInt(i.r, name)
	i.err = err
	return n
}

func (i *input) body(v any) {
	if i.err != nil {
		return
	}
	i.err = decode(i.r, v)
}

type errorRule struct {
	target error
	status int
	msg    string
}

var errorRules = []errorRule{
	{service.ErrNotFound, http.StatusNotFound, "not found"},
	{repository.ErrNotFound, http.StatusNotFound, "not found"},
	{service.ErrGone, http.StatusGone, "gone"},
	{service.ErrInvalid, http.StatusUnprocessableEntity, "invalid input"},
	{service.ErrInvalidUsername, http.StatusUnprocessableEntity, ""},
	{service.ErrWeakPassword, http.StatusUnprocessableEntity, ""},
	{service.ErrForbidden, http.StatusForbidden, "forbidden"},
	{service.ErrBackupNotPermitted, http.StatusForbidden, "backup_not_permitted"},
	{service.ErrLastAdmin, http.StatusConflict, ""},
	{service.ErrShooterHasEntries, http.StatusConflict, ""},
	{service.ErrShooterArchived, http.StatusConflict, ""},
	{service.ErrConflict, http.StatusConflict, "conflict"},
	{service.ErrLaneNotManned, http.StatusConflict, "lane is not manned"},
	{service.ErrNotBetweenStages, http.StatusConflict, "entries can only change between stages"},
	{service.ErrStageLocked, http.StatusConflict, "stage is locked"},
	{service.ErrStageNotAwaitingAcceptance, http.StatusConflict, "stage is not awaiting acceptance"},
	{service.ErrNotCurrentStage, http.StatusConflict, "not the current stage"},
	{service.ErrStageGateNotMet, http.StatusConflict, "every manned lane must accept the current stage first"},
	{service.ErrLanesOpen, http.StatusConflict, "a lane is still open"},
	{service.ErrStageNotLocked, http.StatusConflict, "stage is not locked"},
	{service.ErrUnauthenticated, http.StatusUnauthorized, "unauthorized"},
	{wa1500.ErrUpstream, http.StatusServiceUnavailable, "wa1500 unavailable"},
	{service.ErrRateLimited, http.StatusTooManyRequests, "too many searches, try again shortly"},
	{errTooManyStreams, http.StatusServiceUnavailable, "too many streams"},
	{service.ErrAdminScoringNeeded, http.StatusForbidden, "admin scoring required"},
}

func errorResponse(err error) (int, string) {
	for _, rule := range errorRules {
		if errors.Is(err, rule.target) {
			if rule.msg == "" {
				return rule.status, err.Error()
			}
			return rule.status, rule.msg
		}
	}
	return http.StatusInternalServerError, "internal error"
}

func writeError(w http.ResponseWriter, err error) {
	status, msg := errorResponse(err)
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return service.ErrInvalid
	}
	return nil
}

func pathInt(r *http.Request, name string) (int64, error) {
	n, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		return 0, service.ErrNotFound
	}
	return n, nil
}
