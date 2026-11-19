package onlinesim_test

import (
	"encoding/json"
	"testing"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func TestBalanceSendsIncomeAndParsesDecimalString(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(fixture(t, "getBalance"))})
	b, err := c.User().Balance(ctx)
	must(t, err)
	income := 9.78
	eq(t, b, onlinesim.Balance{Balance: 1442.33, Zbalance: 1023.41, Income: &income})
	eq(t, f.last().path, "/api/getBalance.php")
	eq(t, f.last().query, map[string]string{"income": "1"})
}

func TestBalanceWithoutIncome(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":"1","balance":"0.000","zbalance":0}`)})
	b, err := c.User().Balance(ctx)
	must(t, err)
	eq(t, b, onlinesim.Balance{})
}

func TestProfile(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(fixture(t, "getProfile"))})
	p, err := c.User().Profile(ctx)
	must(t, err)
	eq(t, f.last().path, "/api/getProfile.php")
	eq(t, f.last().query, map[string]string{})
	eq(t, p.ID, int64(1))
	eq(t, p.WebhookURL, "https://example.com/hook")
	var reject []string
	must(t, json.Unmarshal(p.NumberReject, &reject))
	eq(t, reject, []string{"333"})
	eq(t, p.NumberCountry, 1)
}

func TestPaymentHistoryNormalisesEmptyPaylist(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(fixture(t, "getPaymentHistory"))})
	h, err := c.User().PaymentHistory(ctx, 2)
	must(t, err)
	eq(t, f.last().path, "/api/getPaymentHistory.php")
	eq(t, f.last().query, map[string]string{"page": "2"})
	eq(t, h.Paylist, map[string]json.RawMessage{})
	eq(t, h.Orders.Data[0].ID, int64(9462))
	eq(t, h.Orders.Data[0].Sum, "89.00")
	eq(t, h.Orders.Total, 106)
	eq(t, h.Currency, map[string]float64{"USD": 1})
}

func TestSetWebhookURLPostsJSON(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":"1","user":{"id":2}}`)})
	must(t, c.User().SetWebhookURL(ctx, "https://example.com/hook"))
	call := f.last()
	eq(t, call.method, "POST")
	eq(t, call.path, "/api/profile.php")
	eq(t, call.header.Get("Content-Type"), "application/json")
	eq(t, call.header.Get("Authorization"), "Bearer test-key")
	eq(t, call.body, `{"profile":{"webhook_url":"https://example.com/hook"}}`)
}

func TestClearWebhookURLPostsNull(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":"1","user":{"id":2}}`)})
	must(t, c.User().ClearWebhookURL(ctx))
	eq(t, f.last().method, "POST")
	eq(t, f.last().body, `{"profile":{"webhook_url":null}}`)
}

func TestWebhookLogsReturnsThePaginator(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":"1","data":{"current_page":1,"data":[{"id":7,"user_id":2,"webhook_url":"https://example.com/hook","type":"receiving_sms","params":{"webhook_type":"receiving_sms","operation_id":1520,"code":"195895"},"status":"success","error":null,"created_at":"2026-11-19 10:00:01","updated_at":"2026-11-19 10:00:01"}],"from":1,"to":1,"last_page":1,"per_page":10,"total":1}}`)})
	page, err := c.User().WebhookLogs(ctx, 1)
	must(t, err)
	eq(t, f.last().path, "/api/webhook-logs")
	eq(t, f.last().query, map[string]string{"page": "1"})
	eq(t, page.Total, 1)
	log := page.Data[0]
	eq(t, log.Type, onlinesim.WebhookReceivingSMS)
	eq(t, log.Status, "success")
	hook, err := onlinesim.ParseWebhook(log.Params)
	must(t, err)
	eq(t, hook.OperationID, int64(1520))
	eq(t, hook.Code, "195895")
}
