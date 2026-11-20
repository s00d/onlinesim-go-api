package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

// DefaultAPIKey is the key a mock accepts unless WithAPIKey sets another.
const DefaultAPIKey = "mock-key"

// APIKey is the default key.
//
// Deprecated: use DefaultAPIKey (or WithAPIKey). Since 3.1.0 its value is "mock-key".
const APIKey = DefaultAPIKey

// BaseURL is the base URL of clients made by Mock.Client. Never contacted: the mock answers in-process.
const BaseURL = "https://mock.onlinesim.invalid/api/"

// Mock is an in-process fake OnlineSim API for tests: no network, no key, no money, no clock.
// It is an http.RoundTripper that answers like the real backend (and an http.Handler, which Start serves);
// Client returns a normal *onlinesim.Client wired to it. Test helpers add SMS, queue failures and expose the
// state. A Mock is safe for concurrent use.
//
//	m := mock.New(mock.WithBalance(10))
//	client := m.Client()
//	tzid, _ := client.Numbers().Get(ctx, "google", 44)
//	_ = m.AddSMSAfter(tzid, "Your code: 123-456", 2)
//	code, _ := client.Numbers().WaitCode(ctx, tzid, onlinesim.WaitCodeOptions{Sleep: noSleep}) // "123456"
type Mock struct {
	mu    sync.Mutex
	state *state
}

type options struct {
	apiKey        string
	balance       float64
	prices        Prices
	rentTariffs   map[int]RentTariff
	guardInterval int
}

// Option configures New and Start.
type Option func(*options)

// WithAPIKey sets the key the mock accepts as `Authorization: Bearer <key>` (default DefaultAPIKey).
func WithAPIKey(key string) Option { return func(o *options) { o.apiKey = key } }

// WithBalance sets the starting balance, USD (default 100).
func WithBalance(usd float64) Option { return func(o *options) { o.balance = usd } }

// WithPrices replaces the number prices (default {7: {vkcom: 0.5, telegram: 1.2, google: 0.3},
// 44: {google: 0.2, vkcom: 0.36}}). They also feed Numbers().Tariffs.
func WithPrices(p Prices) Option { return func(o *options) { o.prices = p } }

// WithRentTariffs replaces the rent tariffs (default {7: {Days: {1: 1, 3: 2.5, 7: 5}, Extend: 1}}).
func WithRentTariffs(t map[int]RentTariff) Option { return func(o *options) { o.rentTariffs = t } }

// WithGuardInterval sets the guard interval of new operations and rents, seconds (default 0). Above 0 an
// operation without SMS, or an unconfirmed rent without any SMS, cannot be closed (NO_COMPLETE_TZID).
func WithGuardInterval(seconds int) Option { return func(o *options) { o.guardInterval = seconds } }

// New returns a mock in its initial state.
func New(opts ...Option) *Mock {
	o := options{apiKey: DefaultAPIKey, balance: 100, prices: defaultPrices(), rentTariffs: defaultRentTariffs()}
	for _, opt := range opts {
		opt(&o)
	}
	return &Mock{state: newState(o)}
}

// Client returns a normal client wired to the mock with the mock's key; opts are applied after the mock's
// (onlinesim.WithLang etc.). To test a wrong key, build the client yourself: onlinesim.New("wrong",
// onlinesim.WithHTTPClient(&http.Client{Transport: m})). Any base URL works.
func (m *Mock) Client(opts ...onlinesim.Option) *onlinesim.Client {
	m.mu.Lock()
	key := m.state.apiKey
	m.mu.Unlock()
	base := []onlinesim.Option{onlinesim.WithBaseURL(BaseURL), onlinesim.WithHTTPClient(&http.Client{Transport: m})}
	return onlinesim.New(key, append(base, opts...)...)
}

// RoundTrip answers req in-process (http.RoundTripper).
func (m *Mock) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
	}
	status, contentType, payload := m.handle(req.Method, req.URL, req.Header, body)
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode:    status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": {contentType}},
		Body:          io.NopCloser(bytes.NewReader(payload)),
		ContentLength: int64(len(payload)),
		Request:       req,
	}, nil
}

// ServeHTTP answers r (http.Handler), for serving the mock on a real listener (see Start).
func (m *Mock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	status, contentType, payload := m.handle(r.Method, r.URL, r.Header, body)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

// handle: log the request → failure queue → unknown path 404 → wrong key 456 / missing key 200 → handler.
func (m *Mock) handle(method string, u *url.URL, header http.Header, body []byte) (int, string, []byte) {
	path := strings.TrimPrefix(u.Path, "/")
	if at := strings.Index(u.Path, "/api/"); at >= 0 {
		path = u.Path[at+len("/api/"):]
	}
	query := map[string]string{}
	for k, v := range u.Query() {
		query[k] = v[0]
	}
	req := Request{Method: strings.ToUpper(method), Path: path, Query: query}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &req.Body); err != nil {
			req.Body = string(body)
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.state
	s.requests = append(s.requests, req)

	if f, ok := s.takeFailure(path); ok {
		if f.code == "" {
			return text(f.status, "Mock transport failure")
		}
		return jsonBody(f.status, obj{"response": f.code})
	}
	h, ok := routes[path]
	if !ok {
		return text(http.StatusNotFound, "Not Found")
	}
	// A wrong key is HTTP 456 (public endpoints check a sent key too); no key at all is HTTP 200.
	auth := header.Get("Authorization")
	if auth != "" && auth != "Bearer "+s.apiKey {
		return jsonBody(456, obj{"response": "ERROR_WRONG_KEY"})
	}
	if auth == "" && !publicPaths[path] {
		return jsonBody(http.StatusOK, obj{"response": "ERROR_WRONG_KEY"})
	}
	return jsonBody(http.StatusOK, h(s, &req))
}

func jsonBody(status int, v any) (int, string, []byte) {
	raw, err := json.Marshal(v)
	if err != nil { // only plain maps, slices and numbers go in: a bug in the mock
		panic("mock: encode response: " + err.Error())
	}
	return status, "application/json", raw
}

func text(status int, msg string) (int, string, []byte) {
	return status, "text/html", []byte("<html><body>" + strconv.Itoa(status) + " " + msg + "</body></html>")
}

// SMSOption configures AddSMS and AddSMSAfter.
type SMSOption func(*string)

// FromService sets the SMS service (sender); default the operation's service, "other" for a rent.
func FromService(service string) SMSOption { return func(s *string) { *s = service } }

func smsService(opts []SMSOption) string {
	var service string
	for _, opt := range opts {
		opt(&service)
	}
	return service
}

// AddSMS makes an SMS arrive now at number operation or rent tzid. It fails when there is no open
// operation or rent with that id.
func (m *Mock) AddSMS(tzid int64, text string, opts ...SMSOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.state.deliver(tzid, text, smsService(opts)) {
		return fmt.Errorf("mock: no open operation or rent %d", tzid)
	}
	return nil
}

// AddSMSAfter makes the SMS arrive after polls state requests for tzid (getState / rent getRentState,
// listing it alone or with others): with polls = 2 the third poll sees it. Lets WaitCode run with a
// no-op Sleep. It fails when there is no operation or rent with that id.
func (m *Mock) AddSMSAfter(tzid int64, text string, polls int, opts ...SMSOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.state.operations[tzid]; !ok {
		if _, ok := m.state.rents[tzid]; !ok {
			return fmt.Errorf("mock: no operation or rent %d", tzid)
		}
	}
	m.state.schedule(tzid, text, polls, smsService(opts))
	return nil
}

// FailOption configures FailNext and FailNextTransport.
type FailOption func(*failure)

// OnEndpoint limits the failure to requests to path (relative to `/api/`, e.g. "getNum.php").
func OnEndpoint(path string) FailOption { return func(f *failure) { f.endpoint = path } }

// WithStatus sets the HTTP status of the failure (FailNext default 200, FailNextTransport default 502).
func WithStatus(status int) FailOption { return func(f *failure) { f.status = status } }

// FailNext makes the next request (to one endpoint only, with OnEndpoint) answer `{"response": code}`,
// e.g. NO_NUMBER or TRY_AGAIN_LATER; the client turns it into *APIError / *NoNumberError.
func (m *Mock) FailNext(code string, opts ...FailOption) {
	m.queue(failure{code: code, status: http.StatusOK}, opts)
}

// FailNextTransport makes the next request (to one endpoint only, with OnEndpoint) fail at transport level:
// a non-JSON body with HTTP 502 (or WithStatus); the client returns *TransportError.
func (m *Mock) FailNextTransport(opts ...FailOption) {
	m.queue(failure{status: http.StatusBadGateway}, opts)
}

func (m *Mock) queue(f failure, opts []FailOption) {
	for _, opt := range opts {
		opt(&f)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.failures = append(m.state.failures, f)
}

// SetBalance sets the total balance, USD.
func (m *Mock) SetBalance(usd float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.balance = toMills(usd)
}

// Balance is the total balance, USD, including the part frozen by open operations and unconfirmed rents (as
// getBalance reports it).
func (m *Mock) Balance() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fromMills(m.state.balance)
}

// Operations returns copies of the number operations, oldest first.
func (m *Mock) Operations() []Operation {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Operation{}
	for _, op := range m.state.sortedOperations() {
		c := *op
		c.Messages = append([]SMS(nil), op.Messages...)
		out = append(out, c)
	}
	return out
}

// Rents returns copies of the open rents, oldest first.
func (m *Mock) Rents() []Rent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Rent{}
	for _, r := range m.state.sortedRents() {
		c := *r
		c.Messages = append([]RentSMS(nil), r.Messages...)
		out = append(out, c)
	}
	return out
}

// Requests returns every request received, in order.
func (m *Mock) Requests() []Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Request{}, m.state.requests...)
}

// Reset returns to the initial state: starting balance, no operations, rents, logs, schedules or failures.
func (m *Mock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.reset()
}
