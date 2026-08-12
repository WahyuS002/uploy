package handlers

import "testing"

func TestParseDockerStats(t *testing.T) {
	stats, err := parseDockerStats("abc\tapp\t12.5%\t128MiB / 1GiB\t1.2MB / 3.4MiB\n")
	if err != nil {
		t.Fatal(err)
	}
	got := stats["app"]
	if got.cpuPercent != 12.5 || got.memoryUsedBytes != 128*(1<<20) || got.memoryLimitBytes != 1<<30 || got.networkInBytes != 1_200_000 || got.networkOutBytes != 3_565_158 {
		t.Fatalf("unexpected stats: %+v", got)
	}
}

func TestParseDockerInspect(t *testing.T) {
	inspections, err := parseDockerInspect("abc\t/app\trunning\t2026-08-12T01:02:03.123456789Z\n\tmissing\tmissing\t\n")
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

func TestShellQuote(t *testing.T) {
	if got, want := shellQuote("app's service"), `'app'"'"'s service'`; got != want {
		t.Fatalf("shellQuote() = %q; want %q", got, want)
	}
}
