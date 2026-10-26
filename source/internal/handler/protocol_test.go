package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
	"github.com/sollie/matchmark/internal/wa1500"
)

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		err    error
		status int
		msg    string
	}{
		{service.ErrNotFound, 404, "not found"},
		{repository.ErrNotFound, 404, "not found"},
		{service.ErrGone, 410, "gone"},
		{service.ErrInvalid, 422, "invalid input"},
		{service.ErrInvalidUsername, 422, service.ErrInvalidUsername.Error()},
		{service.ErrWeakPassword, 422, service.ErrWeakPassword.Error()},
		{service.ErrForbidden, 403, "forbidden"},
		{service.ErrBackupNotPermitted, 403, "backup_not_permitted"},
		{service.ErrAdminScoringNeeded, 403, "admin scoring required"},
		{service.ErrLastAdmin, 409, service.ErrLastAdmin.Error()},
		{service.ErrConflict, 409, "conflict"},
		{service.ErrLaneNotManned, 409, "lane is not manned"},
		{service.ErrNotBetweenStages, 409, "entries can only change between stages"},
		{service.ErrStageLocked, 409, "stage is locked"},
		{service.ErrStageNotAwaitingAcceptance, 409, "stage is not awaiting acceptance"},
		{service.ErrNotCurrentStage, 409, "not the current stage"},
		{service.ErrStageGateNotMet, 409, "every manned lane must accept the current stage first"},
		{service.ErrLanesOpen, 409, "a lane is still open"},
		{service.ErrStageNotLocked, 409, "stage is not locked"},
		{service.ErrUnauthenticated, 401, "unauthorized"},
		{wa1500.ErrUpstream, 503, "wa1500 unavailable"},
		{service.ErrRateLimited, 429, "too many searches, try again shortly"},
		{errTooManyStreams, 503, "too many streams"},
		{fmt.Errorf("wrapped: %w", service.ErrConflict), 409, "conflict"},
		{errors.New("boom"), 500, "internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			status, msg := errorResponse(tt.err)
			if status != tt.status || msg != tt.msg {
				t.Fatalf("got %d %q, want %d %q", status, msg, tt.status, tt.msg)
			}
		})
	}
}

func TestInputReportsFirstError(t *testing.T) {
	var got *input
	router := chi.NewRouter()
	router.Post("/{a}/{b}", func(w http.ResponseWriter, r *http.Request) {
		got = in(r)
		got.path("a")
		got.path("b")
		var body struct{}
		got.body(&body)
	})
	tests := []struct {
		name, path, body string
		want             error
	}{
		{"valid", "/1/2", "{}", nil},
		{"bad path before bad body", "/x/2", "nope", service.ErrNotFound},
		{"bad body", "/1/2", "nope", service.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body)))
			if !errors.Is(got.err, tt.want) && got.err != tt.want {
				t.Fatalf("got %v, want %v", got.err, tt.want)
			}
		})
	}
}
