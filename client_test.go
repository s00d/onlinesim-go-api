package onlinesim_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func TestBearerHeaderNeverQueryPlusLang(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":"1","price":0.36}`)})
	_, err := c.Numbers().Price(ctx, "vkcom", 44)
	must(t, err)
	call := f.last()
	eq(t, call.header.Get("Authorization"), "Bearer test-key")
	eq(t, call.header.Get("Accept"), "application/json")
	eq(t, call.lang, "en")
	_, hasKey := call.query["apikey"]
	_, hasDev := call.query["dev_id"]
	eq(t, hasKey || hasDev, false)
}

func TestLangDevIDAndBaseWithoutSlash(t *testing.T) {
	f := &fake{t: t, replies: []reply{ok(`{"response":"1","price":1}`)}}
	var host string
	c := onlinesim.New("k",
		onlinesim.WithBaseURL("https://api.test/api"),
		onlinesim.WithLang("ru"),
		onlinesim.WithDevID(42),
		onlinesim.WithHTTPClient(&http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			host = r.URL.Host
			return f.RoundTrip(r)
		})}),
	)
	_, err := c.Numbers().Price(ctx, "vkcom", 7)
	must(t, err)
	call := f.last()
	eq(t, host+call.path, "api.test/api/getPrice.php")
	eq(t, call.lang, "ru")
	eq(t, call.query["dev_id"], "42")
}

func TestNoAuthorizationWithoutKey(t *testing.T) {
	f := &fake{t: t, replies: []reply{ok(`{"response":"1","price":1}`)}}
	c := onlinesim.New("", onlinesim.WithHTTPClient(&http.Client{Transport: f}))
	_, err := c.Numbers().Price(ctx, "vkcom", 7)
	must(t, err)
	eq(t, f.last().header.Get("Authorization"), "")
}

func TestDefaultBaseURL(t *testing.T) {
	var got string
	c := onlinesim.New("k", onlinesim.WithHTTPClient(&http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		got = r.URL.Scheme + "://" + r.URL.Host + r.URL.Path
		return (&fake{t: t, replies: []reply{ok(`{"response":"1","price":1}`)}}).RoundTrip(r)
	})}))
	_, err := c.Numbers().Price(ctx, "vkcom", 7)
	must(t, err)
	eq(t, got, "https://onlinesim.host/api/getPrice.php")
}

func TestNoNumberIsNoNumberErrorAndAPIError(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":"NO_NUMBER"}`), ok(`{"response":"FORWARD_NOT_SUPPORTED"}`)})
	_, err := c.Numbers().Get(ctx, "google", 44)
	var noNum *onlinesim.NoNumberError
	if !errors.As(err, &noNum) {
		t.Fatalf("want *NoNumberError, got %T %v", err, err)
	}
	eq(t, apiErr(t, err, "NO_NUMBER").Message, "No numbers available for this service right now")

	// Only NO_NUMBER is a NoNumberError.
	_, err = c.Numbers().Get(ctx, "google", 44)
	eq(t, errors.As(err, &noNum), false)
	eq(t, apiErr(t, err, "FORWARD_NOT_SUPPORTED").Message, "Number forwarding is no longer available")
}

func TestHTTP429TryAgainLaterIsAPIErrorNotTransport(t *testing.T) {
	c, _ := fakeClient(t, []reply{{status: 429, body: `{"response":"TRY_AGAIN_LATER"}`}})
	_, err := c.Numbers().State(ctx, onlinesim.StateOptions{})
	eq(t, apiErr(t, err, "TRY_AGAIN_LATER").Message, "Rate limit exceeded or temporary error, try again later")
	var te *onlinesim.TransportError
	eq(t, errors.As(err, &te), false)
}

func TestMissingKeyHTTP200WrongKeyIsAPIError(t *testing.T) {
	f := &fake{t: t, replies: []reply{ok(`{"response":"ERROR_WRONG_KEY"}`)}}
	c := onlinesim.New("", onlinesim.WithHTTPClient(&http.Client{Transport: f}))
	_, err := c.User().Balance(ctx)
	eq(t, apiErr(t, err, "ERROR_WRONG_KEY").Message, "Wrong or missing API key")
}

func TestNewErrorMessages(t *testing.T) {
	for code, want := range map[string]string{
		"ACCOUNT_CLEANED":    "Account was deleted",
		"USER_NOT_ACTIVATED": "Account email is not confirmed",
		"ERROR_UNDEFINED":    "Unexpected server error, try again",
		"ERROR_NO_KEY":       "ERROR_NO_KEY", // no longer sent: no fallback text
		"API_DISABLED":       "API_DISABLED",
	} {
		c, _ := fakeClient(t, []reply{ok(`{"response":"` + code + `"}`)})
		_, err := c.User().Balance(ctx)
		eq(t, apiErr(t, err, code).Message, want)
	}
}

func TestHTTP456WrongKeyIsAPIErrorNotTransport(t *testing.T) {
	c, _ := fakeClient(t, []reply{{status: 456, body: `{"response":"ERROR_WRONG_KEY"}`}})
	_, err := c.User().Balance(ctx)
	e := apiErr(t, err, "ERROR_WRONG_KEY")
	eq(t, e.Message, "Wrong or missing API key")
	var te *onlinesim.TransportError
	eq(t, errors.As(err, &te), false)
}

func TestServerMessageUnlessItEchoesTheCode(t *testing.T) {
	c, _ := fakeClient(t, []reply{
		ok(`{"response":"TRY_AGAIN_LATER","message":"Maintenance until 12:00"}`),
		ok(`{"response":"UNDEFINED_COUNTRY","message":"UNDEFINED_COUNTRY"}`),
		ok(`{"response":"SOMETHING_NEW"}`),
	})
	for _, want := range [][2]string{
		{"TRY_AGAIN_LATER", "Maintenance until 12:00"},
		{"UNDEFINED_COUNTRY", "Unknown country"},
		{"SOMETHING_NEW", "SOMETHING_NEW"},
	} {
		_, err := c.Numbers().Get(ctx, "x", 7)
		eq(t, apiErr(t, err, want[0]).Message, want[1])
	}
}

func TestValidationObjectIsValidationError(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":{"webhook_url":["The webhook url must be a valid URL.","Too long."],"name":["Required."]}}`)})
	err := c.User().SetWebhookURL(ctx, "nope")
	eq(t, apiErr(t, err, "VALIDATION_ERROR").Message, "The webhook url must be a valid URL.; Too long.; Required.")
}

func TestNonJSONIsTransportErrorWithStatusAndBody(t *testing.T) {
	c, _ := fakeClient(t, []reply{{status: 502, body: "<html>Bad gateway</html>"}})
	_, err := c.User().Balance(ctx)
	var te *onlinesim.TransportError
	if !errors.As(err, &te) {
		t.Fatalf("want *TransportError, got %T %v", err, err)
	}
	eq(t, te.Status, 502)
	eq(t, te.Body, "<html>Bad gateway</html>")
}

func TestBodyExcerptIs500Bytes(t *testing.T) {
	c, _ := fakeClient(t, []reply{{status: 200, body: strings.Repeat("x", 2000)}})
	_, err := c.User().Balance(ctx)
	var te *onlinesim.TransportError
	if !errors.As(err, &te) {
		t.Fatal(err)
	}
	eq(t, len(te.Body), 500)
}

func TestJSONErrorStatusWithoutResponseIsTransportError(t *testing.T) {
	c, _ := fakeClient(t, []reply{{status: 500, body: `{"error":"boom"}`}})
	_, err := c.User().Balance(ctx)
	var te *onlinesim.TransportError
	if !errors.As(err, &te) || te.Status != 500 {
		t.Fatalf("want TransportError 500, got %v", err)
	}
}

func TestNetworkFailureIsTransportErrorWithoutStatus(t *testing.T) {
	cause := errors.New("connection reset")
	c, _ := fakeClient(t, []reply{{err: cause}})
	_, err := c.User().Balance(ctx)
	var te *onlinesim.TransportError
	if !errors.As(err, &te) {
		t.Fatalf("want *TransportError, got %T %v", err, err)
	}
	eq(t, te.Status, 0)
	eq(t, errors.Is(err, cause), true)
}

// The transport blocks until the request context ends: the per-request timeout is the only way out.
func TestRequestTimeoutIsTransportError(t *testing.T) {
	block := roundTrip(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	c := onlinesim.New("k", onlinesim.WithTimeout(time.Millisecond), onlinesim.WithHTTPClient(&http.Client{Transport: block}))
	_, err := c.User().Balance(ctx)
	var te *onlinesim.TransportError
	if !errors.As(err, &te) {
		t.Fatalf("want *TransportError, got %T %v", err, err)
	}
	eq(t, errors.Is(err, context.DeadlineExceeded), true)
	eq(t, te.Error(), "onlinesim: request timed out after 1ms")
}

func TestCanceledContextIsTransportErrorWrappingCanceled(t *testing.T) {
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	block := roundTrip(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	c := onlinesim.New("k", onlinesim.WithHTTPClient(&http.Client{Transport: block}))
	_, err := c.User().Balance(canceled)
	var te *onlinesim.TransportError
	eq(t, errors.As(err, &te), true)
	eq(t, errors.Is(err, context.Canceled), true)
}

func TestBodiesWithoutResponseAreReturnedAsIs(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`[]`)})
	list, err := c.Numbers().State(ctx, onlinesim.StateOptions{})
	must(t, err)
	eq(t, list, []onlinesim.Operation{})
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
