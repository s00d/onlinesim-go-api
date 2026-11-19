package mock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

// APIKey is the only key the mock accepts (as `Authorization: Bearer mock-apikey`).
const APIKey = "mock-apikey"

// Server is a local OnlineSim-compatible HTTP mock. It is strict like the real API: exact paths
// (`.php` where the API has it), the key only as a Bearer header (HTTP 456 ERROR_WRONG_KEY otherwise),
// GET everywhere except POST profile.php.
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

// StatePath enables JSON persistence of balance, operations and webhook_url.
func (b *Builder) StatePath(path string) *Builder {
	b.statePath = path
	return b
}

// StatePathFromEnv reads the state path from ONLINESIM_MOCK_STATE.
func (b *Builder) StatePathFromEnv() *Builder {
	b.statePath = os.Getenv(StatePathEnv)
	return b
}

// Start builds and starts the mock server.
func (b *Builder) Start() (*Server, error) {
	m := &Server{state: newMockState(), statePath: b.statePath}
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

// Close shuts the mock down.
func (m *Server) Close() {
	if m.server != nil {
		m.server.Close()
	}
}

// BaseURL returns the mock API base, e.g. http://127.0.0.1:1234/api/, for onlinesim.WithBaseURL.
func (m *Server) BaseURL() string {
	return m.server.URL + "/api/"
}

// Client returns a client for this mock with the mock's key; opts are applied after the base URL.
func (m *Server) Client(opts ...onlinesim.Option) *onlinesim.Client {
	return onlinesim.New(APIKey, append([]onlinesim.Option{onlinesim.WithBaseURL(m.BaseURL())}, opts...)...)
}

type route struct {
	method  string
	auth    bool
	handler func(m *Server, r *http.Request) (int, any)
}

var routes = map[string]route{
	"/api/getBalance.php":           {http.MethodGet, true, (*Server).handleBalance},
	"/api/getProfile.php":           {http.MethodGet, true, (*Server).handleProfile},
	"/api/profile.php":              {http.MethodPost, true, (*Server).handleSetProfile},
	"/api/webhook-logs":             {http.MethodGet, true, (*Server).handleWebhookLogs},
	"/api/getPaymentHistory.php":    {http.MethodGet, true, (*Server).handlePaymentHistory},
	"/api/getPrice.php":             {http.MethodGet, true, (*Server).handlePrice},
	"/api/getNum.php":               {http.MethodGet, true, (*Server).handleGetNum},
	"/api/getState.php":             {http.MethodGet, true, (*Server).handleGetState},
	"/api/setOperationOk.php":       {http.MethodGet, true, (*Server).handleClose},
	"/api/setOperationRevise.php":   {http.MethodGet, true, (*Server).handleRevise},
	"/api/getNumbersStats.php":      {http.MethodGet, false, (*Server).handleTariffs},
	"/api/rent/getRentNum.php":      {http.MethodGet, true, (*Server).handleRentGet},
	"/api/rent/getRentState.php":    {http.MethodGet, true, (*Server).handleRentState},
	"/api/rent/extendRentState.php": {http.MethodGet, true, (*Server).handleRentExtend},
	"/api/rent/closeRentNum.php":    {http.MethodGet, true, (*Server).handleRentClose},
	"/api/rent/tariffsRent.php":     {http.MethodGet, false, (*Server).handleRentTariffs},
	"/api/getFreeList":              {http.MethodGet, false, (*Server).handleFreeList},
}

func (m *Server) serve(w http.ResponseWriter, r *http.Request) {
	rt, ok := routes[r.URL.Path]
	switch {
	case !ok:
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not Found"})
	case r.Method != rt.method:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"message": "Method Not Allowed"})
	case r.URL.Query().Has("apikey"):
		writeJSON(w, http.StatusBadRequest, map[string]any{"response": "ERROR_PARAMS", "message": "apikey must be sent as a Bearer header"})
	case rt.auth && r.Header.Get("Authorization") == "":
		writeJSON(w, 456, map[string]any{"response": "ERROR_NO_KEY"})
	case rt.auth && r.Header.Get("Authorization") != "Bearer "+APIKey:
		writeJSON(w, 456, map[string]any{"response": "ERROR_WRONG_KEY"})
	default:
		m.mu.Lock()
		status, body := rt.handler(m, r)
		m.saveLocked()
		m.mu.Unlock()
		writeJSON(w, status, body)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
