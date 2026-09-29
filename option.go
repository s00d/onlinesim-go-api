package onlinesim

import (
	"net/http"
	"time"
)

// Option configures a Client.
type Option func(*Client)

// WithLang sets the API language (default "en").
func WithLang(lang string) Option {
	return func(c *Client) {
		if lang != "" {
			c.lang = lang
		}
	}
}

// WithDevID sets the optional developer id.
func WithDevID(devID int64) Option {
	return func(c *Client) {
		c.devID = &devID
	}
}

// WithOAuth sets an OAuth bearer token used as Authorization header.
func WithOAuth(token string) Option {
	return func(c *Client) {
		c.oauth = token
	}
}

// WithBaseURL overrides the OnlineSim API base URL (also used with mock).
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = baseURL
		}
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithRateLimit sets max requests per second (default 1). Pass 0 to disable.
// setOperationOk is additionally paced at most once per 5s (OpenAPI).
func WithRateLimit(rps int) Option {
	return func(c *Client) {
		c.rateLimit = rps
	}
}

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// WithTimeout sets the HTTP client timeout when using the default client.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}
