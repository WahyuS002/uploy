package auth

import (
	"sync"
	"time"
)

const (
	maxFailedAttempts = 5
	windowDuration    = 15 * time.Minute
	cleanupInterval   = 5 * time.Minute
)

type loginAttempts struct {
	timestamps []time.Time
}

var (
	failedLogins = make(map[string]*loginAttempts)
	mu           sync.Mutex
)

func init() {
	go cleanupLoop()
}

func RecordFailedLogin(email string) {
	mu.Lock()
	defer mu.Unlock()

	entry, ok := failedLogins[email]
	if !ok {
		entry = &loginAttempts{}
		failedLogins[email] = entry
	}
	entry.timestamps = append(entry.timestamps, time.Now())
}

func ClearFailedLogins(email string) {
	mu.Lock()
	defer mu.Unlock()

	delete(failedLogins, email)
}

func IsLoginRateLimited(email string) bool {
	mu.Lock()
	defer mu.Unlock()

	entry, ok := failedLogins[email]
	if !ok {
		return false
	}

	cutoff := time.Now().Add(-windowDuration)
	recent := entry.timestamps[:0]
	for _, t := range entry.timestamps {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	entry.timestamps = recent

	return len(recent) >= maxFailedAttempts
}

func cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		cutoff := time.Now().Add(-windowDuration)
		for email, entry := range failedLogins {
			recent := entry.timestamps[:0]
			for _, t := range entry.timestamps {
				if t.After(cutoff) {
					recent = append(recent, t)
				}
			}
			if len(recent) == 0 {
				delete(failedLogins, email)
			} else {
				entry.timestamps = recent
			}
		}
		mu.Unlock()
	}
}
