package onlinesim

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the production API base.
const DefaultBaseURL = "https://onlinesim.host/api/"

const (
	defaultTimeout = 30 * time.Second
	bodyExcerpt    = 500
	userAgent      = "onlinesim-go-api/3"
)

// Client is the OnlineSim API client. It is safe for concurrent use.
//
//	client := onlinesim.New(os.Getenv("ONLINESIM_KEY"))
//	b, err := client.User().Balance(ctx)
type Client struct {
	apiKey  string
	baseURL string
	lang    string
	devID   string
	timeout time.Duration
	http    *http.Client

	numbers *NumbersAPI
	rent    *RentAPI
	user    *UserAPI
	free    *FreeAPI
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the full API base (default DefaultBaseURL); the trailing slash is optional.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithLang sets the language of server texts, the `lang` query parameter (default "en").
func WithLang(lang string) Option {
	return func(c *Client) { c.lang = lang }
}

// WithDevID sets the partner developer id, the `dev_id` query parameter (not sent by default).
func WithDevID(devID int64) Option {
	return func(c *Client) { c.devID = strconv.FormatInt(devID, 10) }
}

// WithTimeout sets the per-request timeout (default 30 s). It applies to an injected
// *http.Client too. A timeout returns *TransportError.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithHTTPClient sets the transport: a proxy, a custom TLS setup, a rate-limiting
// RoundTripper, or a fake in tests. Default: a plain &http.Client{}.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

// New creates a client. apiKey is sent as `Authorization: Bearer <apiKey>`, never in the URL;
// it may be empty for the key-free calls (tariffs, free numbers).
func New(apiKey string, opts ...Option) *Client {
	c := &Client{apiKey: apiKey, baseURL: DefaultBaseURL, lang: "en", timeout: defaultTimeout}
	for _, opt := range opts {
		opt(c)
	}
	if !strings.HasSuffix(c.baseURL, "/") {
		c.baseURL += "/"
	}
	if c.http == nil {
		c.http = &http.Client{}
	}
	c.numbers = &NumbersAPI{do: c.do}
	c.rent = &RentAPI{do: c.do}
	c.user = &UserAPI{do: c.do}
	c.free = &FreeAPI{do: c.do}
	return c
}

// Numbers returns the numbers section (order a number, read its SMS, close it).
func (c *Client) Numbers() *NumbersAPI { return c.numbers }

// Rent returns the rent section.
func (c *Client) Rent() *RentAPI { return c.rent }

// User returns the account section (balance, profile, payments, webhook URL).
func (c *Client) User() *UserAPI { return c.user }

// Free returns the free public numbers section.
func (c *Client) Free() *FreeAPI { return c.free }

// doFunc is the one request function the sections share. endpoint is relative to the base URL,
// suffix included ("getNum.php"). A non-nil body switches to POST with a JSON body.
// The decoded response is stored into out (may be nil).
type doFunc func(ctx context.Context, endpoint string, q url.Values, body, out any) error

func (c *Client) do(ctx context.Context, endpoint string, q url.Values, body, out any) error {
	if q == nil {
		q = url.Values{}
	}
	q.Set("lang", c.lang)
	if c.devID != "" {
		q.Set("dev_id", c.devID)
	}
	u := c.baseURL + endpoint + "?" + q.Encode()

	method := http.MethodGet
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("onlinesim: encode body: %w", err)
		}
		method, reader = http.MethodPost, bytes.NewReader(raw)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return fmt.Errorf("onlinesim: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return c.networkError(ctx, err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return c.networkError(ctx, err)
	}
	return decode(res.StatusCode, raw, out)
}

func (c *Client) networkError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &TransportError{Err: context.DeadlineExceeded, msg: fmt.Sprintf("request timed out after %s", c.timeout)}
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return &TransportError{Err: ctxErr, msg: "request canceled: " + ctxErr.Error()}
	}
	return &TransportError{Err: err, msg: "network error: " + err.Error()}
}

// decode applies the response rules: non-JSON → TransportError; an object with `response`
// → success or APIError (checked before the HTTP status: a wrong key is HTTP 456 and the rate limit HTTP 429,
// both with a JSON code);
// anything else → TransportError on a non-2xx status, the body otherwise.
func decode(status int, body []byte, out any) error {
	excerpt := string(body)
	if len(excerpt) > bodyExcerpt {
		excerpt = excerpt[:bodyExcerpt]
	}
	if !json.Valid(body) {
		return &TransportError{Status: status, Body: excerpt, msg: fmt.Sprintf("HTTP %d: response is not JSON", status)}
	}
	if trimmed := bytes.TrimSpace(body); len(trimmed) > 0 && trimmed[0] == '{' {
		var top map[string]json.RawMessage
		if err := json.Unmarshal(body, &top); err == nil {
			if code, ok := top["response"]; ok {
				if err := checkResponse(code, top["message"]); err != nil {
					return err
				}
				return into(status, body, excerpt, out)
			}
		}
	}
	if status < 200 || status >= 300 {
		return &TransportError{Status: status, Body: excerpt, msg: fmt.Sprintf("HTTP %d", status)}
	}
	return into(status, body, excerpt, out)
}

func into(status int, body []byte, excerpt string, out any) error {
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &TransportError{Status: status, Body: excerpt, Err: err, msg: "unexpected response shape: " + err.Error()}
	}
	return nil
}

// checkResponse: `1`/`"1"` is success, an object is a validation error, anything else an error code.
func checkResponse(response, message json.RawMessage) error {
	raw := bytes.TrimSpace(response)
	if string(raw) == "1" || string(raw) == `"1"` {
		return nil
	}
	if len(raw) > 0 && raw[0] == '{' {
		// profile.php: {"response":{"webhook_url":["The webhook url must be a valid URL."]}}
		return newAPIError("VALIDATION_ERROR", validationMessage(raw))
	}
	var code string
	if err := json.Unmarshal(raw, &code); err != nil {
		code = string(raw)
	}
	var msg string
	_ = json.Unmarshal(message, &msg)
	apiErr := newAPIError(code, msg)
	if code == "NO_NUMBER" {
		return &NoNumberError{APIError: *apiErr}
	}
	return apiErr
}

// validationMessage joins the field messages with "; ", in the order the server sent them.
func validationMessage(raw json.RawMessage) string {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil { // {
		return ""
	}
	var parts []string
	for dec.More() {
		if _, err := dec.Token(); err != nil { // field name
			break
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			break
		}
		var list []any
		var one any
		switch {
		case json.Unmarshal(value, &list) == nil:
			for _, v := range list {
				parts = append(parts, fmt.Sprint(v))
			}
		case json.Unmarshal(value, &one) == nil:
			parts = append(parts, fmt.Sprint(one))
		}
	}
	return strings.Join(parts, "; ")
}

func boolParam(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func itoa(n int) string { return strconv.Itoa(n) }

func tzidParam(tzid int64) url.Values {
	return url.Values{"tzid": {strconv.FormatInt(tzid, 10)}}
}
