package onlinesim

import (
	"context"
	"errors"
	"time"
)

// WaitCodeOptions configures WaitCode. Each zero field means its default.
type WaitCodeOptions struct {
	// Interval between polls; default 5 s.
	Interval time.Duration
	// Timeout: give up after this long with ErrTimeout; default 600 s.
	Timeout time.Duration
	// KeepOpen calls Next instead of Close after the code, to wait for another SMS.
	KeepOpen bool
	// FullMessage returns the full SMS text instead of the code.
	FullMessage bool
	// Since is ignored as a code (the previous code when waiting for the next one).
	Since string
	// Sleep waits d or until ctx is done (tests inject a fake); default a real timer.
	Sleep func(ctx context.Context, d time.Duration) error
	// Now is the clock (tests inject a fake); default time.Now.
	Now func() time.Time
}

const (
	defaultWaitInterval = 5 * time.Second
	defaultWaitTimeout  = 600 * time.Second
)

// WaitCode polls StateOne(tzid) every Interval until a code different from Since arrives, then
// calls Close (or Next with KeepOpen) and returns the code.
//
// A finished operation (TZ_OVER_OK, ERROR_NO_ITEMS) without a new code returns *APIError with that
// state as the code. Running out of Timeout returns an error wrapping ErrTimeout. A *TransportError
// is retried until the timeout; any *APIError is returned at once, and so is ctx's error.
func (a *NumbersAPI) WaitCode(ctx context.Context, tzid int64, opts WaitCodeOptions) (string, error) {
	interval, timeout := opts.Interval, opts.Timeout
	if interval <= 0 {
		interval = defaultWaitInterval
	}
	if timeout <= 0 {
		timeout = defaultWaitTimeout
	}
	sleep, now := opts.Sleep, opts.Now
	if sleep == nil {
		sleep = realSleep
	}
	if now == nil {
		now = time.Now
	}

	deadline := now().Add(timeout)
	for {
		op, err := a.StateOne(ctx, tzid, StateOptions{FullMessage: opts.FullMessage})
		var transport *TransportError
		switch {
		case err == nil:
			if code := op.Msg.Text; code != "" && code != opts.Since {
				finish := a.Close
				if opts.KeepOpen {
					finish = a.Next
				}
				if err := finish(ctx, tzid); err != nil {
					return "", err
				}
				return code, nil
			}
			if op.Response == TZOverOK || op.Response == TZErrorNoItems {
				return "", newAPIError(string(op.Response), "")
			}
		case !errors.As(err, &transport) || ctx.Err() != nil:
			return "", err
		}

		left := deadline.Sub(now())
		if left <= 0 {
			return "", waitTimeout(tzid, timeout)
		}
		if err := sleep(ctx, min(interval, left)); err != nil {
			return "", err
		}
	}
}

func realSleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
