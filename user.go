package onlinesim

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// UserAPI covers balance, profile, payments and webhook settings.
type UserAPI struct{ c *Client }

// Balance is an account balance snapshot.
type Balance struct {
	Balance   float64  `json:"balance"`
	Zbalance  float64  `json:"zbalance"`
	Income    *float64 `json:"income,omitempty"`
	IncomeUSD *float64 `json:"income_usd,omitempty"`
}

// UnmarshalJSON accepts balance fields as numbers or DECIMAL strings from the API.
func (b *Balance) UnmarshalJSON(data []byte) error {
	var raw struct {
		Balance   json.RawMessage `json:"balance"`
		Zbalance  json.RawMessage `json:"zbalance"`
		Income    json.RawMessage `json:"income"`
		IncomeUSD json.RawMessage `json:"income_usd"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var err error
	if b.Balance, err = parseFlexFloat(raw.Balance); err != nil {
		return fmt.Errorf("balance: %w", err)
	}
	if b.Zbalance, err = parseFlexFloat(raw.Zbalance); err != nil {
		return fmt.Errorf("zbalance: %w", err)
	}
	if len(raw.Income) > 0 && string(raw.Income) != "null" {
		v, err := parseFlexFloat(raw.Income)
		if err != nil {
			return fmt.Errorf("income: %w", err)
		}
		b.Income = &v
	}
	if len(raw.IncomeUSD) > 0 && string(raw.IncomeUSD) != "null" {
		v, err := parseFlexFloat(raw.IncomeUSD)
		if err != nil {
			return fmt.Errorf("income_usd: %w", err)
		}
		b.IncomeUSD = &v
	}
	return nil
}

// UserPayment is payment summary inside a profile.
type UserPayment struct {
	Payment float64 `json:"payment"`
	Income  float64 `json:"income"`
	Spent   float64 `json:"spent"`
	Now     float64 `json:"now"`
}

// User is a profile payload.
type User struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	Username      string       `json:"username"`
	Email         string       `json:"email"`
	Apikey        string       `json:"apikey"`
	APIAccess     any          `json:"api_access"`
	Locale        string       `json:"locale"`
	NumberRegion  any          `json:"number_region"`
	NumberCountry any          `json:"number_country"`
	NumberReject  any          `json:"number_reject"`
	WebhookURL    *string      `json:"webhook_url"`
	Payment       *UserPayment `json:"payment"`
	CreatedAt     string       `json:"created_at"`
}

// PayList is payment history / paylist payload.
type PayList map[string]json.RawMessage

// Pay is the result of createEmpty.
type Pay struct {
	PayID  int64          `json:"payId"`
	Params map[string]any `json:"params"`
}

// Balance returns the account balance.
func (a *UserAPI) Balance(ctx context.Context) (Balance, error) {
	var out Balance
	err := a.c.tr.GetJSON(ctx, "getBalance", map[string]string{"income": "1"}, true, &out)
	return out, err
}

// Profile returns the user profile.
func (a *UserAPI) Profile(ctx context.Context) (User, error) {
	var resp struct {
		Response any  `json:"response"`
		Profile  User `json:"profile"`
	}
	err := a.c.tr.GetJSON(ctx, "getProfile", map[string]string{"income": "1"}, true, &resp)
	return resp.Profile, err
}

// PaymentHistory returns payment history / paylist.
func (a *UserAPI) PaymentHistory(ctx context.Context) (PayList, error) {
	out := PayList{}
	err := a.c.tr.GetJSON(ctx, "getPaymentHistory", map[string]string{}, true, &out)
	return out, err
}

// CreateEmpty creates an empty payment with the given params.
//
// Uses path pay/createEmpty without .php (same as Rust). May be unavailable
// on some deployments — treat errors accordingly.
func (a *UserAPI) CreateEmpty(ctx context.Context, params map[string]string) (Pay, error) {
	var out Pay
	err := a.c.tr.GetJSON(ctx, "pay/createEmpty", params, false, &out)
	return out, err
}

// SetWebhookURL enables or changes the profile webhook URL.
func (a *UserAPI) SetWebhookURL(ctx context.Context, url *string) error {
	var webhook any
	if url != nil {
		webhook = *url
	} else {
		webhook = ""
	}
	payload := map[string]any{
		"profile": map[string]any{
			"webhook_url": webhook,
		},
	}
	return a.c.tr.PostRawJSON(ctx, a.c.baseURL, "profile", payload, true, nil)
}

// ClearWebhookURL disables webhooks.
func (a *UserAPI) ClearWebhookURL(ctx context.Context) error {
	return a.SetWebhookURL(ctx, nil)
}

// WebhookLogsPage is a page of webhook delivery logs.
type WebhookLogsPage struct {
	Data        []WebhookLog `json:"data"`
	Total       *int64       `json:"total"`
	PerPage     *int64       `json:"per_page"`
	CurrentPage *int64       `json:"current_page"`
	LastPage    *int64       `json:"last_page"`
}

// WebhookLog is one delivery log row.
type WebhookLog struct {
	ID         int64           `json:"id"`
	Type       string          `json:"type"`
	UserID     *int64          `json:"user_id"`
	WebhookURL string          `json:"webhook_url"`
	Params     json.RawMessage `json:"params"`
	Status     string          `json:"status"`
	Error      string          `json:"error"`
	CreatedAt  string          `json:"created_at"`
}

// WebhookLogs returns paginated webhook delivery logs.
func (a *UserAPI) WebhookLogs(ctx context.Context, page int64) (WebhookLogsPage, error) {
	if page <= 0 {
		page = 1
	}
	var wrap struct {
		Data WebhookLogsPage `json:"data"`
	}
	err := a.c.tr.GetJSON(ctx, "webhook-logs", map[string]string{
		"page": strconv.FormatInt(page, 10),
	}, false, &wrap)
	if err != nil {
		return WebhookLogsPage{}, err
	}
	return wrap.Data, nil
}
