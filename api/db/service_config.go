package db

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/WahyuS002/uploy/telemetry"
	"sort"

	"github.com/WahyuS002/uploy/crypto"
)

// configSchemaVersion rides along in every snapshot so a stored one can still be
// read after this struct grows a field. Nothing branches on it yet; it is here
// because the alternative is a migration over a JSONB column later, and one
// integer now is cheaper than that.
const configSchemaVersion = 2

// ServiceConfig is everything about a service that a deploy actually puts on the
// server, and nothing else.
//
// The boundary is exactly what buildDockerRunCmd reads: the image, the container
// name and ports, the server it lands on, the domains that become the Traefik
// rule, and the env vars. Service *name* is deliberately absent — it never
// reaches the container, so renaming a service leaves the running thing
// identical and must not read as a change waiting to be deployed.
//
// One of these is built when a deployment is created, rendered into the
// `docker run`, and stored on the deployment row. Comparing the current one
// against the stored one is what "pending changes" means.
type ServiceConfig struct {
	SchemaVersion int    `json:"schema_version"`
	Image         string `json:"image"`
	ContainerName string `json:"container_name"`
	ContainerPort int32  `json:"container_port"`
	// HostPort is nil for a service that is not published on the host at all.
	HostPort *int32 `json:"host_port"`
	ServerID string `json:"server_id"`
	// Sorted, always non-nil. Both matter: the config is compared as a document,
	// so a different order or a null-versus-empty distinction would surface as a
	// change nobody made.
	Domains []string  `json:"domains"`
	EnvVars []EnvPair `json:"env_vars"`
}

// EnvPair is stored inside snapshots, so it needs stable field names on the
// wire — the Go field names are what an unmarshalled old snapshot has to match.
func (e EnvPair) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{e.Key, e.Value})
}

func (e *EnvPair) UnmarshalJSON(data []byte) error {
	var raw struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.Key, e.Value = raw.Key, raw.Value
	return nil
}

// ServiceConfigs builds the current config for a set of services in two queries,
// however many services there are. The canvas asks for every service in an
// environment at once, and a query per service is what that turns into if the
// per-service loaders are used in a loop.
func ServiceConfigs(ctx context.Context, svcs []Service) (map[string]ServiceConfig, error) {
	configs := make(map[string]ServiceConfig, len(svcs))
	if len(svcs) == 0 {
		return configs, nil
	}

	ids := make([]string, len(svcs))
	for i, s := range svcs {
		ids[i] = s.ID
	}

	domainRows, err := Queries.ListDomainNamesByServiceIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	domains := make(map[string][]string, len(svcs))
	for _, r := range domainRows {
		domains[r.ServiceID] = append(domains[r.ServiceID], r.Domain)
	}

	envRows, err := Queries.GetServiceEnvVarsByServiceIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list env vars: %w", err)
	}
	envs := make(map[string][]EnvPair, len(svcs))
	for _, r := range envRows {
		envs[r.ServiceID] = append(envs[r.ServiceID], EnvPair{
			Key:   r.Key,
			Value: decryptEnvValue(r.ServiceID, r.Key, r.Value),
		})
	}

	for _, s := range svcs {
		configs[s.ID] = serviceConfig(s, domains[s.ID], envs[s.ID])
	}
	return configs, nil
}

// ServiceConfigOf is the single-service form, used where a deployment is about
// to be created and there is exactly one config to build.
func ServiceConfigOf(ctx context.Context, svc Service) (ServiceConfig, error) {
	configs, err := ServiceConfigs(ctx, []Service{svc})
	if err != nil {
		return ServiceConfig{}, err
	}
	return configs[svc.ID], nil
}

func serviceConfig(svc Service, domains []string, envs []EnvPair) ServiceConfig {
	// Never nil: an absent list and an empty one must serialise identically, or a
	// service that had its last domain removed would not compare equal to one
	// that never had any.
	if domains == nil {
		domains = []string{}
	}
	if envs == nil {
		envs = []EnvPair{}
	}
	// The queries already order these, but sorting here means the guarantee
	// belongs to the config rather than to whichever query happened to build it.
	sort.Strings(domains)
	sort.Slice(envs, func(i, j int) bool { return envs[i].Key < envs[j].Key })

	return ServiceConfig{
		SchemaVersion: configSchemaVersion,
		Image:         svc.Image,
		ContainerName: svc.ContainerName,
		ContainerPort: svc.ContainerPort,
		HostPort:      svc.HostPort,
		ServerID:      svc.ServerID,
		Domains:       domains,
		EnvVars:       envs,
	}
}

// DeployedConfigs returns, per service, the config its last successful
// deployment actually shipped — or no entry at all when there is none to read.
//
// A missing entry covers two cases that behave the same way: a service that has
// never deployed, and one whose deployments all predate snapshots being stored.
// Neither can be compared against, so neither can be called up to date.
func DeployedConfigs(ctx context.Context, serviceIDs []string) (map[string]ServiceConfig, error) {
	deployed := make(map[string]ServiceConfig, len(serviceIDs))
	if len(serviceIDs) == 0 {
		return deployed, nil
	}

	rows, err := Queries.ListLatestSuccessfulConfigs(ctx, serviceIDs)
	if err != nil {
		return nil, fmt.Errorf("list deployed configs: %w", err)
	}
	for _, r := range rows {
		cfg, err := decodeSnapshot(r.ConfigurationSnapshot.String)
		if err != nil {
			// A snapshot that cannot be read is the same as not having one: the
			// service reads as pending, which is the safe direction. Skipping it
			// beats failing the whole request over one unreadable row.
			telemetry.Printf("deployed config for service %s is unreadable: %v", r.ServiceID, err)
			continue
		}
		deployed[r.ServiceID] = cfg
	}
	return deployed, nil
}

// ConfigChange is one difference between what is running and what is configured,
// at the granularity a person edits in: a single field, a single domain, a single
// environment variable. A service whose image changed and which gained two
// domains is three changes, not one.
type ConfigChange struct {
	// Key identifies the change for the UI to key a list on. Stable across
	// requests, unique within a diff.
	Key   string `json:"key"`
	Label string `json:"label"`
	// Type is "added", "removed" or "changed".
	Type string `json:"type"`
	// OldValue and NewValue are nil on the side where the thing does not exist,
	// which is what "added" and "removed" mean.
	OldValue *string `json:"old_value"`
	NewValue *string `json:"new_value"`
}

// DiffConfigs reports how current differs from deployed, in a stable order:
// scalar fields first in a fixed sequence, then domains, then variables, each
// alphabetically. Stable because the list is rendered every time the review
// dialog opens, and rows that reshuffle between two identical states read as
// something having happened.
func DiffConfigs(deployed, current ServiceConfig) []ConfigChange {
	changes := []ConfigChange{}

	scalar := func(key, label, old, new string) {
		if old == new {
			return
		}
		changes = append(changes, ConfigChange{
			Key: key, Label: label, Type: "changed",
			OldValue: &old, NewValue: &new,
		})
	}

	scalar("image", "Image", deployed.Image, current.Image)
	scalar("container_name", "Container name", deployed.ContainerName, current.ContainerName)
	scalar("container_port", "Container port", fmt.Sprint(deployed.ContainerPort), fmt.Sprint(current.ContainerPort))
	scalar("host_port", "Published port", hostPortLabel(deployed.HostPort), hostPortLabel(current.HostPort))
	scalar("server_id", "Server", deployed.ServerID, current.ServerID)

	changes = append(changes, diffSet(
		"domain", "Domain", deployed.Domains, current.Domains,
	)...)
	changes = append(changes, diffMap(
		"env", "Variable", envMap(deployed.EnvVars), envMap(current.EnvVars),
	)...)

	return changes
}

// hostPortLabel spells out the absent case rather than leaving it blank: "" next
// to a port number reads as missing data, and this is a real state a service can
// be in on purpose.
func hostPortLabel(port *int32) string {
	if port == nil {
		return "not published"
	}
	return fmt.Sprint(*port)
}

func envMap(pairs []EnvPair) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.Key] = p.Value
	}
	return m
}

// diffSet reports membership changes for a list whose entries carry no value of
// their own — a domain is either attached or it is not, so it can only be added
// or removed, never changed.
func diffSet(keyPrefix, labelPrefix string, deployed, current []string) []ConfigChange {
	in := func(list []string, want string) bool {
		for _, v := range list {
			if v == want {
				return true
			}
		}
		return false
	}

	changes := []ConfigChange{}
	for _, name := range sortedUnion(deployed, current) {
		was, is := in(deployed, name), in(current, name)
		switch {
		case is && !was:
			changes = append(changes, ConfigChange{
				Key: keyPrefix + ":" + name, Label: labelPrefix + " " + name,
				Type: "added", NewValue: strPtr(name),
			})
		case was && !is:
			changes = append(changes, ConfigChange{
				Key: keyPrefix + ":" + name, Label: labelPrefix + " " + name,
				Type: "removed", OldValue: strPtr(name),
			})
		}
	}
	return changes
}

// diffMap reports per-key changes for something keyed and valued — an
// environment variable can appear, disappear, or keep its name and change what
// it holds, and the dialog says which.
func diffMap(keyPrefix, labelPrefix string, deployed, current map[string]string) []ConfigChange {
	keys := make([]string, 0, len(deployed)+len(current))
	for k := range deployed {
		keys = append(keys, k)
	}
	for k := range current {
		if _, dup := deployed[k]; !dup {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	changes := []ConfigChange{}
	for _, k := range keys {
		oldVal, was := deployed[k]
		newVal, is := current[k]
		change := ConfigChange{Key: keyPrefix + ":" + k, Label: labelPrefix + " " + k}
		switch {
		case is && !was:
			change.Type, change.NewValue = "added", &newVal
		case was && !is:
			change.Type, change.OldValue = "removed", &oldVal
		case oldVal != newVal:
			change.Type, change.OldValue, change.NewValue = "changed", &oldVal, &newVal
		default:
			continue
		}
		changes = append(changes, change)
	}
	return changes
}

func sortedUnion(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, v := range list {
			if _, dup := seen[v]; dup {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

func strPtr(s string) *string { return &s }

// PendingChangeCounts reports, per service, how many changes are waiting to be
// deployed. Zero means the running container matches what the service says it
// is.
//
// A service with no deployed config to compare against counts as one change —
// it is pending (it has either never run, or ran before Uploy recorded what it
// shipped), but there is no honest itemised answer to give, so the count says
// "something" rather than inventing a list.
//
// Three queries for a list of any length, none of them per service.
func PendingChangeCounts(ctx context.Context, svcs []Service) (map[string]int, error) {
	counts := make(map[string]int, len(svcs))
	if len(svcs) == 0 {
		return counts, nil
	}

	current, err := ServiceConfigs(ctx, svcs)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(svcs))
	for i, s := range svcs {
		ids[i] = s.ID
	}
	deployed, err := DeployedConfigs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, s := range svcs {
		base, ok := deployed[s.ID]
		if !ok {
			counts[s.ID] = 1
			continue
		}
		counts[s.ID] = len(DiffConfigs(base, current[s.ID]))
	}
	return counts, nil
}

// PendingChanges is the itemised form for one service: the changes themselves,
// and whether there was anything to compare against at all.
//
// hasBaseline false means the service is pending but Uploy cannot say how — the
// caller should say that rather than render an empty list, which would read as
// "nothing to deploy" directly under a badge saying otherwise.
func PendingChanges(ctx context.Context, svc Service) (changes []ConfigChange, hasBaseline bool, err error) {
	current, err := ServiceConfigOf(ctx, svc)
	if err != nil {
		return nil, false, err
	}
	deployed, err := DeployedConfigs(ctx, []string{svc.ID})
	if err != nil {
		return nil, false, err
	}
	base, ok := deployed[svc.ID]
	if !ok {
		return []ConfigChange{}, false, nil
	}
	return DiffConfigs(base, current), true, nil
}

// encodeSnapshot renders a config for storage on the deployment row.
//
// Encrypted, because a config holds environment variable values and those are
// encrypted in service_env_vars — writing them in the clear here would move the
// secret rather than protect it, into a table whose whole purpose is to be kept
// around. The comparison never looks inside the stored form, so nothing is lost
// by it being opaque.
func encodeSnapshot(cfg ServiceConfig) (string, error) {
	plain, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return crypto.Encrypt(string(plain))
}

// decodeSnapshot reverses encodeSnapshot. An unreadable snapshot is reported as
// an error rather than guessed at: the caller treats that as having no baseline,
// which marks the service pending — the safe direction, since the alternative is
// claiming a server matches a config nobody could read.
func decodeSnapshot(stored string) (ServiceConfig, error) {
	plain, err := crypto.Decrypt(stored)
	if err != nil {
		return ServiceConfig{}, err
	}
	var cfg ServiceConfig
	if err := json.Unmarshal([]byte(plain), &cfg); err != nil {
		return ServiceConfig{}, err
	}
	return cfg, nil
}
