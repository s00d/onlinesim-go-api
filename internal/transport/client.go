package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

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
	body, err := c.do(ctx, http.MethodGet, c.cfg.BaseURL, path, params, nil, phpSuffix, false)
	if err != nil {
		return err
	}
	if err := CheckResponseField(body, c.cfg.ToError); err != nil {
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

// PostJSON performs POST with form or JSON body.
func (c *Client) PostJSON(ctx context.Context, path string, params map[string]string, phpSuffix bool, dest any) error {
	body, err := c.do(ctx, http.MethodPost, c.cfg.BaseURL, path, params, nil, phpSuffix, true)
	if err != nil {
		return err
	}
	if err := CheckResponseField(body, c.cfg.ToError); err != nil {
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

// PostRawJSON posts a JSON body to path (no php suffix by default for profile/webhook).
func (c *Client) PostRawJSON(ctx context.Context, baseURL, path string, payload any, phpSuffix bool, dest any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body, err := c.do(ctx, http.MethodPost, baseURL, path, nil, raw, phpSuffix, false)
	if err != nil {
		return err
	}
	if err := CheckResponseField(body, c.cfg.ToError); err != nil {
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

func (c *Client) do(ctx context.Context, method, base, path string, params map[string]string, rawBody []byte, phpSuffix, asForm bool) ([]byte, error) {
	return c.doWithAuth(ctx, method, base, path, params, rawBody, phpSuffix, asForm, c.cfg.OAuth)
}

func (c *Client) doWithAuth(ctx context.Context, method, base, path string, params map[string]string, rawBody []byte, phpSuffix, asForm bool, bearer string) ([]byte, error) {
	if err := c.cfg.Limiter.Wait(ctx); err != nil {
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
