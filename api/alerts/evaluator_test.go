package alerts

import (
	"testing"
	"time"
)

func TestTrackerWaitsBeforeStartingAndDeduplicates(t *testing.T) {
	tracker := NewTracker()
	rule := Rule{ID: "rule-1", Condition: ConditionCPUHigh, Threshold: 85, Duration: 10 * time.Minute}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	observation := Observation{TargetID: "svc-1", TargetName: "api", Reachable: true, CPUPercent: 92, ObservedAt: start}

	if got := tracker.Evaluate(rule, observation, false, time.Time{}, start).Kind; got != TransitionNone {
		t.Fatalf("first sample transition = %q, want none", got)
	}
	observation.ObservedAt = start.Add(5 * time.Minute)
	if got := tracker.Evaluate(rule, observation, false, time.Time{}, start.Add(5*time.Minute)).Kind; got != TransitionNone {
		t.Fatalf("early sample transition = %q, want none", got)
	}
	observation.ObservedAt = start.Add(10 * time.Minute)
	transition := tracker.Evaluate(rule, observation, false, time.Time{}, start.Add(10*time.Minute))
	if transition.Kind != TransitionStarted || !transition.Since.Equal(start) {
		t.Fatalf("start transition = %+v, want started at %s", transition, start)
	}
	if got := tracker.Evaluate(rule, observation, true, start, start.Add(20*time.Minute)).Kind; got != TransitionNone {
		t.Fatalf("active sample transition = %q, want none", got)
	}
}

func TestTrackerResolvesWhenConditionClears(t *testing.T) {
	tracker := NewTracker()
	rule := Rule{ID: "rule-1", Condition: ConditionMemoryHigh, Threshold: 90, Duration: time.Minute}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	transition := tracker.Evaluate(rule, Observation{TargetID: "svc-1", TargetName: "api", Reachable: true, MemoryPercent: 50}, true, now.Add(-5*time.Minute), now)
	if transition.Kind != TransitionResolved {
		t.Fatalf("recovery transition = %q, want resolved", transition.Kind)
	}
}

func TestValidateRule(t *testing.T) {
	if err := ValidateRule(Rule{Condition: ConditionServiceDown, Threshold: 1, Duration: 5 * time.Minute}); err == nil {
		t.Fatal("service down with threshold should be rejected")
	}
	if err := ValidateRule(Rule{Condition: ConditionCPUHigh, Threshold: 85, Duration: 5 * time.Minute}); err != nil {
		t.Fatalf("valid CPU rule rejected: %v", err)
	}
}
