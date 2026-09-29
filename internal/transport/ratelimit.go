package transport

import (
	"context"
	"sync"
	"time"
)

// Limiter paces outbound API calls.
//
// OnlineSim returns INTERVAL_CONCURRENT_REQUESTS_ERROR when requests are too
// frequent. OpenAPI documents a hard rule for setOperationOk: at most one
// close/ban per tzid every 5 seconds. A global 1 rps default keeps getNum and
// friends under the general frequency limit.
type Limiter struct {
	mu sync.Mutex

	interval time.Duration // global min gap; 0 = disabled
	last     time.Time

	pathInterval map[string]time.Duration
	pathLast     map[string]time.Time
}

// NewLimiter creates a limiter for rps requests per second. rps <= 0 disables limiting.
func NewLimiter(rps int) *Limiter {
	if rps <= 0 {
		return &Limiter{}
	}
	return &Limiter{
		interval:     time.Second / time.Duration(rps),
		pathInterval: map[string]time.Duration{},
		pathLast:     map[string]time.Time{},
	}
}

// SetPathInterval sets a per-path minimum gap (e.g. setOperationOk → 5s).
// No-op when the limiter is disabled (rps <= 0).
func (l *Limiter) SetPathInterval(path string, d time.Duration) {
	if l == nil || l.interval <= 0 || path == "" || d <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.pathInterval == nil {
		l.pathInterval = map[string]time.Duration{}
	}
	if l.pathLast == nil {
		l.pathLast = map[string]time.Time{}
	}
	l.pathInterval[path] = d
}

// Wait blocks until the next global slot is free.
func (l *Limiter) Wait(ctx context.Context) error {
	return l.WaitPath(ctx, "")
}

// WaitPath blocks until both the global and path-specific gaps have elapsed.
func (l *Limiter) WaitPath(ctx context.Context, path string) error {
	if l == nil || l.interval <= 0 {
		return nil
	}

	l.mu.Lock()
	now := time.Now()
	waitUntil := now

	if !l.last.IsZero() {
		if t := l.last.Add(l.interval); t.After(waitUntil) {
			waitUntil = t
		}
	}
	if path != "" {
		if d, ok := l.pathInterval[path]; ok && d > 0 {
			if prev, ok := l.pathLast[path]; ok && !prev.IsZero() {
				if t := prev.Add(d); t.After(waitUntil) {
					waitUntil = t
				}
			}
		}
	}

	delay := waitUntil.Sub(now)
	if delay <= 0 {
		l.last = now
		if path != "" {
			l.pathLast[path] = now
		}
		l.mu.Unlock()
		return nil
	}

	// Reserve the slot before sleeping so concurrent Waiters queue correctly.
	l.last = waitUntil
	if path != "" {
		l.pathLast[path] = waitUntil
	}
	l.mu.Unlock()

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
