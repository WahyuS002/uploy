package handlers

import (
	"strings"
	"testing"
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
