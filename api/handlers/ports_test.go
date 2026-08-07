package handlers

import "testing"

func intPtr(v int) *int { return &v }

// Two rules, and they are not the same rule. 80 is a fine port for an image to
// listen on but never a port to publish on, because Traefik owns it. And no host
// port at all is always valid — it means the service is reachable only by other
// services on the uploy network, which is what a database should be.
func TestValidatePorts(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		hostPort *int
		wantOK   bool
	}{
		{"nginx published elsewhere", 80, intPtr(9090), true},
		// Not published is always fine, whatever the image listens on: nothing
		// outside the machine can reach it, so it cannot collide with the proxy.
		{"nginx kept internal", 80, nil, true},
		{"443 kept internal", 443, nil, true},
		{"database kept internal", 6379, nil, true},
		{"database published on purpose", 5432, intPtr(15432), true},
		{"host port may not take the proxy's 80", 3000, intPtr(80), false},
		{"host port may not take the proxy's 443", 3000, intPtr(443), false},
		{"container port out of range", 0, nil, false},
		{"container port above range", 70000, nil, false},
		{"host port out of range", 3000, intPtr(70000), false},
		{"host port zero", 3000, intPtr(0), false},
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
