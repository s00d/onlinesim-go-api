package mock_test

// The mock through the real client: every endpoint, every control helper.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
	"github.com/s00d/onlinesim-go-api/v3/mock"
)

var ctx = context.Background()

func noSleep(context.Context, time.Duration) error { return nil }

var fast = onlinesim.WaitCodeOptions{Sleep: noSleep}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func get[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		must(t, err)
		return v
	}
}

// code returns the *APIError code of err, failing the test for any other error.
func code(t *testing.T, err error) string {
	t.Helper()
	var e *onlinesim.APIError
	if !errors.As(err, &e) {
		t.Fatalf("want *APIError, got %T %v", err, err)
	}
	return e.Code
}

func eq(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %#v\nwant %#v", got, want)
	}
}

func paths(m *mock.Mock) []string {
	var out []string
	for _, r := range m.Requests() {
		out = append(out, r.Path)
	}
	return out
}

func withKey(m *mock.Mock, key string) *onlinesim.Client {
	return onlinesim.New(key, onlinesim.WithHTTPClient(&http.Client{Transport: m}))
}

// ----- auth and transport -----

// raw sends one request straight to the mock's transport and returns the status and the body.
func raw(t *testing.T, m *mock.Mock, path, key string) (int, string) {
	t.Helper()
	req := must2(http.NewRequest(http.MethodGet, "https://x.test/api/"+path, nil))
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	res, err := m.RoundTrip(req)
	must(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	must(t, err)
	return res.StatusCode, string(body)
}

func TestWrongKeyIs456MissingKeyIs200(t *testing.T) {
	m := mock.New()
	for _, key := range []string{"nope", ""} {
		_, err := withKey(m, key).User().Balance(ctx)
		eq(t, code(t, err), "ERROR_WRONG_KEY")
	}
	status, body := raw(t, m, "getBalance.php", "")
	eq(t, [2]any{status, body}, [2]any{200, `{"response":"ERROR_WRONG_KEY"}`})
	status, _ = raw(t, m, "getBalance.php", "nope")
	eq(t, status, 456)
}

// must2 drops the error of a request builder that cannot fail on a constant URL.
func must2(r *http.Request, _ error) *http.Request { return r }

func TestPublicEndpointsNeedNoKeyButCheckASentOne(t *testing.T) {
	m := mock.New()
	c := withKey(m, "")
	eq(t, get[onlinesim.CountryStat](t)(c.Numbers().TariffsOne(ctx, 44)).Code, 44)
	eq(t, get[onlinesim.RentTariff](t)(c.Rent().TariffsOne(ctx, 7)).Code, 7)
	eq(t, get[onlinesim.FreeList](t)(c.Free().List(ctx, onlinesim.FreeListOptions{})).Messages.Country, 44)
	_, err := withKey(m, "nope").Free().List(ctx, onlinesim.FreeListOptions{})
	eq(t, code(t, err), "ERROR_WRONG_KEY")
}

func TestTransportOfAHandBuiltClientAnyBaseURL(t *testing.T) {
	m := mock.New(mock.WithAPIKey("k"), mock.WithBalance(5))
	c := onlinesim.New("k", onlinesim.WithBaseURL("https://anything.test/x/api"), onlinesim.WithHTTPClient(&http.Client{Transport: m}))
	b := get[onlinesim.Balance](t)(c.User().Balance(ctx))
	if b.Balance != 5 || b.Zbalance != 0 || b.Income == nil || *b.Income != 0 {
		t.Fatalf("%+v", b)
	}
	eq(t, m.Requests(), []mock.Request{{Method: "GET", Path: "getBalance.php", Query: map[string]string{"income": "1", "lang": "en"}}})
}

func TestUnknownEndpointIsNonJSON404(t *testing.T) {
	res, err := mock.New().RoundTrip(must2(http.NewRequest(http.MethodGet, "https://x.test/api/nope.php", nil)))
	must(t, err)
	_ = res.Body.Close()
	eq(t, res.StatusCode, http.StatusNotFound)
	eq(t, res.Header.Get("Content-Type"), "text/html")
}

// ----- numbers -----

func TestPrice(t *testing.T) {
	n := mock.New().Client().Numbers()
	eq(t, *get[*float64](t)(n.Price(ctx, "telegram", 7)), 1.2)
	if p := get[*float64](t)(n.Price(ctx, "nope", 7)); p != nil {
		t.Fatalf("want nil, got %v", *p)
	}
	_, err := n.Price(ctx, "google", 999)
	eq(t, code(t, err), "UNDEFINED_COUNTRY")
}

func TestGetFreezesPriceDeterministicNumbers(t *testing.T) {
	m := mock.New()
	c := m.Client()
	eq(t, get[int64](t)(c.Numbers().Get(ctx, "google", 44)), int64(1))
	eq(t, get[onlinesim.NumberOrder](t)(c.Numbers().GetWithNumber(ctx, "vkcom", 7)),
		onlinesim.NumberOrder{Tzid: 2, Number: "+7900000002", Country: 7, Time: 1200, Service: "vkcom", Title: "vkcom"})
	var numberFlags []string
	for _, r := range m.Requests() {
		numberFlags = append(numberFlags, r.Query["number"])
	}
	eq(t, numberFlags, []string{"", "1"})
	b := get[onlinesim.Balance](t)(c.User().Balance(ctx))
	if b.Balance != 100 || b.Zbalance != 0.7 {
		t.Fatalf("%+v", b)
	}
	var got [][3]any
	for _, op := range m.Operations() {
		got = append(got, [3]any{op.Tzid, op.Number, op.Status})
	}
	eq(t, got, [][3]any{{int64(1), "+44900000001", onlinesim.TZNumWait}, {int64(2), "+7900000002", onlinesim.TZNumWait}})
}

func TestGetErrors(t *testing.T) {
	m := mock.New(mock.WithBalance(0.25))
	n := m.Client().Numbers()
	_, err := n.Get(ctx, "google", 999)
	eq(t, code(t, err), "UNDEFINED_COUNTRY")
	_, err = n.Get(ctx, "nope", 7)
	eq(t, code(t, err), "ERROR_NO_SERVICE")
	_, err = n.Get(ctx, "google", 7)
	eq(t, code(t, err), "WARNING_LOW_BALANCE")
	_, err = n.Get(ctx, "google", 44, "4490", "7")
	var noNum *onlinesim.NoNumberError
	if !errors.As(err, &noNum) {
		t.Fatalf("want *NoNumberError, got %v", err)
	}
	reqs := m.Requests()
	eq(t, reqs[len(reqs)-1].Query["reject"], "4490,7")
	// a non-matching reject passes; the rejected attempt did not consume a number
	eq(t, get[onlinesim.NumberOrder](t)(n.GetWithNumber(ctx, "google", 44, "4480")).Number, "+44900000001")
	// 0.25 - 0.2 frozen = 0.05 available
	_, err = n.Get(ctx, "google", 44)
	eq(t, code(t, err), "WARNING_LOW_BALANCE")
}

func TestStateLifecycle(t *testing.T) {
	m := mock.New()
	n := m.Client().Numbers()
	eq(t, get[[]onlinesim.Operation](t)(n.State(ctx, onlinesim.StateOptions{})), []onlinesim.Operation{})
	a := get[int64](t)(n.Get(ctx, "google", 44))
	b := get[int64](t)(n.Get(ctx, "vkcom", 7))
	one := get[onlinesim.Operation](t)(n.StateOne(ctx, a, onlinesim.StateOptions{}))
	if one.Tzid != a || one.Response != onlinesim.TZNumWait || one.Number != "+44900000001" || one.Sum != 0.2 ||
		one.Time != 1200 || one.Msg.Text != "" || one.Msg.List != nil {
		t.Fatalf("%+v", one)
	}

	must(t, m.AddSMS(a, "Your Google code is G-123 456. Do not share it."))
	one = get[onlinesim.Operation](t)(n.StateOne(ctx, a, onlinesim.StateOptions{}))
	eq(t, [2]any{one.Response, one.Msg.Text}, [2]any{onlinesim.TZNumAnswer, "123456"})
	one = get[onlinesim.Operation](t)(n.StateOne(ctx, a, onlinesim.StateOptions{FullMessage: true}))
	eq(t, one.Msg.Text, "Your Google code is G-123 456. Do not share it.")
	var order []int64
	for _, op := range get[[]onlinesim.Operation](t)(n.State(ctx, onlinesim.StateOptions{OrderBy: onlinesim.OrderDesc})) {
		order = append(order, op.Tzid)
	}
	eq(t, order, []int64{b, a})

	must(t, n.Close(ctx, a))
	one = get[onlinesim.Operation](t)(n.StateOne(ctx, a, onlinesim.StateOptions{}))
	eq(t, [2]any{one.Response, one.Time}, [2]any{onlinesim.TZOverOK, 0})
	_, err := n.StateOne(ctx, 999, onlinesim.StateOptions{})
	eq(t, code(t, err), "ERROR_NO_OPERATIONS")
}

func TestFullTextNextAndMsgList(t *testing.T) {
	m := mock.New()
	n := m.Client().Numbers()
	tzid := get[int64](t)(n.Get(ctx, "telegram", 7))
	must(t, m.AddSMS(tzid, "Welcome!", mock.FromService("telegram")))
	eq(t, get[onlinesim.Operation](t)(n.StateOne(ctx, tzid, onlinesim.StateOptions{})).Msg.Text, "Welcome!")
	must(t, n.Next(ctx, tzid))
	one := get[onlinesim.Operation](t)(n.StateOne(ctx, tzid, onlinesim.StateOptions{}))
	eq(t, [2]any{one.Response, one.Msg.Text}, [2]any{onlinesim.TZNumWait, ""})
	must(t, m.AddSMS(tzid, "code 98765"))
	eq(t, get[onlinesim.Operation](t)(n.StateOne(ctx, tzid, onlinesim.StateOptions{})).Msg.Text, "98765")
	eq(t, get[onlinesim.Operation](t)(n.StateOne(ctx, tzid, onlinesim.StateOptions{MsgList: true})).Msg.List, []onlinesim.OperationSMS{
		{Service: "telegram", Msg: "Welcome!"},
		{Service: "telegram", Msg: "98765"},
	})
	reqs := 0
	for _, p := range paths(m) {
		if p == "setOperationRevise.php" {
			reqs++
		}
	}
	eq(t, reqs, 1)
}

func TestNextAnswerAndTzidErrors(t *testing.T) {
	m := mock.New()
	n := m.Client().Numbers()
	tzid := get[int64](t)(n.Get(ctx, "google", 44))
	_, body := raw(t, m, "setOperationRevise.php?tzid=1", mock.DefaultAPIKey)
	eq(t, body, `{"response":"1","tzid":"1"}`)
	eq(t, tzid, int64(1))
	eq(t, code(t, n.Next(ctx, 424242)), "ERROR_WRONG_TZID")
	eq(t, code(t, n.Next(ctx, 0)), "ERROR_NO_TZID")
	eq(t, code(t, n.Close(ctx, 0)), "ERROR_NO_TZID")
}

func TestGetWithNumberGuardInterval(t *testing.T) {
	m := mock.New(mock.WithGuardInterval(60))
	o := get[onlinesim.NumberOrder](t)(m.Client().Numbers().GetWithNumber(ctx, "google", 44))
	eq(t, [2]any{o.Title, o.GuardIntervalRemainingSeconds}, [2]any{"google", 60})
	_, body := raw(t, m, "getNum.php?service=google&country=44&number=1", mock.DefaultAPIKey)
	if !strings.Contains(body, `"response_text":"Waiting for SMS"`) {
		t.Fatal(body)
	}
}

func TestCloseSpendsOrRefundsBanAfterSMS(t *testing.T) {
	m := mock.New(mock.WithBalance(10))
	c := m.Client()
	n := c.Numbers()
	paid := get[int64](t)(n.Get(ctx, "telegram", 7))
	empty := get[int64](t)(n.Get(ctx, "telegram", 7))
	must(t, m.AddSMS(paid, "1111"))
	eq(t, code(t, n.Ban(ctx, paid)), "0")
	must(t, n.Close(ctx, paid))
	must(t, n.Close(ctx, empty))
	b := get[onlinesim.Balance](t)(c.User().Balance(ctx))
	if b.Balance != 8.8 || b.Zbalance != 0 {
		t.Fatalf("%+v", b)
	}
	eq(t, m.Balance(), 8.8)
	eq(t, code(t, n.Close(ctx, paid)), "ERROR_WRONG_TZID")
	eq(t, code(t, n.Close(ctx, 999)), "ERROR_WRONG_TZID")
	banned := get[int64](t)(n.Get(ctx, "google", 7))
	must(t, n.Ban(ctx, banned))
	ops := m.Operations()
	eq(t, ops[len(ops)-1].Status, onlinesim.TZOverOK)
}

func TestGuardInterval(t *testing.T) {
	m := mock.New(mock.WithGuardInterval(120))
	n := m.Client().Numbers()
	tzid := get[int64](t)(n.Get(ctx, "google", 7))
	eq(t, get[onlinesim.Operation](t)(n.StateOne(ctx, tzid, onlinesim.StateOptions{})).GuardIntervalRemainingSeconds, 120)
	eq(t, code(t, n.Close(ctx, tzid)), "NO_COMPLETE_TZID")
	must(t, m.AddSMS(tzid, "5555"))
	eq(t, get[onlinesim.Operation](t)(n.StateOne(ctx, tzid, onlinesim.StateOptions{})).GuardIntervalRemainingSeconds, 0)
	must(t, n.Close(ctx, tzid))
}

func TestWaitCodeWithAddSMSAfterNoClock(t *testing.T) {
	m := mock.New()
	n := m.Client().Numbers()
	tzid := get[int64](t)(n.Get(ctx, "google", 44))
	must(t, m.AddSMSAfter(tzid, "Your code: 4321", 2))
	eq(t, get[string](t)(n.WaitCode(ctx, tzid, fast)), "4321")
	eq(t, paths(m), []string{"getNum.php", "getState.php", "getState.php", "getState.php", "setOperationOk.php"})
	eq(t, m.Operations()[0].Status, onlinesim.TZOverOK)
}

func TestWaitCodeKeepOpenSinceAndListPolls(t *testing.T) {
	m := mock.New()
	n := m.Client().Numbers()
	tzid := get[int64](t)(n.Get(ctx, "google", 44))
	must(t, m.AddSMSAfter(tzid, "1111", 0))
	opts := fast
	opts.KeepOpen = true
	first := get[string](t)(n.WaitCode(ctx, tzid, opts))
	must(t, m.AddSMSAfter(tzid, "2222", 1))
	_ = get[[]onlinesim.Operation](t)(n.State(ctx, onlinesim.StateOptions{})) // one poll of every operation
	opts = fast
	opts.Since = first
	eq(t, get[string](t)(n.WaitCode(ctx, tzid, opts)), "2222")
}

func TestTariffsFromPrices(t *testing.T) {
	c := mock.New().Client()
	all := get[map[string]onlinesim.CountryStat](t)(c.Numbers().Tariffs(ctx))
	eq(t, len(all), 2)
	eq(t, all["44"].Services["service_vkcom"], onlinesim.ServiceStat{Count: 100, Code: 44, Price: 0.36, ID: 2, Service: "vkcom", Slug: "vkcom"})
	_, err := c.Numbers().TariffsOne(ctx, 1)
	eq(t, code(t, err), "NO_COUNTRY")
	empty := mock.New(mock.WithPrices(mock.Prices{1: {}})).Client()
	eq(t, get[onlinesim.CountryStat](t)(empty.Numbers().TariffsOne(ctx, 1)).Services, map[string]onlinesim.ServiceStat{}) // `[]` on the wire
}

// ----- rent -----

func TestRentReserveConfirmExtendClose(t *testing.T) {
	m := mock.New(mock.WithBalance(10))
	c := m.Client()
	r := c.Rent()
	rent := get[onlinesim.RentItem](t)(r.Get(ctx, 7, 3, true))
	if rent.Tzid != 1 || rent.Number != "900000001" || rent.Status != 0 || rent.Checked || rent.Sum != 2.5 || rent.Days != 0 ||
		rent.Time != 20 || rent.TimeSeconds != 1200 || rent.Rent != 3 || rent.DaysAutoextend != 3 || rent.Extension != 1 ||
		rent.GuardIntervalRemainingSeconds != 0 || len(rent.Messages) != 0 {
		t.Fatalf("%+v", rent)
	}
	eq(t, rent.Extend, map[string]float64{"1": 1, "3": 2.5, "7": 5})
	// reserved: the total balance stays, the price is frozen
	eq(t, m.Balance(), 10.0)
	b := get[onlinesim.Balance](t)(c.User().Balance(ctx))
	eq(t, [2]float64{b.Balance, b.Zbalance}, [2]float64{10, 2.5})
	_, err := r.Extend(ctx, rent.Tzid, 1)
	eq(t, code(t, err), "TRY_AGAIN_LATER") // not confirmed yet

	must(t, m.AddSMS(rent.Tzid, "Bank: 777 888", mock.FromService("bank")))
	must(t, m.AddSMS(rent.Tzid, "no code here"))
	state := get[onlinesim.RentItem](t)(r.StateOne(ctx, rent.Tzid))
	if state.Status != 1 || !state.Checked || state.Days != 3 || state.Time != 3*1440 || state.TimeSeconds != 3*86400 {
		t.Fatalf("%+v", state)
	}
	var msgs [][2]string
	for _, sms := range state.Messages {
		msgs = append(msgs, [2]string{sms.Service, sms.Code})
	}
	eq(t, msgs, [][2]string{{"other", ""}, {"bank", "777888"}}) // newest first
	eq(t, m.Balance(), 7.5)
	b = get[onlinesim.Balance](t)(c.User().Balance(ctx))
	eq(t, [2]float64{b.Balance, b.Zbalance}, [2]float64{7.5, 0})

	ext := get[onlinesim.RentItem](t)(r.Extend(ctx, rent.Tzid, 1))
	eq(t, [4]any{ext.Days, ext.Rent, ext.DaysAutoextend, ext.Sum}, [4]any{4, 4, 3, 3.5})
	eq(t, m.Balance(), 6.5)
	_, err = r.Extend(ctx, rent.Tzid, 2)
	eq(t, code(t, err), "TRY_AGAIN_LATER") // unknown period
	_, err = r.Extend(ctx, 999, 1)
	eq(t, code(t, err), "TRY_AGAIN_LATER") // unknown tzid

	fixed := get[onlinesim.RentItem](t)(r.Get(ctx, 7, 1, false))
	must(t, m.AddSMS(fixed.Tzid, "1234"))
	fixed = get[onlinesim.RentItem](t)(r.StateOne(ctx, fixed.Tzid))
	eq(t, [2]any{fixed.Extension, fixed.Extend}, [2]any{0, map[string]float64{}})
	_, err = r.Extend(ctx, fixed.Tzid, 1)
	eq(t, code(t, err), "TRY_AGAIN_LATER")
	var ids []int64
	for _, it := range get[[]onlinesim.RentItem](t)(r.State(ctx)) {
		ids = append(ids, it.Tzid)
	}
	eq(t, ids, []int64{rent.Tzid, fixed.Tzid})

	must(t, r.Close(ctx, fixed.Tzid)) // confirmed: no refund
	eq(t, m.Balance(), 5.5)
	eq(t, len(m.Rents()), 1)
	eq(t, m.Rents()[0].Tzid, rent.Tzid)
	_, err = r.StateOne(ctx, fixed.Tzid)
	eq(t, code(t, err), "ERROR_NO_OPERATIONS")
	eq(t, code(t, r.Close(ctx, fixed.Tzid)), "ERROR_NO_OPERATIONS")
}

func TestRentSumNumberOrStringOnTheWire(t *testing.T) {
	m := mock.New()
	_, got := raw(t, m, "rent/getRentNum.php?days=3", mock.DefaultAPIKey)
	_, listed := raw(t, m, "rent/getRentState.php", mock.DefaultAPIKey)
	if !strings.Contains(got, `"sum":2.5,`) || !strings.Contains(listed, `"sum":"2.50"`) {
		t.Fatalf("%s\n%s", got, listed)
	}
}

func TestRentUnconfirmedCloseNotifyGuard(t *testing.T) {
	m := mock.New(mock.WithBalance(10), mock.WithGuardInterval(120))
	c := m.Client()
	r := c.Rent()
	rent := get[onlinesim.RentItem](t)(r.Get(ctx, 7, 3, true))
	eq(t, rent.GuardIntervalRemainingSeconds, 120)
	eq(t, code(t, r.Close(ctx, rent.Tzid)), "NO_COMPLETE_TZID")
	must(t, m.AddSMS(rent.Tzid, "Messages from this sender are not accepted", mock.FromService("notify")))
	state := get[onlinesim.RentItem](t)(r.StateOne(ctx, rent.Tzid))
	eq(t, [3]any{state.Status, state.Checked, state.GuardIntervalRemainingSeconds}, [3]any{0, false, 0})
	eq(t, [2]string{state.Messages[0].Service, state.Messages[0].Code}, [2]string{"notify", "0"})
	_, body := raw(t, m, "rent/getRentState.php", mock.DefaultAPIKey)
	if !strings.Contains(body, `"code":0,`) {
		t.Fatal(body) // the integer 0 on the wire
	}
	must(t, r.Close(ctx, rent.Tzid))
	eq(t, m.Balance(), 10.0) // released, nothing charged
	b := get[onlinesim.Balance](t)(c.User().Balance(ctx))
	eq(t, [2]float64{b.Balance, b.Zbalance}, [2]float64{10, 0})
}

func TestRentErrors(t *testing.T) {
	r := mock.New(mock.WithBalance(1.5)).Client().Rent()
	_, err := r.Get(ctx, 44, 1, true)
	eq(t, code(t, err), "UNDEFINED_COUNTRY")
	_, err = r.Get(ctx, 7, 2, true)
	eq(t, code(t, err), "UNDEFINED_DAYS")
	_, err = r.Get(ctx, 7, 3, true)
	eq(t, code(t, err), "WARNING_LOW_BALANCE")
}

func TestRentAddSMSAfterAndTariffs(t *testing.T) {
	m := mock.New()
	r := m.Client().Rent()
	tzid := get[onlinesim.RentItem](t)(r.Get(ctx, 7, 1, true)).Tzid
	must(t, m.AddSMSAfter(tzid, "6060", 1))
	eq(t, len(get[onlinesim.RentItem](t)(r.StateOne(ctx, tzid)).Messages), 0)
	item := get[[]onlinesim.RentItem](t)(r.State(ctx))[0]
	eq(t, [3]any{item.Status, item.Messages[0].Code, item.Messages[0].Service}, [3]any{1, "6060", "other"})
	count := map[string]int{"1": 100, "3": 100, "7": 100}
	eq(t, get[map[string]onlinesim.RentTariff](t)(r.Tariffs(ctx)), map[string]onlinesim.RentTariff{
		"7": {Code: 7, Name: "russia", Enabled: true, Position: 7, Extend: 1, Days: map[string]float64{"1": 1, "3": 2.5, "7": 5}, Count: count, ExtCounts: count},
	})
	_, err := r.TariffsOne(ctx, 44)
	eq(t, code(t, err), "NO_COUNTRY")
}

// ----- user and free -----

func TestProfileWebhookAndLogs(t *testing.T) {
	m := mock.New(mock.WithAPIKey("secret"))
	c := m.Client()
	u := c.User()
	p := get[onlinesim.Profile](t)(u.Profile(ctx))
	eq(t, [3]any{p.ID, p.APIKey, p.WebhookURL}, [3]any{int64(1), "secret", ""})
	err := u.SetWebhookURL(ctx, "not a url")
	var apiErr *onlinesim.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "VALIDATION_ERROR" || apiErr.Message != "The webhook url must be a valid URL." {
		t.Fatalf("%v", err)
	}
	must(t, u.SetWebhookURL(ctx, "ftp://x")) // any scheme://… URL
	must(t, u.SetWebhookURL(ctx, "https://example.com/hook"))
	reqs := m.Requests()
	last := reqs[len(reqs)-1]
	eq(t, [3]any{last.Method, last.Path, last.Body}, [3]any{"POST", "profile.php", map[string]any{"profile": map[string]any{"webhook_url": "https://example.com/hook"}}})
	eq(t, get[onlinesim.Profile](t)(u.Profile(ctx)).WebhookURL, "https://example.com/hook")

	tzid := get[int64](t)(c.Numbers().Get(ctx, "google", 44))
	must(t, m.AddSMS(tzid, "code 1234"))
	rent := get[onlinesim.RentItem](t)(c.Rent().Get(ctx, 7, 1, true))
	must(t, m.AddSMS(rent.Tzid, "code 5678"))
	must(t, m.AddSMS(rent.Tzid, "no code"))
	logs := get[onlinesim.Paginator[onlinesim.WebhookLog]](t)(u.WebhookLogs(ctx, 1))
	eq(t, [3]int{logs.Total, logs.CurrentPage, logs.PerPage}, [3]int{3, 1, 10})
	var hooks []onlinesim.WebhookPayload
	for _, l := range logs.Data {
		hooks = append(hooks, get[onlinesim.WebhookPayload](t)(onlinesim.ParseWebhook(l.Params)))
	}
	eq(t, [3]onlinesim.WebhookType{hooks[0].WebhookType, hooks[1].WebhookType, hooks[2].WebhookType},
		[3]onlinesim.WebhookType{onlinesim.WebhookRentSMS, onlinesim.WebhookRentSMS, onlinesim.WebhookReceivingSMS})
	eq(t, [3]string{hooks[0].Code, hooks[1].Code, hooks[2].Code}, [3]string{"", "5678", "1234"})
	eq(t, [2]int64{hooks[0].OperationID, hooks[2].OperationID}, [2]int64{rent.Tzid, tzid})
	// activation: +number, seconds left, order time; rent: digits without +, minutes left, SMS time
	eq(t, [3]any{hooks[2].Number, hooks[2].TimeLeft, hooks[2].TimeStart}, [3]any{"+44900000001", int64(1200), "2026-01-01 00:00:01"})
	eq(t, [3]any{hooks[1].Number, hooks[1].TimeLeft, hooks[1].TimeStart}, [3]any{"7900000002", int64(1440), "2026-01-01 00:00:03"})

	must(t, u.ClearWebhookURL(ctx))
	eq(t, get[onlinesim.Profile](t)(u.Profile(ctx)).WebhookURL, "")
}

func TestBalanceIncludesFrozenPaymentHistoryEmpty(t *testing.T) {
	m := mock.New(mock.WithBalance(3))
	c := m.Client()
	_ = get[int64](t)(c.Numbers().Get(ctx, "telegram", 7))
	b := get[onlinesim.Balance](t)(c.User().Balance(ctx))
	if b.Balance != 3 || b.Zbalance != 1.2 {
		t.Fatalf("%+v", b)
	}
	h := get[onlinesim.PaymentHistory](t)(c.User().PaymentHistory(ctx, 1))
	if len(h.Paylist) != 0 || h.Paylist == nil || len(h.Orders.Data) != 0 || h.Orders.Total != 0 || h.Orders.CurrentPage != 1 {
		t.Fatalf("%+v", h)
	}
	eq(t, [3]any{h.Forms, h.Buttons, h.Domains}, [3]any{map[string]string{"1": "Card"}, map[string][]float64{"USD": {5, 10, 20}}, []string{"onlinesim.io"}})
}

func TestFreeList(t *testing.T) {
	f := mock.New().Client().Free()
	uk := get[onlinesim.FreeList](t)(f.List(ctx, onlinesim.FreeListOptions{}))
	eq(t, []int{uk.Countries[0].Country, uk.Countries[1].Country}, []int{44, 7})
	eq(t, len(uk.Numbers), 1)
	eq(t, uk.Numbers["7459608320"].FullNumber, "447459608320")
	eq(t, [4]any{uk.Messages.Number, uk.Messages.Country, uk.Messages.Total, uk.Messages.PerPage}, [4]any{"7459608320", 44, 2, 10})
	eq(t, [2]string{uk.Messages.Data[0].Code, uk.Messages.Data[0].CreatedAt}, [2]string{"195895", "2026-01-01T00:00:00.000000Z"})
	eq(t, get[onlinesim.FreeList](t)(f.List(ctx, onlinesim.FreeListOptions{Count: 1})).Messages.PerPage, 10) // clamped to 10..500
	eq(t, get[onlinesim.FreeList](t)(f.List(ctx, onlinesim.FreeListOptions{Count: 9999})).Messages.PerPage, 500)
	eq(t, get[onlinesim.FreeList](t)(f.List(ctx, onlinesim.FreeListOptions{Country: 999})).Messages.Country, 44) // unknown → the first
	_, body := raw(t, mock.New(), "getFreeList", "")
	if !strings.Contains(body, `"content":null`) || !strings.Contains(body, `"ignore":"`) {
		t.Fatal(body)
	}
	ru := get[onlinesim.FreeList](t)(f.List(ctx, onlinesim.FreeListOptions{Country: 7}))
	eq(t, [2]string{ru.Messages.Data[0].Code, ru.Messages.Data[0].MyNumber}, [2]string{"51423", "9001234567"})
}

// ----- control API -----

func TestFailNext(t *testing.T) {
	m := mock.New()
	c := m.Client()
	m.FailNext("NO_NUMBER", mock.OnEndpoint("getNum.php"))
	eq(t, get[onlinesim.Balance](t)(c.User().Balance(ctx)).Balance, 100.0) // other endpoint: untouched
	_, err := c.Numbers().Get(ctx, "google", 7)
	var noNum *onlinesim.NoNumberError
	if !errors.As(err, &noNum) {
		t.Fatalf("want *NoNumberError, got %v", err)
	}
	eq(t, get[int64](t)(c.Numbers().Get(ctx, "google", 7)), int64(1))
	m.FailNext("TRY_AGAIN_LATER")
	_, err = c.User().Profile(ctx)
	eq(t, code(t, err), "TRY_AGAIN_LATER")
	m.FailNext("ERROR_WRONG_KEY", mock.WithStatus(456))
	_, err = c.User().Profile(ctx)
	eq(t, code(t, err), "ERROR_WRONG_KEY")
}

func TestFailNextTransportRetriedByWaitCode(t *testing.T) {
	m := mock.New()
	c := m.Client()
	m.FailNextTransport()
	_, err := c.User().Balance(ctx)
	var te *onlinesim.TransportError
	if !errors.As(err, &te) || te.Status != 502 {
		t.Fatalf("want *TransportError 502, got %v", err)
	}
	tzid := get[int64](t)(c.Numbers().Get(ctx, "google", 7))
	must(t, m.AddSMS(tzid, "3030"))
	m.FailNextTransport(mock.WithStatus(503), mock.OnEndpoint("getState.php"))
	eq(t, get[string](t)(c.Numbers().WaitCode(ctx, tzid, fast)), "3030")
	eq(t, paths(m)[2:], []string{"getState.php", "getState.php", "setOperationOk.php"})
}

func TestUnknownIDsSetBalanceRequestsReset(t *testing.T) {
	m := mock.New(mock.WithBalance(1))
	c := m.Client(onlinesim.WithLang("ru"))
	if err := m.AddSMS(5, "x"); err == nil || !strings.Contains(err.Error(), "no open operation or rent 5") {
		t.Fatalf("%v", err)
	}
	if err := m.AddSMSAfter(5, "x", 1); err == nil || !strings.Contains(err.Error(), "no operation or rent 5") {
		t.Fatalf("%v", err)
	}
	m.SetBalance(50)
	eq(t, m.Balance(), 50.0)
	tzid := get[int64](t)(c.Numbers().Get(ctx, "telegram", 7))
	must(t, c.Numbers().Close(ctx, tzid))
	if m.AddSMS(tzid, "late") == nil {
		t.Fatal("AddSMS to a closed operation must fail")
	}
	eq(t, m.Requests()[0], mock.Request{Method: "GET", Path: "getNum.php", Query: map[string]string{"service": "telegram", "country": "7", "lang": "ru"}})

	m.FailNext("NO_NUMBER")
	m.Reset()
	eq(t, m.Balance(), 1.0)
	eq(t, m.Operations(), []mock.Operation{})
	eq(t, m.Rents(), []mock.Rent{})
	eq(t, len(m.Requests()), 0)
	eq(t, get[int64](t)(c.Numbers().Get(ctx, "google", 44)), int64(1)) // ids restart, failure queue cleared
}

func TestOperationsReturnsCopies(t *testing.T) {
	m := mock.New()
	tzid := get[int64](t)(m.Client().Numbers().Get(ctx, "google", 7))
	must(t, m.AddSMS(tzid, "1234"))
	ops := m.Operations()
	ops[0].Status = onlinesim.TZOverOK
	ops[0].Messages[0].Text = "changed"
	eq(t, m.Operations()[0].Status, onlinesim.TZNumAnswer)
	eq(t, m.Operations()[0].Messages[0].Text, "1234")
}

func TestCodeExtraction(t *testing.T) {
	m := mock.New()
	n := m.Client().Numbers()
	for text, want := range map[string]string{
		"G-123 456":            "123456",
		"code 12-34":           "1234",
		"123":                  "123",
		"call 123456789 now":   "call 123456789 now",
		"pin 4321, ref 98765":  "4321",
		"Your code: 1 2 3 4 5": "12345",
	} {
		tzid := get[int64](t)(n.Get(ctx, "google", 7))
		must(t, m.AddSMS(tzid, text))
		eq(t, get[onlinesim.Operation](t)(n.StateOne(ctx, tzid, onlinesim.StateOptions{})).Msg.Text, want)
	}
}

// ----- HTTP server -----

func TestServerServesTheSameMock(t *testing.T) {
	srv := mock.Start(mock.WithBalance(10))
	defer srv.Close()
	if !strings.HasPrefix(srv.BaseURL(), "http://127.0.0.1:") || !strings.HasSuffix(srv.BaseURL(), "/api/") {
		t.Fatal(srv.BaseURL())
	}
	c := srv.Client()
	n := c.Numbers()
	tzid := get[int64](t)(n.Get(ctx, "google", 44))
	must(t, srv.AddSMSAfter(tzid, "Your code: 123-456", 2))
	eq(t, get[string](t)(n.WaitCode(ctx, tzid, fast)), "123456")
	eq(t, srv.Balance(), 9.8)
	_, err := onlinesim.New("wrong", onlinesim.WithBaseURL(srv.BaseURL())).User().Balance(ctx)
	eq(t, code(t, err), "ERROR_WRONG_KEY")
	res, err := http.Get(srv.BaseURL() + "nope.php")
	must(t, err)
	_ = res.Body.Close()
	eq(t, res.StatusCode, http.StatusNotFound)
	eq(t, paths(srv.Mock)[0:5], []string{"getNum.php", "getState.php", "getState.php", "getState.php", "setOperationOk.php"})
}
