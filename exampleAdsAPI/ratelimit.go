package main

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	times  []time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window}
}

func (l *rateLimiter) allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	kept := l.times[:0]
	for _, t := range l.times {
		if now.Sub(t) < l.window {
			kept = append(kept, t)
		}
	}
	l.times = kept
	if len(l.times) >= l.limit {
		return false
	}
	l.times = append(l.times, now)
	return true
}
