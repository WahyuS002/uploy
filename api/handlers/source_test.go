package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/handlers"
	"github.com/WahyuS002/uploy/source"
)

type fakeSourceAnalyzer struct {
	analyze func(context.Context, source.Repo) (source.Analysis, error)
}

func (f fakeSourceAnalyzer) Analyze(ctx context.Context, repo source.Repo) (source.Analysis, error) {
	return f.analyze(ctx, repo)
}

func TestAnalyzeSourceReturnsServerSideFacts(t *testing.T) {
	sha := strings.Repeat("a", 40)
	server := &handlers.Server{SourceAnalyzer: fakeSourceAnalyzer{analyze: func(_ context.Context, repo source.Repo) (source.Analysis, error) {
		if repo != (source.Repo{Owner: "owner", Name: "demo", Branch: "main"}) {
			t.Fatalf("repo = %+v", repo)
		}
		return source.Analysis{
			Repo: repo,
			SHA:  sha,
			Info: source.Info{Provider: "node", RuntimeVersions: map[string]string{"node": "22.11.0"}, StartCommand: "node server.js"},
		}, nil
	}}}

	req := httptest.NewRequest(http.MethodPost, "/api/source/analyze", strings.NewReader(`{"repo_url":"owner/demo"}`))
	rec := httptest.NewRecorder()
	server.AnalyzeSource(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got gen.AnalyzeSourceResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Owner != "owner" || got.Name != "demo" || got.Branch != "main" || got.Sha != sha || got.Provider != "node" || got.RuntimeVersions["node"] != "22.11.0" || got.SuggestedName != "demo" || got.SuggestedPort != 3000 || got.StartCommand == nil || *got.StartCommand != "node server.js" {
		t.Fatalf("response = %+v", got)
	}
}

func TestAnalyzeSourceRouteIsPublic(t *testing.T) {
	server := &handlers.Server{SourceAnalyzer: fakeSourceAnalyzer{analyze: func(_ context.Context, repo source.Repo) (source.Analysis, error) {
		return source.Analysis{Repo: repo, SHA: strings.Repeat("b", 40), Info: source.Info{Provider: "go"}}, nil
	}}}
	mux := newTestMuxForServer(server)
	req := httptest.NewRequest(http.MethodPost, "/api/source/analyze", strings.NewReader(`{"repo_url":"owner/demo"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func newTestMuxForServer(server *handlers.Server) http.Handler {
	// Keep this small route harness independent from the auth/database setup in
	// the rest of the handler tests: source analysis is intentionally public.
	return gen.HandlerWithOptions(server, gen.StdHTTPServerOptions{
		BaseRouter: http.NewServeMux(),
	})
}

func TestAnalyzeSourceMapsErrors(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		err    error
		status int
		want   string
	}{
		{"invalid URL", `{"repo_url":"https://gitlab.com/owner/demo"}`, nil, http.StatusBadRequest, "public github.com"},
		{"missing branch", `{"repo_url":"owner/demo","branch":"feature/missing"}`, source.ErrBranchNotFound, http.StatusBadRequest, "feature/missing"},
		{"GitHub unavailable", `{"repo_url":"owner/demo"}`, source.ErrRemoteUnavailable, http.StatusBadGateway, "could not reach GitHub"},
		{"unsupported", `{"repo_url":"owner/demo"}`, source.ErrUnsupportedSource, http.StatusUnprocessableEntity, "could not determine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &handlers.Server{SourceAnalyzer: fakeSourceAnalyzer{analyze: func(_ context.Context, repo source.Repo) (source.Analysis, error) {
				if errors.Is(tt.err, source.ErrBranchNotFound) && repo.Branch != "feature/missing" {
					t.Fatalf("repo branch = %q", repo.Branch)
				}
				return source.Analysis{}, tt.err
			}}}
			req := httptest.NewRequest(http.MethodPost, "/api/source/analyze", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			server.AnalyzeSource(rec, req)
			if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.want) {
				t.Fatalf("status = %d, body = %s; want %d containing %q", rec.Code, rec.Body.String(), tt.status, tt.want)
			}
		})
	}
}

func TestAnalyzeSourceTimeoutIsExplicit(t *testing.T) {
	server := &handlers.Server{SourceAnalyzer: fakeSourceAnalyzer{analyze: func(ctx context.Context, _ source.Repo) (source.Analysis, error) {
		<-ctx.Done()
		return source.Analysis{}, ctx.Err()
	}}}
	// The production timeout is deliberately long; canceling the request still
	// exercises the same explicit deadline response path without waiting 90s.
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/source/analyze", strings.NewReader(`{"repo_url":"owner/demo"}`)).WithContext(ctx)
	rec := httptest.NewRecorder()
	server.AnalyzeSource(rec, req)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "90 second") {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
