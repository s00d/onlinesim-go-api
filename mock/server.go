package mock

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"

	onlinesim "github.com/s00d/onlinesim-go-api/v2"
)

// Server is a local OnlineSim-compatible HTTP mock.
type Server struct {
	mu        sync.Mutex
	server    *httptest.Server
	state     *mockState
	statePath string
}

// Builder configures an optional persisted mock.
type Builder struct {
	statePath string
}

// NewBuilder returns a mock builder.
func NewBuilder() *Builder { return &Builder{} }

// StatePath enables JSON persistence for balance / ops / webhook_url.
func (b *Builder) StatePath(path string) *Builder {
	b.statePath = path
	return b
}

// StatePathFromEnv reads ONLINESIM_MOCK_STATE.
func (b *Builder) StatePathFromEnv() *Builder {
	b.statePath = os.Getenv(StatePathEnv)
	return b
}

// Start builds and starts the mock server.
func (b *Builder) Start() (*Server, error) {
	m := &Server{
		state:     newMockState(),
		statePath: b.statePath,
	}
	if err := m.loadState(); err != nil {
		return nil, err
	}
	m.server = httptest.NewServer(http.HandlerFunc(m.serve))
	return m, nil
}

// Start starts an ephemeral mock (no persistence).
func Start() *Server {
	m, err := NewBuilder().Start()
	if err != nil {
		panic(err)
	}
	return m
}

// Close shuts down the mock server.
func (m *Server) Close() {
	if m.server != nil {
		m.server.Close()
	}
}

// BaseURL returns the mock base URL (no trailing slash), suitable for WithBaseURL.
func (m *Server) BaseURL() string {
	return strings.TrimRight(m.server.URL, "/")
}

// Client returns an onlinesim.Client pointed at this mock.
// Rate limiting is disabled for fast tests.
func (m *Server) Client(opts ...onlinesim.Option) *onlinesim.Client {
	base := []onlinesim.Option{
		onlinesim.WithBaseURL(m.BaseURL()),
		onlinesim.WithRateLimit(0),
	}
	base = append(base, opts...)
	return onlinesim.New("mock-apikey", base...)
}
