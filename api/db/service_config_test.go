package db

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/WahyuS002/uploy/crypto"
)

const testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func port(v int32) *int32 { return &v }

func baseConfig() ServiceConfig {
	return ServiceConfig{
		SchemaVersion: configSchemaVersion,
		Image:         "nginx:latest",
		ContainerName: "web",
		ContainerPort: 80,
		HostPort:      port(8080),
		ServerID:      "srv-1",
		Domains:       []string{"a.example.com"},
		EnvVars:       []EnvPair{{Key: "PORT", Value: "80"}},
	}
}

// An unchanged config is the state the badge is quiet in, so this is the case
// that has to hold before any of the others mean anything.
func TestDiffConfigsUnchanged(t *testing.T) {
	if got := DiffConfigs(baseConfig(), baseConfig()); len(got) != 0 {
		t.Fatalf("identical configs produced %d changes: %+v", len(got), got)
	}
}

// The bug this whole model replaced: adding a domain and removing it again left
// the service marked as pending forever, because a timestamp cannot tell that a
// change was undone. A comparison can.
func TestDiffConfigsIgnoresUndoneChange(t *testing.T) {
	deployed := baseConfig()

	added := baseConfig()
	added.Domains = []string{"a.example.com", "b.example.com"}
	if len(DiffConfigs(deployed, added)) != 1 {
		t.Fatal("adding a domain should be one change")
	}

	// Same list again, built separately — the undo has to compare equal without
	// relying on it being the very same slice.
	undone := baseConfig()
	undone.Domains = []string{"a.example.com"}
	if got := DiffConfigs(deployed, undone); len(got) != 0 {
		t.Fatalf("adding then removing a domain left %d changes: %+v", len(got), got)
	}
}

// Ordering must not read as a change: the config is compared as a document, and
// the database is free to hand back rows in whatever order it likes.
func TestDiffConfigsIgnoresOrdering(t *testing.T) {
	deployed := serviceConfig(
		Service{Image: "nginx", ContainerName: "web", ContainerPort: 80, ServerID: "srv-1"},
		[]string{"b.example.com", "a.example.com"},
		[]EnvPair{{Key: "B", Value: "2"}, {Key: "A", Value: "1"}},
	)
	current := serviceConfig(
		Service{Image: "nginx", ContainerName: "web", ContainerPort: 80, ServerID: "srv-1"},
		[]string{"a.example.com", "b.example.com"},
		[]EnvPair{{Key: "A", Value: "1"}, {Key: "B", Value: "2"}},
	)
	if got := DiffConfigs(deployed, current); len(got) != 0 {
		t.Fatalf("reordering produced %d changes: %+v", len(got), got)
	}
}

// The count is what the canvas bar shows, so one edit per thing edited — not one
// per service, which is what it used to say.
func TestDiffConfigsCountsEachThing(t *testing.T) {
	deployed := baseConfig()
	current := baseConfig()
	current.Image = "nginx:1.27"
	current.Domains = []string{"a.example.com", "b.example.com", "c.example.com"}
	current.EnvVars = []EnvPair{{Key: "PORT", Value: "3000"}, {Key: "DEBUG", Value: "1"}}

	changes := DiffConfigs(deployed, current)
	// image changed, two domains added, PORT changed, DEBUG added.
	if len(changes) != 5 {
		t.Fatalf("want 5 changes, got %d: %+v", len(changes), changes)
	}

	byKey := map[string]ConfigChange{}
	for _, c := range changes {
		byKey[c.Key] = c
	}
	if c := byKey["image"]; c.Type != "changed" || *c.OldValue != "nginx:latest" || *c.NewValue != "nginx:1.27" {
		t.Errorf("image change wrong: %+v", c)
	}
	if c := byKey["domain:b.example.com"]; c.Type != "added" || c.OldValue != nil {
		t.Errorf("added domain should have no old value: %+v", c)
	}
	if c := byKey["env:PORT"]; c.Type != "changed" || *c.OldValue != "80" || *c.NewValue != "3000" {
		t.Errorf("env change should carry both values: %+v", c)
	}
	if c := byKey["env:DEBUG"]; c.Type != "added" || *c.NewValue != "1" {
		t.Errorf("added env wrong: %+v", c)
	}
}

func TestDiffConfigsRemovals(t *testing.T) {
	deployed := baseConfig()
	current := baseConfig()
	current.Domains = []string{}
	current.EnvVars = []EnvPair{}

	changes := DiffConfigs(deployed, current)
	if len(changes) != 2 {
		t.Fatalf("want 2 removals, got %d: %+v", len(changes), changes)
	}
	for _, c := range changes {
		if c.Type != "removed" || c.NewValue != nil {
			t.Errorf("removal should have no new value: %+v", c)
		}
	}
}

// Unpublishing a service is a real edit with no number on one side of it, so the
// absent case has to say something rather than render as blank.
func TestDiffConfigsHostPort(t *testing.T) {
	deployed := baseConfig()
	current := baseConfig()
	current.HostPort = nil

	changes := DiffConfigs(deployed, current)
	if len(changes) != 1 {
		t.Fatalf("want 1 change, got %d: %+v", len(changes), changes)
	}
	if *changes[0].OldValue != "8080" || *changes[0].NewValue != "not published" {
		t.Errorf("host port change wrong: %+v", changes[0])
	}
}

// A snapshot is written once and read back much later, so a round trip has to
// compare equal to itself — otherwise every service reads as changed forever.
func TestSnapshotRoundTrip(t *testing.T) {
	if err := crypto.Init(testKey); err != nil {
		t.Fatalf("crypto.Init: %v", err)
	}

	cfg := baseConfig()
	encoded, err := encodeSnapshot(cfg)
	if err != nil {
		t.Fatalf("encodeSnapshot: %v", err)
	}

	// Env var values are encrypted in service_env_vars, so a snapshot that
	// repeated them in the clear would move the secret into the deployment
	// history rather than protect it.
	if bytes.Contains([]byte(encoded), []byte("PORT")) {
		t.Error("snapshot stored a variable name in the clear")
	}

	decoded, err := decodeSnapshot(encoded)
	if err != nil {
		t.Fatalf("decodeSnapshot: %v", err)
	}
	if got := DiffConfigs(decoded, cfg); len(got) != 0 {
		t.Fatalf("round trip produced %d changes: %+v", len(got), got)
	}
}

// An empty list and an absent one must serialise the same, or a service that had
// its last domain removed would never compare equal to one that never had any.
func TestSnapshotNormalisesEmptyLists(t *testing.T) {
	fromNil := serviceConfig(Service{Image: "redis"}, nil, nil)
	fromEmpty := serviceConfig(Service{Image: "redis"}, []string{}, []EnvPair{})

	// Compared before encryption: AES-GCM uses a fresh nonce every time, so two
	// ciphertexts of the same config never match and would say nothing here.
	a, err := json.Marshal(fromNil)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b, err := json.Marshal(fromEmpty)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(a) != string(b) {
		t.Fatalf("nil and empty lists serialised differently:\n%s\n%s", a, b)
	}
}
