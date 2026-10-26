package handler_test

import (
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/handler"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

var specOnlyOperations = map[string]bool{"GET /api/admin/backup": true}

func specOperations(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	pathLine := regexp.MustCompile(`^  (/api/\S+):$`)
	methodLine := regexp.MustCompile(`^    (get|post|put|delete|patch):$`)
	ops := map[string]bool{}
	path := ""
	inPaths := false
	for _, line := range strings.Split(string(data), "\n") {
		if line == "paths:" {
			inPaths = true
			continue
		}
		if inPaths && line != "" && line[0] != ' ' {
			break
		}
		if m := pathLine.FindStringSubmatch(line); m != nil {
			path = m[1]
		} else if m := methodLine.FindStringSubmatch(line); m != nil && path != "" {
			ops[strings.ToUpper(m[1])+" "+path] = true
		}
	}
	return ops
}

func routerOperations(t *testing.T) map[string]bool {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	router := handler.NewRouter(service.New(repo), repo, handler.WithAuth(service.NewAuth(repo), "http://localhost"))
	ops := map[string]bool{}
	err = chi.Walk(router.(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		ops[method+" "+strings.TrimSuffix(route, "/")] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	return ops
}

func TestOpenAPISpecMatchesRoutes(t *testing.T) {
	spec, routes := specOperations(t), routerOperations(t)
	var problems []string
	for op := range routes {
		if !spec[op] {
			problems = append(problems, "missing from spec: "+op)
		}
	}
	for op := range spec {
		if !routes[op] && !specOnlyOperations[op] {
			problems = append(problems, "not implemented: "+op)
		}
	}
	sort.Strings(problems)
	for _, p := range problems {
		t.Error(p)
	}
}
