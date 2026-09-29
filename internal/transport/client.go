package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TemporaryError is implemented by API errors that are safe to retry after a delay
// (e.g. INTERVAL_CONCURRENT_REQUESTS_ERROR).
type TemporaryError interface {
	Temporary() bool
}

// Config holds shared HTTP transport settings.
type Config struct {
	BaseURL    string
	APIKey     string
	Lang       string
	DevID      *int64
	OAuth      string
	UserAgent  string
	HTTPClient *http.Client
	Limiter    *Limiter
	ToError    ResponseChecker
	// RetryInterval is the pause between retries on Temporary errors (default 5s).
	RetryInterval time.Duration
	// MaxRetries is extra attempts after the first failure (default 2 → 3 tries total).
	MaxRetries int
}

// Client performs authenticated OnlineSim HTTP calls.
type Client struct {
	cfg Config
}

// New creates a transport Client.
func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

// GetJSON performs GET path with query params and JSON-decodes into dest.
// phpSuffix appends ".php" to the path (OnlineSim convention).
func (c *Client) GetJSON(ctx context.Context, path string, params map[string]string, phpSuffix bool, dest any) error {
	return c.roundTrip(ctx, http.MethodGet, c.cfg.BaseURL, path, params, nil, phpSuffix, false, dest)
}

// PostJSON performs POST with form or JSON body.
func (c *Client) PostJSON(ctx context.Context, path string, params map[string]string, phpSuffix bool, dest any) error {
	return c.roundTrip(ctx, http.MethodPost, c.cfg.BaseURL, path, params, nil, phpSuffix, true, dest)
}

// PostRawJSON posts a JSON body to path (no php suffix by default for profile/webhook).
func (c *Client) PostRawJSON(ctx context.Context, baseURL, path string, payload any, phpSuffix bool, dest any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.roundTrip(ctx, http.MethodPost, baseURL, path, nil, raw, phpSuffix, false, dest)
}

func (c *Client) roundTrip(ctx context.Context, method, base, path string, params map[string]string, rawBody []byte, phpSuffix, asForm bool, dest any) error {
	maxRetries := c.cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}
	retryEvery := c.cfg.RetryInterval
	if retryEvery <= 0 {
		retryEvery = 5 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(retryEvery)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		body, err := c.doWithAuth(ctx, method, base, path, params, rawBody, phpSuffix, asForm, c.cfg.OAuth)
		if err != nil {
			return err
		}
		if err := CheckResponseField(body, c.cfg.ToError); err != nil {
			lastErr = err
			if attempt < maxRetries && isTemporary(err) {
				continue
			}
			return err
		}
		if dest == nil {
			return nil
		}
		if err := json.Unmarshal(body, dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	}
	return lastErr
}

func isTemporary(err error) bool {
	var t TemporaryError
	return errors.As(err, &t) && t.Temporary()
}

func (c *Client) doWithAuth(ctx context.Context, method, base, path string, params map[string]string, rawBody []byte, phpSuffix, asForm bool, bearer string) ([]byte, error) {
	if err := c.cfg.Limiter.WaitPath(ctx, path); err != nil {
		return nil, err
	}

	u, err := joinURL(base, path, phpSuffix)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	for k, v := range params {
		if v != "" || k == "apikey" || k == "lang" {
			q.Set(k, v)
		}
	}
	if c.cfg.APIKey != "" && q.Get("apikey") == "" {
		q.Set("apikey", c.cfg.APIKey)
	}
	if c.cfg.Lang != "" && q.Get("lang") == "" {
		q.Set("lang", c.cfg.Lang)
	}
	if c.cfg.DevID != nil && q.Get("dev_id") == "" {
		q.Set("dev_id", fmt.Sprintf("%d", *c.cfg.DevID))
	}

	var reqBody io.Reader
	headers := http.Header{}
	headers.Set("User-Agent", c.cfg.UserAgent)
	headers.Set("Accept", "application/json")

	switch method {
	case http.MethodGet:
		u.RawQuery = q.Encode()
	case http.MethodPost:
		if rawBody != nil {
			reqBody = strings.NewReader(string(rawBody))
			headers.Set("Content-Type", "application/json")
			if len(q) > 0 {
				u.RawQuery = q.Encode()
			}
		} else if asForm {
			reqBody = strings.NewReader(q.Encode())
			headers.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			u.RawQuery = q.Encode()
		}
	}

	if bearer != "" {
		headers.Set("Authorization", "Bearer "+bearer)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}
	req.Header = headers

	hc := c.cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d: %s", res.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

func joinURL(base, path string, phpSuffix bool) (*url.URL, error) {
	base = strings.TrimRight(base, "/")
	path = strings.TrimLeft(path, "/")
	if phpSuffix && path != "" && !strings.HasSuffix(path, ".php") {
		path += ".php"
	}
	return url.Parse(base + "/" + path)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
