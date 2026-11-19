package onlinesim_test

import (
	"testing"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func TestParseWebhookReceivingSMS(t *testing.T) {
	p, err := onlinesim.ParseWebhook([]byte(`{"webhook_type":"receiving_sms","operation_id":1520,"user_id":2,"country_code":7,"number":"+79001234567","sender":"VK","message":"VK: 195895 is your code","code":"195895","time_start":"2026-11-19 10:00:00","time_left":1100}`))
	must(t, err)
	eq(t, p, onlinesim.WebhookPayload{
		WebhookType: onlinesim.WebhookReceivingSMS, OperationID: 1520, UserID: 2, CountryCode: 7,
		Number: "+79001234567", Sender: "VK", Message: "VK: 195895 is your code", Code: "195895",
		TimeStart: "2026-11-19 10:00:00", TimeLeft: 1100,
	})
}

func TestParseWebhookIsLenient(t *testing.T) {
	p, err := onlinesim.ParseWebhook([]byte(`{"webhook_type":"rent_sms","operation_id":"430","user_id":null,"code":2991,"time_left":"soon","country_code":[1],"extra":true}`))
	must(t, err)
	eq(t, p, onlinesim.WebhookPayload{WebhookType: onlinesim.WebhookRentSMS, OperationID: 430, Code: "2991"})

	p, err = onlinesim.ParseWebhook([]byte(`{"webhook_type":"other"}`))
	must(t, err)
	eq(t, p.WebhookType, onlinesim.WebhookType(""))
}

func TestParseWebhookRejectsOnlyNonObjects(t *testing.T) {
	for _, body := range []string{"not json", "[1,2]", "null", ""} {
		if _, err := onlinesim.ParseWebhook([]byte(body)); err == nil {
			t.Fatalf("%q: want error", body)
		}
	}
}
