package catalog

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu            sync.Mutex
	nextAllowedAt time.Time
	interval      time.Duration
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 1
	}
	return &RateLimiter{interval: time.Second / time.Duration(requestsPerSecond)}
}

func (r *RateLimiter) WaitTurn() {
	r.mu.Lock()
	now := time.Now()
	scheduled := now
	if r.nextAllowedAt.After(now) {
		scheduled = r.nextAllowedAt
	}
	r.nextAllowedAt = scheduled.Add(r.interval)
	r.mu.Unlock()

	if sleep := time.Until(scheduled); sleep > 0 {
		time.Sleep(sleep)
	}
}
