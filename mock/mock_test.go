package mock_test

// The mock through the real client: every endpoint, every control helper.

import (
	"context"
	"errors"
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

func TestWrongOrMissingKeyIs456WrongKey(t *testing.T) {
	m := mock.New()
	for _, key := range []string{"nope", ""} {
		_, err := withKey(m, key).User().Balance(ctx)
		eq(t, code(t, err), "ERROR_WRONG_KEY")
	}
	res, err := m.RoundTrip(must2(http.NewRequest(http.MethodGet, "https://x.test/api/getBalance.php", nil)))
	must(t, err)
	_ = res.Body.Close()
	eq(t, res.StatusCode, 456)
}

// must2 drops the error of a request builder that cannot fail on a constant URL.
func must2(r *http.Request, _ error) *http.Request { return r }

func TestPublicEndpointsNeedNoKey(t *testing.T) {
	c := withKey(mock.New(), "")
	eq(t, get[onlinesim.CountryStat](t)(c.Numbers().TariffsOne(ctx, 44)).Code, 44)
	eq(t, get[onlinesim.RentTariff](t)(c.Rent().TariffsOne(ctx, 7)).Code, 7)
	eq(t, get[onlinesim.FreeList](t)(c.Free().List(ctx, onlinesim.FreeListOptions{})).Messages.Country, 44)
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
		onlinesim.NumberOrder{Tzid: 2, Number: "+7900000002", Country: 7, Time: 1200, Service: "vkcom"})
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
	must(t, n.Next(ctx, 424242)) // always succeeds, like the backend
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

func TestRentLifecycle(t *testing.T) {
	m := mock.New(mock.WithBalance(10))
	r := m.Client().Rent()
	rent := get[onlinesim.RentItem](t)(r.Get(ctx, 7, 3, true))
	if rent.Tzid != 1 || rent.Number != "900000001" || rent.Sum != 2.5 || rent.Days != 3 || rent.Time != 3*1440 ||
		rent.TimeSeconds != 3*86400 || rent.Extension != 1 || len(rent.Messages) != 0 {
		t.Fatalf("%+v", rent)
	}
	eq(t, rent.Extend, map[string]float64{"1": 1, "3": 2.5, "7": 5})
	eq(t, m.Balance(), 7.5)

	must(t, m.AddSMS(rent.Tzid, "Bank: 777 888", mock.FromService("bank")))
	must(t, m.AddSMS(rent.Tzid, "no code here"))
	var msgs [][2]string
	for _, sms := range get[onlinesim.RentItem](t)(r.StateOne(ctx, rent.Tzid)).Messages {
		msgs = append(msgs, [2]string{sms.Service, sms.Code})
	}
	eq(t, msgs, [][2]string{{"other", ""}, {"bank", "777888"}}) // newest first

	ext := get[onlinesim.RentItem](t)(r.Extend(ctx, rent.Tzid, 1))
	eq(t, [2]any{ext.Days, ext.Sum}, [2]any{4, 3.5})
	eq(t, m.Balance(), 6.5)

	fixed := get[onlinesim.RentItem](t)(r.Get(ctx, 7, 1, false))
	eq(t, [2]any{fixed.Extension, fixed.Extend}, [2]any{0, map[string]float64{}})
	_, err := r.Extend(ctx, fixed.Tzid, 1)
	eq(t, code(t, err), "TRY_AGAIN_LATER")
	var ids []int64
	for _, it := range get[[]onlinesim.RentItem](t)(r.State(ctx)) {
		ids = append(ids, it.Tzid)
	}
	eq(t, ids, []int64{rent.Tzid, fixed.Tzid})

	must(t, r.Close(ctx, fixed.Tzid))
	eq(t, len(m.Rents()), 1)
	eq(t, m.Rents()[0].Tzid, rent.Tzid)
	eq(t, m.Balance(), 5.5) // closing a rent gives no refund
	_, err = r.StateOne(ctx, fixed.Tzid)
	eq(t, code(t, err), "ERROR_NO_OPERATIONS")
	eq(t, code(t, r.Close(ctx, fixed.Tzid)), "ERROR_WRONG_TZID")
}

func TestRentErrors(t *testing.T) {
	r := mock.New(mock.WithBalance(1.5)).Client().Rent()
	_, err := r.Get(ctx, 44, 1, true)
	eq(t, code(t, err), "NO_COUNTRY")
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
	sms := get[[]onlinesim.RentItem](t)(r.State(ctx))[0].Messages[0]
	eq(t, [2]string{sms.Code, sms.Service}, [2]string{"6060", "other"})
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
	err := u.SetWebhookURL(ctx, "ftp://x")
	var apiErr *onlinesim.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "VALIDATION_ERROR" || apiErr.Message != "The webhook url must be a valid URL." {
		t.Fatalf("%v", err)
	}
	must(t, u.SetWebhookURL(ctx, "https://example.com/hook"))
	reqs := m.Requests()
	last := reqs[len(reqs)-1]
	eq(t, [3]any{last.Method, last.Path, last.Body}, [3]any{"POST", "profile.php", map[string]any{"profile": map[string]any{"webhook_url": "https://example.com/hook"}}})
	eq(t, get[onlinesim.Profile](t)(u.Profile(ctx)).WebhookURL, "https://example.com/hook")

	tzid := get[int64](t)(c.Numbers().Get(ctx, "google", 44))
	must(t, m.AddSMS(tzid, "code 1234"))
	rent := get[onlinesim.RentItem](t)(c.Rent().Get(ctx, 7, 1, true))
	must(t, m.AddSMS(rent.Tzid, "code 5678"))
	logs := get[onlinesim.Paginator[onlinesim.WebhookLog]](t)(u.WebhookLogs(ctx, 1))
	eq(t, [3]int{logs.Total, logs.CurrentPage, logs.PerPage}, [3]int{2, 1, 10})
	var kinds []string
	for _, l := range logs.Data {
		kinds = append(kinds, string(l.Type)+" "+string(l.Params))
	}
	if len(kinds) != 2 || !strings.HasPrefix(kinds[0], "rent_sms") || !strings.Contains(kinds[0], `"code":"5678"`) ||
		!strings.HasPrefix(kinds[1], "receiving_sms") || !strings.Contains(kinds[1], `"operation_id":1`) {
		t.Fatalf("%v", kinds)
	}

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
}

func TestFreeList(t *testing.T) {
	f := mock.New().Client().Free()
	uk := get[onlinesim.FreeList](t)(f.List(ctx, onlinesim.FreeListOptions{}))
	eq(t, []int{uk.Countries[0].Country, uk.Countries[1].Country}, []int{7, 44})
	eq(t, len(uk.Numbers), 1)
	eq(t, uk.Numbers["7459608320"].FullNumber, "447459608320")
	eq(t, [3]any{uk.Messages.Number, uk.Messages.Country, uk.Messages.Total}, [3]any{"7459608320", 44, 2})
	eq(t, uk.Messages.Data[0].Code, "195895")
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
