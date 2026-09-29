package onlinesim

import (
	"net/http"
	"time"

	"github.com/s00d/onlinesim-go-api/v2/internal/transport"
)

const (
	defaultBaseURL   = "https://onlinesim.host/api"
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.89 Safari/537.36"
	// defaultRateLimit is 1 rps — OnlineSim rejects denser traffic with
	// INTERVAL_CONCURRENT_REQUESTS_ERROR (OpenAPI: reduce request intensity).
	defaultRateLimit = 1
	defaultTimeout   = 30 * time.Second
	// opOkMinInterval matches OpenAPI: setOperationOk on the same tzid at most once / 5s.
	opOkMinInterval = 5 * time.Second
	// DefaultCountry is the dial code used when Country is omitted (0).
	// Matches OnlineSim OpenAPI / JS / PHP defaults (Rust currently uses 1).
	DefaultCountry int = 7
)

// Client is the OnlineSim API client.
type Client struct {
	apiKey     string
	lang       string
	devID      *int64
	oauth      string
	baseURL    string
	userAgent  string
	rateLimit  int
	timeout    time.Duration
	httpClient *http.Client

	tr *transport.Client
}

// New creates a Client with the given API key and options.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		lang:       "en",
		baseURL:    defaultBaseURL,
		userAgent:  defaultUserAgent,
		rateLimit:  defaultRateLimit,
		timeout:    defaultTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}
	lim := transport.NewLimiter(c.rateLimit)
	// Backend: repeated setOperationOk for the same tzid more often than once
	// every 5 seconds → INTERVAL_CONCURRENT_REQUESTS_ERROR.
	lim.SetPathInterval("setOperationOk", opOkMinInterval)
	c.tr = transport.New(transport.Config{
		BaseURL:       c.baseURL,
		APIKey:        c.apiKey,
		Lang:          c.lang,
		DevID:         c.devID,
		OAuth:         c.oauth,
		UserAgent:     c.userAgent,
		HTTPClient:    c.httpClient,
		Limiter:       lim,
		ToError:       ErrorFromAPICode,
		RetryInterval: opOkMinInterval,
		MaxRetries:    2,
	})
	return c
}

// BaseURL returns the configured OnlineSim base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// Numbers returns the temporary SMS numbers API.
func (c *Client) Numbers() *NumbersAPI { return &NumbersAPI{c: c} }

// Free returns the free public numbers API.
func (c *Client) Free() *FreeAPI { return &FreeAPI{c: c} }

// Rent returns the long-term rent API.
func (c *Client) Rent() *RentAPI { return &RentAPI{c: c} }

// User returns the user / balance / webhook settings API.
func (c *Client) User() *UserAPI { return &UserAPI{c: c} }
