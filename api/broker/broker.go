package broker

import (
	"sync"
	"time"
)

type EventType int

const (
	Log EventType = iota
	Done
)

type Event struct {
	Type      EventType
	ID        int64
	Order     int
	CreatedAt time.Time
	Output    string
	LogType   string // "stdout" or "stderr"
	Status    string // only for Done
}

var (
	mu   sync.Mutex
	subs = make(map[string]map[chan Event]struct{})
)

func Subscribe(deploymentID string) chan Event {
	ch := make(chan Event, 64)
	mu.Lock()
	defer mu.Unlock()
	if subs[deploymentID] == nil {
		subs[deploymentID] = make(map[chan Event]struct{})
	}
	subs[deploymentID][ch] = struct{}{}
	return ch
}

func Unsubscribe(deploymentID string, ch chan Event) {
	mu.Lock()
	defer mu.Unlock()
	if s, ok := subs[deploymentID]; ok {
		if _, exists := s[ch]; exists {
			delete(s, ch)
			close(ch)
		}
		if len(s) == 0 {
			delete(subs, deploymentID)
		}
	}
}

func publish(deploymentID string, event Event) {
	mu.Lock()
	defer mu.Unlock()
	s := subs[deploymentID]
	for ch := range s {
		select {
		case ch <- event:
		default:
			// subscriber too slow — close to force reconnect via Last-Event-ID
			delete(s, ch)
			close(ch)
		}
	}
	if len(s) == 0 {
		delete(subs, deploymentID)
	}
}

func PublishLog(deploymentID string, id int64, order int, createdAt time.Time, output, logType string) {
	publish(deploymentID, Event{
		Type:      Log,
		ID:        id,
		Order:     order,
		CreatedAt: createdAt,
		Output:    output,
		LogType:   logType,
	})
}

func PublishDone(deploymentID string, status string) {
	publish(deploymentID, Event{
		Type:   Done,
		Status: status,
	})
}
