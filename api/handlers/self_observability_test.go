package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/handlers"
)

func TestHealthzIsLiveWithoutDatabase(t *testing.T) {
	server := &handlers.Server{}
	recorder := httptest.NewRecorder()
	server.Healthz(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Fatalf("healthz status = %q, want ok", body.Status)
	}
}

func TestReadyzFailsWhenDatabaseIsUnavailable(t *testing.T) {
	previous := db.Pool
	db.Pool = nil
	t.Cleanup(func() { db.Pool = previous })

	server := &handlers.Server{}
	recorder := httptest.NewRecorder()
	server.Readyz(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	var body struct {
		Status   string `json:"status"`
		Database string `json:"database"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "not_ready" || body.Database != "unavailable" {
		t.Fatalf("readyz body = %+v, want not_ready/unavailable", body)
	}
}
