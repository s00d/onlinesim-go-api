package mock_test

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
	"github.com/s00d/onlinesim-go-api/v3/mock"
)

var ctx = context.Background()

func noSleep(context.Context, time.Duration) error { return nil }

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func code(t *testing.T, err error) string {
	t.Helper()
	var e *onlinesim.APIError
	if !errors.As(err, &e) {
		t.Fatalf("want *APIError, got %T %v", err, err)
	}
	return e.Code
}

func TestNumberLifecycle(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	m.ScriptSMS(mock.SmsScript{Service: "telegram", Number: "+447000000001", Country: 44, Code: "654321", PollsBeforeCode: 2})
	n := m.Client().Numbers()

	order, err := n.GetWithNumber(ctx, "telegram", 44)
	must(t, err)
	if order.Number != "+447000000001" || order.Country != 44 {
		t.Fatalf("%+v", order)
	}
	list, err := n.State(ctx, onlinesim.StateOptions{})
	must(t, err)
	if len(list) != 1 || list[0].Tzid != order.Tzid {
		t.Fatalf("%+v", list)
	}
	got, err := n.WaitCode(ctx, order.Tzid, onlinesim.WaitCodeOptions{Sleep: noSleep})
	must(t, err)
	if got != "654321" {
		t.Fatalf("code %q", got)
	}
	// Closed by WaitCode: gone from the state.
	list, err = n.State(ctx, onlinesim.StateOptions{})
	must(t, err)
	if len(list) != 0 {
		t.Fatalf("%+v", list)
	}
	_, err = n.StateOne(ctx, order.Tzid, onlinesim.StateOptions{})
	if code(t, err) != "ERROR_NO_OPERATIONS" {
		t.Fatal(err)
	}
}

func TestFullMessageAndKeepOpen(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	n := m.Client().Numbers()
	tzid, err := n.Get(ctx, "telegram", 1)
	must(t, err)
	first, err := n.WaitCode(ctx, tzid, onlinesim.WaitCodeOptions{KeepOpen: true, FullMessage: true, Sleep: noSleep})
	must(t, err)
	if first != "Your code is 123456" {
		t.Fatalf("%q", first)
	}
	must(t, n.Ban(ctx, tzid))
}

func TestNoNumber(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	m.FailNoNumber("telegram")
	_, err := m.Client().Numbers().Get(ctx, "telegram", 1)
	var noNum *onlinesim.NoNumberError
	if !errors.As(err, &noNum) {
		t.Fatalf("want NoNumberError, got %v", err)
	}
}

func TestStrictKeyAndPaths(t *testing.T) {
	m := mock.Start()
	defer m.Close()

	_, err := onlinesim.New("wrong", onlinesim.WithBaseURL(m.BaseURL())).User().Balance(ctx)
	if code(t, err) != "ERROR_WRONG_KEY" {
		t.Fatal(err)
	}
	_, err = onlinesim.New("", onlinesim.WithBaseURL(m.BaseURL())).User().Balance(ctx)
	if code(t, err) != "ERROR_NO_KEY" {
		t.Fatal(err)
	}
	// Key-free endpoints work without a key.
	_, err = onlinesim.New("", onlinesim.WithBaseURL(m.BaseURL())).Numbers().TariffsOne(ctx, 44)
	must(t, err)

	for _, path := range []string{"getBalance", "getFreeList.php"} {
		res, err := http.Get(m.BaseURL() + path)
		must(t, err)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("%s: status %d", path, res.StatusCode)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, m.BaseURL()+"getBalance.php?apikey="+mock.APIKey, nil)
	res, err := http.DefaultClient.Do(req)
	must(t, err)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("apikey in query: status %d", res.StatusCode)
	}
}

func TestUserAndWebhook(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	m.SetBalance(42.5, 1, 3)
	u := m.Client().User()

	b, err := u.Balance(ctx)
	must(t, err)
	if b.Balance != 42.5 || b.Zbalance != 1 || *b.Income != 3 {
		t.Fatalf("%+v", b)
	}
	m.FailBalance("ACCOUNT_BLOCKED")
	_, err = u.Balance(ctx)
	if code(t, err) != "ACCOUNT_BLOCKED" {
		t.Fatal(err)
	}

	must(t, u.SetWebhookURL(ctx, "https://example.com/hook"))
	p, err := u.Profile(ctx)
	must(t, err)
	if p.WebhookURL != "https://example.com/hook" {
		t.Fatalf("%q", p.WebhookURL)
	}
	if code(t, u.SetWebhookURL(ctx, "not a url")) != "VALIDATION_ERROR" {
		t.Fatal("want VALIDATION_ERROR")
	}
	must(t, u.ClearWebhookURL(ctx))
	p, err = u.Profile(ctx)
	must(t, err)
	if p.WebhookURL != "" {
		t.Fatalf("%q", p.WebhookURL)
	}

	h, err := u.PaymentHistory(ctx, 1)
	must(t, err)
	if h.Paylist == nil {
		t.Fatal("paylist nil")
	}
	_, err = u.WebhookLogs(ctx, 1)
	must(t, err)
}

func TestRentLifecycle(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	r := m.Client().Rent()

	rent, err := r.Get(ctx, 7, 1, true)
	must(t, err)
	if rent.Sum != 5 || len(rent.Extend) == 0 {
		t.Fatalf("%+v", rent)
	}
	if !m.AddRentSMS(rent.Tzid, "vkcom", "VK: 1111", "1111") {
		t.Fatal("AddRentSMS")
	}
	one, err := r.StateOne(ctx, rent.Tzid)
	must(t, err)
	if len(one.Messages) != 1 || one.Messages[0].Code != "1111" {
		t.Fatalf("%+v", one.Messages)
	}
	ext, err := r.Extend(ctx, rent.Tzid, 3)
	must(t, err)
	if ext.Days != 4 {
		t.Fatalf("days %d", ext.Days)
	}

	fixed, err := r.Get(ctx, 7, 1, false)
	must(t, err)
	if len(fixed.Extend) != 0 {
		t.Fatalf("%+v", fixed.Extend)
	}
	if code(t, func() error { _, err := r.Extend(ctx, fixed.Tzid, 1); return err }()) != "TRY_AGAIN_LATER" {
		t.Fatal("extend of a non-extendable rent")
	}
	must(t, r.Close(ctx, fixed.Tzid))
	list, err := r.State(ctx)
	must(t, err)
	if len(list) != 1 {
		t.Fatalf("%d rents", len(list))
	}

	_, err = r.TariffsOne(ctx, 380)
	if code(t, err) != "NO_COUNTRY" {
		t.Fatal(err)
	}
}

func TestTariffsAndFree(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	c := m.Client()
	all, err := c.Numbers().Tariffs(ctx)
	must(t, err)
	if all["44"].Services["service_telegram"].Slug != "telegram" {
		t.Fatalf("%+v", all)
	}
	free, err := c.Free().List(ctx, onlinesim.FreeListOptions{})
	must(t, err)
	if free.Numbers["7459608320"].FullNumber != "447459608320" {
		t.Fatalf("%+v", free)
	}
}

func TestPersistState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	m, err := mock.NewBuilder().StatePath(path).Start()
	must(t, err)
	m.SetBalance(77, 2, 1)
	m.Close()

	m2, err := mock.NewBuilder().StatePath(path).Start()
	must(t, err)
	defer m2.Close()
	b, err := m2.Client().User().Balance(ctx)
	must(t, err)
	if b.Balance != 77 {
		t.Fatalf("got %v", b.Balance)
	}
}
