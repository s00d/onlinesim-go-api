package onlinesim_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

// fakeClock: Sleep only moves Now, nothing really waits.
type fakeClock struct {
	now    time.Time
	sleeps []time.Duration
}

func (c *fakeClock) sleep(_ context.Context, d time.Duration) error {
	c.sleeps = append(c.sleeps, d)
	c.now = c.now.Add(d)
	return nil
}

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) opts(o onlinesim.WaitCodeOptions) onlinesim.WaitCodeOptions {
	o.Sleep, o.Now = c.sleep, c.Now
	return o
}

const (
	waiting = `[{"tzid":7,"response":"TZ_NUM_WAIT"}]`
	closed  = `{"response":1,"tzid":7}`
)

func answer(msg string) reply {
	return ok(`[{"tzid":7,"response":"TZ_NUM_ANSWER","msg":"` + msg + `"}]`)
}

func TestWaitCodeReturnsTheCodeAndCloses(t *testing.T) {
	clk := &fakeClock{}
	c, f := fakeClient(t, []reply{ok(waiting), ok(waiting), answer("195895"), ok(closed)})
	code, err := c.Numbers().WaitCode(ctx, 7, clk.opts(onlinesim.WaitCodeOptions{}))
	must(t, err)
	eq(t, code, "195895")
	eq(t, f.paths(), []string{"/api/getState.php", "/api/getState.php", "/api/getState.php", "/api/setOperationOk.php"})
	eq(t, clk.sleeps, []time.Duration{5 * time.Second, 5 * time.Second})
	eq(t, f.calls[0].query["message_to_code"], "1")
	eq(t, f.calls[0].query["tzid"], "7")
}

func TestWaitCodeKeepOpenSinceFullMessage(t *testing.T) {
	clk := &fakeClock{}
	c, f := fakeClient(t, []reply{answer("old"), answer("Your code is 2991"), ok(`{"response":"1","tzid":7}`)})
	code, err := c.Numbers().WaitCode(ctx, 7, clk.opts(onlinesim.WaitCodeOptions{
		KeepOpen: true, Since: "old", FullMessage: true, Interval: time.Second,
	}))
	must(t, err)
	eq(t, code, "Your code is 2991")
	eq(t, f.paths(), []string{"/api/getState.php", "/api/getState.php", "/api/setOperationRevise.php"})
	eq(t, f.calls[1].query["message_to_code"], "0")
	eq(t, clk.sleeps, []time.Duration{time.Second})
}

func TestWaitCodeFinishedOperationFailsAtOnce(t *testing.T) {
	for _, status := range []string{"TZ_OVER_OK", "ERROR_NO_ITEMS"} {
		clk := &fakeClock{}
		c, f := fakeClient(t, []reply{ok(`[{"tzid":7,"response":"` + status + `"}]`)})
		_, err := c.Numbers().WaitCode(ctx, 7, clk.opts(onlinesim.WaitCodeOptions{}))
		apiErr(t, err, status)
		eq(t, len(f.paths()), 1)
	}
}

func TestWaitCodeFinishedOperationWithNewCodeReturnsIt(t *testing.T) {
	clk := &fakeClock{}
	c, _ := fakeClient(t, []reply{ok(`[{"tzid":7,"response":"TZ_OVER_OK","msg":"1234"}]`), ok(closed)})
	code, err := c.Numbers().WaitCode(ctx, 7, clk.opts(onlinesim.WaitCodeOptions{}))
	must(t, err)
	eq(t, code, "1234")
}

func TestWaitCodeTimeoutCutsTheLastSleep(t *testing.T) {
	clk := &fakeClock{}
	c, f := fakeClient(t, []reply{ok(waiting), ok(waiting), ok(waiting), ok(waiting)})
	_, err := c.Numbers().WaitCode(ctx, 7, clk.opts(onlinesim.WaitCodeOptions{Interval: 4 * time.Second, Timeout: 10 * time.Second}))
	if !errors.Is(err, onlinesim.ErrTimeout) {
		t.Fatalf("want ErrTimeout, got %v", err)
	}
	eq(t, clk.sleeps, []time.Duration{4 * time.Second, 4 * time.Second, 2 * time.Second})
	eq(t, len(f.paths()), 4)
	eq(t, clk.now, time.Time{}.Add(10*time.Second))
}

func TestWaitCodeRetriesTransportErrors(t *testing.T) {
	clk := &fakeClock{}
	c, f := fakeClient(t, []reply{{err: errors.New("reset")}, {status: 502, body: "bad gateway"}, answer("42"), ok(closed)})
	code, err := c.Numbers().WaitCode(ctx, 7, clk.opts(onlinesim.WaitCodeOptions{}))
	must(t, err)
	eq(t, code, "42")
	eq(t, len(f.paths()), 4)
	eq(t, clk.sleeps, []time.Duration{5 * time.Second, 5 * time.Second})
}

func TestWaitCodeReturnsAPIErrorAtOnce(t *testing.T) {
	clk := &fakeClock{}
	c, f := fakeClient(t, []reply{ok(`{"response":"ERROR_NO_OPERATIONS"}`)})
	_, err := c.Numbers().WaitCode(ctx, 7, clk.opts(onlinesim.WaitCodeOptions{}))
	apiErr(t, err, "ERROR_NO_OPERATIONS")
	eq(t, len(f.paths()), 1)
	eq(t, len(clk.sleeps), 0)
}

func TestWaitCodeStopsOnCanceledContext(t *testing.T) {
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	block := roundTrip(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	c := onlinesim.New("k", onlinesim.WithHTTPClient(&http.Client{Transport: block}))
	_, err := c.Numbers().WaitCode(canceled, 7, onlinesim.WaitCodeOptions{}) // real Sleep and clock: never reached
	eq(t, errors.Is(err, context.Canceled), true)
}

func TestWaitCodeReturnsSleepError(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(waiting)})
	stop := errors.New("stop")
	_, err := c.Numbers().WaitCode(ctx, 7, onlinesim.WaitCodeOptions{
		Sleep: func(context.Context, time.Duration) error { return stop },
	})
	eq(t, err, stop)
}
