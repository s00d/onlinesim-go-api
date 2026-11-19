package onlinesim_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

var ctx = context.Background()

// reply is a canned answer of the fake transport: a body with a status (default 200), or a transport error.
type reply struct {
	status int
	body   string
	err    error
}

func ok(body string) reply { return reply{body: body} }

// call is a recorded request.
type call struct {
	method string
	path   string
	query  map[string]string // without the `lang` every request carries
	lang   string
	header http.Header
	body   string
}

// fake is an http.RoundTripper that answers replies in order and records every request.
type fake struct {
	t       *testing.T
	mu      sync.Mutex
	replies []reply
	calls   []call
}

func (f *fake) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	q := map[string]string{}
	for k, v := range r.URL.Query() {
		q[k] = strings.Join(v, "|")
	}
	lang := q["lang"]
	delete(q, "lang")
	f.calls = append(f.calls, call{r.Method, r.URL.Path, q, lang, r.Header.Clone(), string(body)})
	if len(f.replies) == 0 {
		f.t.Errorf("unexpected request %s", r.URL)
		return nil, errors.New("unexpected request")
	}
	rep := f.replies[0]
	f.replies = f.replies[1:]
	if rep.err != nil {
		return nil, rep.err
	}
	status := rep.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(rep.body)), Header: http.Header{}, Request: r}, nil
}

// last is the only request made; it fails the test when there were more or none.
func (f *fake) last() call {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) != 1 {
		f.t.Fatalf("expected 1 request, got %d", len(f.calls))
	}
	return f.calls[0]
}

func (f *fake) paths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	for i, c := range f.calls {
		out[i] = c.path
	}
	return out
}

// fakeClient is a client with key "test-key" on base https://api.test/api/ and a fake transport.
func fakeClient(t *testing.T, replies []reply, opts ...onlinesim.Option) (*onlinesim.Client, *fake) {
	t.Helper()
	f := &fake{t: t, replies: replies}
	base := []onlinesim.Option{onlinesim.WithBaseURL("https://api.test/api/"), onlinesim.WithHTTPClient(&http.Client{Transport: f})}
	return onlinesim.New("test-key", append(base, opts...)...), f
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func eq[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\n got: %#v\nwant: %#v", got, want)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// apiErr asserts err is an *APIError with code and returns a copy of it.
func apiErr(t *testing.T, err error, code string) onlinesim.APIError {
	t.Helper()
	var e *onlinesim.APIError
	if !errors.As(err, &e) {
		t.Fatalf("want *APIError %s, got %T %v", code, err, err)
	}
	if e.Code != code {
		t.Fatalf("want code %s, got %s (%v)", code, e.Code, err)
	}
	return *e
}
