package handlers

import "testing"

func strPtr(v string) *string { return &v }

// The days are the part worth checking: Docker takes a Go duration and Go
// durations have no day unit, so "7d" has to reach it spelled in hours or the
// stream comes back empty with nothing to explain why. The unknown case is the
// whole validation — the generated binder hands over any string it is given.
func TestLogSinceFlag(t *testing.T) {
	tests := []struct {
		name   string
		since  *string
		want   string
		wantOK bool
	}{
		{"no range asked for", nil, "", true},
		{"hours pass through", strPtr("6h"), " --since 6h", true},
		{"a week is 168 hours", strPtr("7d"), " --since 168h", true},
		{"a month is 720 hours", strPtr("30d"), " --since 720h", true},
		{"a range nobody offers", strPtr("all"), "", false},
		// Docker would accept this happily; we do not, because then the value
		// reaching the shell would no longer be one of five known strings.
		{"a duration we never listed", strPtr("90m"), "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := logSinceFlag(tt.since)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("flag = %q, want %q", got, tt.want)
			}
		})
	}
}
