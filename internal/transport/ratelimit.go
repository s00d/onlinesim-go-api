package transport

import (
	"context"
	"sync"
	"time"
)

// Limiter is a simple request-per-second limiter.
type Limiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

// NewLimiter creates a limiter for rps requests per second. rps <= 0 disables limiting.
func NewLimiter(rps int) *Limiter {
	if rps <= 0 {
		return &Limiter{}
	}
	return &Limiter{interval: time.Second / time.Duration(rps)}
}

// Wait blocks until the next request is allowed or ctx is done.
func (l *Limiter) Wait(ctx context.Context) error {
	if l == nil || l.interval <= 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if l.last.IsZero() {
		l.last = now
		return nil
	}
	waitUntil := l.last.Add(l.interval)
	if !now.Before(waitUntil) {
		l.last = now
		return nil
	}
	delay := waitUntil.Sub(now)
	l.last = waitUntil
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
