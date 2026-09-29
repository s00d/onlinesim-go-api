package onlinesim_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	onlinesim "github.com/s00d/onlinesim-go-api/v2"
	"github.com/s00d/onlinesim-go-api/v2/mock"
)

func TestBalanceAndProfile(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	m.SetBalance(42.5, 1, 3)

	client := m.Client()
	ctx := context.Background()

	bal, err := client.User().Balance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if bal.Balance != 42.5 {
		t.Fatalf("balance=%v", bal.Balance)
	}

	profile, err := client.User().Profile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != 1 {
		t.Fatalf("profile id=%d", profile.ID)
	}
}

func TestGetNumAndWaitCode(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	m.ScriptSMS(mock.SmsScript{
		Service:         "telegram",
		Number:          "+19001234567",
		Code:            "654321",
		PollsBeforeCode: 1,
	})

	client := m.Client()
	ctx := context.Background()

	ordered, err := client.Numbers().GetWithNumber(ctx, onlinesim.GetNumberParams{Service: "telegram"})
	if err != nil {
		t.Fatal(err)
	}
	if ordered.Number != "+19001234567" {
		t.Fatalf("number=%s", ordered.Number)
	}

	code, err := client.Numbers().WaitCode(ctx, ordered.Tzid, onlinesim.WaitCodeOptions{
		Interval:    0,
		MaxAttempts: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != "654321" {
		t.Fatalf("code=%s", code)
	}
}

func TestNoNumber(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	m.FailNoNumber("telegram")

	_, err := m.Client().Numbers().Get(context.Background(), onlinesim.GetNumberParams{Service: "telegram"})
	var noNum *onlinesim.NoNumberError
	if !errors.As(err, &noNum) {
		t.Fatalf("want NoNumberError, got %v", err)
	}
}

func TestWaitCodeTimeout(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	m.ScriptSMS(mock.SmsScript{
		Service:         "telegram",
		Code:            "1",
		PollsBeforeCode: 100,
	})
	client := m.Client()
	ctx := context.Background()
	tzid, err := client.Numbers().Get(ctx, onlinesim.GetNumberParams{Service: "telegram"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Numbers().WaitCode(ctx, tzid, onlinesim.WaitCodeOptions{Interval: 0, MaxAttempts: 2})
	if !errors.Is(err, onlinesim.ErrTimeout) {
		t.Fatalf("want timeout, got %v", err)
	}
}

func TestWebhookParse(t *testing.T) {
	body := `{"user_id":"1","country_code":1,"number":"+19001234567","sender":"Telegram","message":"code 123456","time_start":"2026-01-01 00:00:00","time_left":10,"operation_id":1000,"webhook_type":"receiving_sms","code":"123456"}`
	p, err := onlinesim.ParseWebhookJSON([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if p.Code != "123456" || p.UserID != 1 {
		t.Fatalf("%+v", p)
	}

	wrapped := `{"data":` + body + `}`
	p2, err := onlinesim.ParseWebhookJSON([]byte(wrapped))
	if err != nil {
		t.Fatal(err)
	}
	if p2.Code != "123456" {
		t.Fatalf("%+v", p2)
	}
}

func TestFreeAndRent(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	client := m.Client()
	ctx := context.Background()

	countries, err := client.Free().Countries(ctx)
	if err != nil || len(countries) == 0 {
		t.Fatalf("countries: %v %v", countries, err)
	}

	item, err := client.Rent().Get(ctx, onlinesim.GetRentParams{Country: 7, Days: 1})
	if err != nil {
		t.Fatal(err)
	}
	if item.Tzid == 0 {
		t.Fatal("empty tzid")
	}
}

func TestBanUsesSetOperationOk(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	m.ScriptSMS(mock.SmsScript{Service: "telegram", Code: "1"})

	client := m.Client()
	ctx := context.Background()
	tzid, err := client.Numbers().Get(ctx, onlinesim.GetNumberParams{Service: "telegram"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Numbers().Ban(ctx, tzid); err != nil {
		t.Fatal(err)
	}
	list, err := client.Numbers().State(ctx, onlinesim.StateParams{MsgList: true, Clean: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range list {
		if st.Tzid == tzid {
			t.Fatalf("banned tzid still in state: %+v", st)
		}
	}
}

func TestTariffDecodeLiveShapes(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want float64
		keys int
	}{
		{
			name: "map numeric",
			raw:  `{"name":"Russia","position":1,"code":7,"new":false,"enabled":true,"services":{"telegram":{"count":1,"popular":true,"price":12.5,"id":1,"service":"telegram","slug":"telegram"}}}`,
			want: 12.5,
			keys: 1,
		},
		{
			name: "string price and empty array",
			raw:  `{"name":"X","position":1,"code":1,"new":false,"enabled":true,"services":[]}`,
			want: 0,
			keys: 0,
		},
		{
			name: "string price in map",
			raw:  `{"name":"Russia","position":1,"code":7,"new":false,"enabled":true,"services":{"telegram":{"count":1,"popular":true,"price":"12.50","id":1,"service":"telegram","slug":"telegram"}}}`,
			want: 12.5,
			keys: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got onlinesim.TariffCountryOne
			if err := json.Unmarshal([]byte(tc.raw), &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Services) != tc.keys {
				t.Fatalf("services=%d want %d", len(got.Services), tc.keys)
			}
			if tc.keys > 0 && got.Services["telegram"].Price != tc.want {
				t.Fatalf("price=%v want %v", got.Services["telegram"].Price, tc.want)
			}
		})
	}
}

func TestSetWebhook(t *testing.T) {
	m := mock.Start()
	defer m.Close()
	client := m.Client()
	ctx := context.Background()
	url := "https://example.com/hook"
	if err := client.User().SetWebhookURL(ctx, &url); err != nil {
		t.Fatal(err)
	}
	profile, err := client.User().Profile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if profile.WebhookURL == nil || *profile.WebhookURL != url {
		t.Fatalf("webhook=%v", profile.WebhookURL)
	}
}
