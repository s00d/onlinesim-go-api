package onlinesim

import (
	"context"
	"time"
)

// WaitCodeOptions configures WaitCode polling.
type WaitCodeOptions struct {
	// Interval between polls. Zero is valid for tests (no sleep).
	Interval time.Duration
	// MaxAttempts before returning ErrTimeout (default 10).
	MaxAttempts int
	// NotEnd calls Next instead of Close when a code arrives.
	NotEnd bool
	// FullMessage requests full SMS text (message_to_code=0).
	FullMessage bool
}

// DefaultWaitCodeOptions returns sensible production defaults.
func DefaultWaitCodeOptions() WaitCodeOptions {
	return WaitCodeOptions{
		Interval:    3 * time.Second,
		MaxAttempts: 10,
	}
}

// WaitCode polls getState until an SMS code arrives or attempts are exhausted.
func (a *NumbersAPI) WaitCode(ctx context.Context, tzid int64, opts WaitCodeOptions) (string, error) {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 10
	}
	// Only apply default interval when using completely zero Interval AND
	// caller didn't intentionally pass Interval:0 with MaxAttempts set via Default.
	// For tests Interval:0 is intentional. Production should use DefaultWaitCodeOptions
	// or set Interval explicitly. If Interval is 0 we sleep nothing (test-friendly).

	msgToCode := 1
	if opts.FullMessage {
		msgToCode = 0
	}
	lastCode := ""

	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		if attempt > 0 && opts.Interval > 0 {
			timer := time.NewTimer(opts.Interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", ctx.Err()
			case <-timer.C:
			}
		} else if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}
		}

		st, err := a.StateOne(ctx, tzid, StateParams{
			MessageToCode: IntPtr(msgToCode),
			MsgList:       false,
			Clean:         true,
		})
		if err != nil {
			return "", err
		}
		code := st.MsgString()
		if code != "" && code != lastCode {
			if opts.NotEnd {
				if err := a.Next(ctx, tzid); err != nil {
					return "", err
				}
			} else {
				if err := a.Close(ctx, tzid); err != nil {
					return "", err
				}
			}
			return code, nil
		}
		lastCode = code
	}
	return "", ErrTimeout
}
