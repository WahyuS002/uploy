package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/WahyuS002/uploy/db"
)

func TestParseDockerStats(t *testing.T) {
	stats, err := parseDockerStats("abc|app|12.5%|128MiB / 1GiB|1.2MB / 3.4MiB\n")
	if err != nil {
		t.Fatal(err)
	}
	got := stats["app"]
	if got.cpuPercent != 12.5 || got.memoryUsedBytes != 128*(1<<20) || got.memoryLimitBytes != 1<<30 || got.networkInBytes != 1_200_000 || got.networkOutBytes != 3_565_158 {
		t.Fatalf("unexpected stats: %+v", got)
	}
}

func TestParseDockerInspect(t *testing.T) {
	inspections, err := parseDockerInspect("abc|/app|running|2026-08-12T01:02:03.123456789Z\n|missing|missing|\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := inspections["app"]; got.id != "abc" || got.state != "running" || got.startedAt.IsZero() {
		t.Fatalf("unexpected app inspection: %+v", got)
	}
	if got := inspections["missing"]; got.id != "" || got.state != "missing" || !got.startedAt.IsZero() {
		t.Fatalf("unexpected missing inspection: %+v", got)
	}
}

func TestDockerCommandsUseParseableRows(t *testing.T) {
	inspect := dockerInspectCommand("sudo -n docker", []string{"app"})
	stats := dockerStatsCommand("sudo -n docker", []string{"app"})
	if !strings.Contains(inspect, "{{.ID}}|{{.Name}}|{{.State.Status}}|{{.State.StartedAt}}") {
		t.Fatalf("inspect command has incompatible row format: %s", inspect)
	}
	if !strings.Contains(stats, "{{.ID}}|{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}") {
		t.Fatalf("stats command has incompatible row format: %s", stats)
	}
}

func TestParseDockerBytes(t *testing.T) {
	tests := map[string]int64{
		"0B":    0,
		"1.5kB": 1_500,
		"2KiB":  2 << 10,
		"3MB":   3_000_000,
		"4MiB":  4 << 20,
	}
	for input, want := range tests {
		got, err := parseDockerBytes(input)
		if err != nil || got != want {
			t.Fatalf("parseDockerBytes(%q) = %d, %v; want %d, nil", input, got, err, want)
		}
	}
}

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
