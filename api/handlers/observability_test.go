package handlers

import (
	"testing"
	"time"

	"github.com/WahyuS002/uploy/db"
)

func TestImageCommit(t *testing.T) {
	tests := map[string]string{
		"ghcr.io/acme/api@sha256:0123456789abcdef0123456789abcdef": "0123456789ab",
		"ghcr.io/acme/api:a1b4f2c":                                 "a1b4f2c",
		"ghcr.io/acme/api:latest":                                  "",
		"ghcr.io/acme/api":                                         "",
	}
	for image, want := range tests {
		if got := imageCommit(image); got != want {
			t.Errorf("imageCommit(%q) = %q, want %q", image, got, want)
		}
	}
}

func TestDeploymentMarkerDuration(t *testing.T) {
	created := time.Date(2026, time.August, 18, 10, 0, 0, 0, time.UTC)
	started := created.Add(15 * time.Second)
	completed := started.Add(105 * time.Second)
	marker := deploymentMarker(
		db.Service{ID: "svc_1", Name: "api"},
		db.Deployment{
			ID:          "dpl_1",
			Status:      "success",
			CreatedAt:   created,
			StartedAt:   started,
			CompletedAt: &completed,
		},
		"ghcr.io/acme/api:a1b4f2c",
		completed.Add(time.Minute),
	)
	if marker.DurationSeconds != 105 {
		t.Fatalf("duration = %d, want 105", marker.DurationSeconds)
	}
	if marker.Commit != "a1b4f2c" || marker.Status != "success" {
		t.Fatalf("unexpected marker metadata: %+v", marker)
	}

	inProgress := deploymentMarker(
		db.Service{ID: "svc_1", Name: "api"},
		db.Deployment{ID: "dpl_2", Status: "in_progress", CreatedAt: created, StartedAt: started},
		"ghcr.io/acme/api:latest",
		started.Add(30*time.Second),
	)
	if inProgress.DurationSeconds != 30 || inProgress.Status != "in_progress" {
		t.Fatalf("unexpected in-progress marker: %+v", inProgress)
	}
}
