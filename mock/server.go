package mock

import (
	"net/http/httptest"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

// Server is a Mock served on a local listener (httptest), for code that needs a real URL: another process,
// a client you cannot inject an http.Client into. Every Mock helper works on it.
type Server struct {
	*Mock
	srv *httptest.Server
}

// Start starts a mock server on 127.0.0.1 with a random port. Close it when done.
func Start(opts ...Option) *Server {
	m := New(opts...)
	return &Server{Mock: m, srv: httptest.NewServer(m)}
}

// Close shuts the server down.
func (s *Server) Close() { s.srv.Close() }

// BaseURL returns the API base, e.g. http://127.0.0.1:1234/api/, for onlinesim.WithBaseURL.
func (s *Server) BaseURL() string { return s.srv.URL + "/api/" }

// Client returns a client that talks to the server over HTTP with the mock's key; opts are applied after
// the base URL.
func (s *Server) Client(opts ...onlinesim.Option) *onlinesim.Client {
	s.mu.Lock()
	key := s.state.apiKey
	s.mu.Unlock()
	return onlinesim.New(key, append([]onlinesim.Option{onlinesim.WithBaseURL(s.BaseURL())}, opts...)...)
}
