package handlers

import "testing"

func intPtr(v int) *int { return &v }

// The rule that matters: 80 is a fine port for an image to listen on, but never
// a port to publish on, because Traefik owns it. Getting this backwards is what
// made nginx impossible to deploy — the form offered 80, the API refused it, and
// every number the user could type instead was published straight through to a
// container that was not listening there.
func TestValidatePorts(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		hostPort *int
		wantOK   bool
	}{
		{"nginx published elsewhere", 80, intPtr(9090), true},
		{"nginx with no host port collides with the proxy", 80, nil, false},
		{"443 with no host port collides too", 443, nil, false},
		{"database on its own number", 6379, nil, true},
		{"database with an explicit host port", 5432, intPtr(15432), true},
		{"host port may not take the proxy's 80", 3000, intPtr(80), false},
		{"host port may not take the proxy's 443", 3000, intPtr(443), false},
		{"container port out of range", 0, nil, false},
		{"container port above range", 70000, nil, false},
		{"host port out of range", 3000, intPtr(70000), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := validatePorts(tc.port, tc.hostPort)
			if tc.wantOK && msg != "" {
				t.Errorf("port=%d hostPort=%v rejected: %s", tc.port, tc.hostPort, msg)
			}
			if !tc.wantOK && msg == "" {
				t.Errorf("port=%d hostPort=%v was accepted, want rejected", tc.port, tc.hostPort)
			}
		})
	}
}
