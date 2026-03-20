package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/handlers"
	"github.com/WahyuS002/uploy/respond"
)

// newTestMux wires the Server through the generated handler with the same
// middleware and error handler used in production.
func newTestMux() http.Handler {
	s := &handlers.Server{}
	mux := http.NewServeMux()
	gen.HandlerWithOptions(s, gen.StdHTTPServerOptions{
		BaseRouter: mux,
		Middlewares: []gen.MiddlewareFunc{
			testAuthMiddleware,
		},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: err.Error()})
		},
	})
	return mux
}

// testAuthMiddleware mirrors specAuthMiddleware from main.go — checks
// CookieAuthScopes in context to decide whether auth is required.
func testAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(gen.CookieAuthScopes) != nil {
			// No session cookie set → reject as 401
			respond.JSON(w, http.StatusUnauthorized, gen.ErrorResponse{Error: "authentication required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, wantStatus, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var errResp gen.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode ErrorResponse: %v", err)
	}
	if errResp.Error == "" {
		t.Fatal("ErrorResponse.Error is empty")
	}
}

// Test: protected route without session returns JSON ErrorResponse 401
func TestProtectedRouteWithoutSession_Returns401JSON(t *testing.T) {
	mux := newTestMux()

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/auth/logout"},
		{"GET", "/api/auth/me"},
		{"POST", "/api/deployments"},
		{"GET", "/api/deployments/550e8400-e29b-41d4-a716-446655440000/logs"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertJSONError(t, rec, http.StatusUnauthorized)
		})
	}
}

// Test: invalid UUID in logs route returns JSON ErrorResponse 400
func TestInvalidUUIDInLogsRoute_Returns400JSON(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest("GET", "/api/deployments/not-a-uuid/logs", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Generated wrapper validates UUID before auth middleware runs,
	// so this returns 400 with JSON error
	assertJSONError(t, rec, http.StatusBadRequest)
}
