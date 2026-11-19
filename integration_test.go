package onlinesim_test

// Opt-in integration test against a LOCAL backend. Skipped unless both are set:
//
//	ONLINESIM_IT_BASE=https://on.test/api/ ONLINESIM_IT_KEY=<apikey> go test -run Integration ./...
//
// With ONLINESIM_IT_MOCK=1 it also runs the number lifecycle on the provider-free `test` driver
// (`spec/verify/backend_mock.php setup` first, a key of a user without webhook_url, teardown after).

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

// itClient returns a client for the local backend and its base URL, or skips the test.
func itClient(t *testing.T, key string) (*onlinesim.Client, *http.Client, string) {
	t.Helper()
	base, itKey := os.Getenv("ONLINESIM_IT_BASE"), os.Getenv("ONLINESIM_IT_KEY")
	if base == "" || itKey == "" {
		t.Skip("set ONLINESIM_IT_BASE and ONLINESIM_IT_KEY to run")
	}
	u, err := url.Parse(base)
	must(t, err)
	host := u.Hostname()
	local := host == "localhost" || host == "127.0.0.1" || strings.HasSuffix(host, ".test") || strings.HasSuffix(host, ".localhost")
	if !local {
		t.Fatalf("refusing non-local integration host %s", host)
	}
	if key == "" {
		key = itKey
	}
	// Test-only transport: local backends use a self-signed certificate. Not part of the SDK API.
	hc := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec // local test host only
	return onlinesim.New(key, onlinesim.WithBaseURL(base), onlinesim.WithHTTPClient(hc)), hc, base
}

func TestIntegrationReadOnly(t *testing.T) {
	c, _, _ := itClient(t, "")

	t.Run("user.Balance", func(t *testing.T) {
		_, err := c.User().Balance(ctx)
		must(t, err)
	})
	t.Run("user.Profile", func(t *testing.T) {
		p, err := c.User().Profile(ctx)
		must(t, err)
		if p.ID == 0 {
			t.Fatal("profile id 0")
		}
	})
	t.Run("numbers.TariffsOne(44)", func(t *testing.T) {
		uk, err := c.Numbers().TariffsOne(ctx, 44)
		must(t, err)
		eq(t, uk.Code, 44)
		if uk.Services == nil {
			t.Fatal("services nil")
		}
	})
	t.Run("numbers.State", func(t *testing.T) {
		list, err := c.Numbers().State(ctx, onlinesim.StateOptions{})
		must(t, err)
		if list == nil {
			t.Fatal("nil list")
		}
	})
	t.Run("rent.TariffsOne(7)", func(t *testing.T) {
		ru, err := c.Rent().TariffsOne(ctx, 7)
		must(t, err)
		eq(t, ru.Code, 7)
	})
	t.Run("rent.State", func(t *testing.T) {
		list, err := c.Rent().State(ctx)
		must(t, err)
		for _, r := range list {
			if r.Extend == nil || r.Messages == nil {
				t.Fatalf("not normalised: %+v", r)
			}
		}
	})
	t.Run("user.PaymentHistory", func(t *testing.T) {
		h, err := c.User().PaymentHistory(ctx, 1)
		must(t, err)
		if h.Paylist == nil || h.Orders.Data == nil {
			t.Fatalf("%+v", h)
		}
	})
	t.Run("user.WebhookLogs", func(t *testing.T) {
		_, err := c.User().WebhookLogs(ctx, 1)
		must(t, err)
	})
	t.Run("free.List", func(t *testing.T) {
		f, err := c.Free().List(ctx, onlinesim.FreeListOptions{})
		must(t, err)
		if f.Countries == nil || f.Numbers == nil {
			t.Fatalf("%+v", f)
		}
	})
	t.Run("wrong key is APIError ERROR_WRONG_KEY", func(t *testing.T) {
		bad, _, _ := itClient(t, "wrong-key")
		_, err := bad.User().Balance(ctx)
		apiErr(t, err, "ERROR_WRONG_KEY")
	})
}

func TestIntegrationNumberLifecycle(t *testing.T) {
	c, hc, base := itClient(t, "")
	if os.Getenv("ONLINESIM_IT_MOCK") != "1" {
		t.Skip("set ONLINESIM_IT_MOCK=1 (after backend_mock.php setup) to run")
	}
	const country, service = 44, "google"
	n := c.Numbers()

	// injectSMS delivers an SMS through the test driver callback (local backends only, no auth).
	injectSMS := func(tzid int64, text string) {
		t.Helper()
		body := `{"id":` + strconv.FormatInt(tzid, 10) + `,"phoneFrom":"VK","text":"` + text + `"}`
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(base, "/")+"/sms_in_getaways/test", bytes.NewBufferString(body))
		must(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		res, err := hc.Do(req)
		must(t, err)
		_ = res.Body.Close()
		eq(t, res.StatusCode, http.StatusOK)
	}

	price, err := n.Price(ctx, service, country)
	must(t, err)
	if price == nil {
		t.Fatal("no price")
	}

	tzid, err := n.Get(ctx, service, country)
	must(t, err)
	order, err := n.GetWithNumber(ctx, service, country)
	must(t, err)
	if !regexp.MustCompile(`^\+?44\d+$`).MatchString(order.Number) || order.Country != country {
		t.Fatalf("%+v", order)
	}

	list, err := n.State(ctx, onlinesim.StateOptions{})
	must(t, err)
	seen := map[int64]bool{}
	for _, op := range list {
		seen[op.Tzid] = true
	}
	if !seen[tzid] || !seen[order.Tzid] {
		t.Fatalf("state misses %d/%d: %+v", tzid, order.Tzid, list)
	}
	op, err := n.StateOne(ctx, tzid, onlinesim.StateOptions{})
	must(t, err)
	if !strings.HasPrefix(string(op.Response), "TZ_") {
		t.Fatalf("%+v", op)
	}

	// The test driver answers NO_NUMBER to any reject list: proves `reject` is sent.
	_, err = n.Get(ctx, service, country, "4470")
	var noNum *onlinesim.NoNumberError
	if !errors.As(err, &noNum) {
		t.Fatalf("want NoNumberError, got %v", err)
	}

	// No SMS yet: the guard interval keeps it open.
	apiErr(t, n.Close(ctx, order.Tzid), "NO_COMPLETE_TZID")

	injectSMS(tzid, "VK: 195895 is your code")
	wait, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	code, err := n.WaitCode(wait, tzid, onlinesim.WaitCodeOptions{Interval: time.Second, Timeout: 30 * time.Second})
	must(t, err)
	eq(t, code, "195895")

	u := c.User()
	must(t, u.SetWebhookURL(ctx, "https://example.com/hook"))
	p, err := u.Profile(ctx)
	must(t, err)
	eq(t, p.WebhookURL, "https://example.com/hook")
	apiErr(t, u.SetWebhookURL(ctx, "not a url"), "VALIDATION_ERROR")
	must(t, u.ClearWebhookURL(ctx))
	p, err = u.Profile(ctx)
	must(t, err)
	eq(t, p.WebhookURL, "")
}
