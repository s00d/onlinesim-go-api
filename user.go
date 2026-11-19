package onlinesim

import (
	"context"
	"encoding/json"
	"net/url"
)

// UserAPI: the account, client.User().
type UserAPI struct {
	do doFunc
}

// Balance is the account balance, USD.
type Balance struct {
	// Balance is the total; it includes Zbalance: available = Balance - Zbalance.
	Balance float64
	// Zbalance is the sum frozen by numbers in progress.
	Zbalance float64
	// Income is the partner income; nil when the server sends none.
	Income *float64
}

// Profile is the account profile.
type Profile struct {
	ID              int64  `json:"id"`
	HashID          string `json:"hash_id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	EmailVerifiedAt string `json:"email_verified_at"`
	APIKey          string `json:"apikey"`
	// APIAccess is true, or a map of allowed IPs.
	APIAccess     json.RawMessage `json:"api_access"`
	Locale        string          `json:"locale"`
	Currency      string          `json:"currency"`
	NumberCountry int             `json:"number_country"`
	// NumberReject is the default reject list: a list or a map of prefixes.
	NumberReject          json.RawMessage `json:"number_reject"`
	WebhookURL            string          `json:"webhook_url"`
	Number                string          `json:"number"`
	Country               int             `json:"country"`
	NumberRegion          int             `json:"number_region"`
	IsAPI                 bool            `json:"is_api"`
	Active                string          `json:"active"`
	Block                 int             `json:"block"`
	Disable               bool            `json:"disable"`
	Verify                int             `json:"verify"`
	UGroup                int             `json:"ugroup"`
	Sub                   int             `json:"sub"`
	FirstLogin            bool            `json:"first_login"`
	IsProjectSubscription bool            `json:"is_project_subscription"`
	ResetCount            int             `json:"resetCount"`
	CreatedAt             string          `json:"created_at"`
	NotificationAt        string          `json:"notification_at"`
}

// Paginator is a Laravel paginator page.
type Paginator[T any] struct {
	CurrentPage  int    `json:"current_page"`
	Data         []T    `json:"data"`
	From         int    `json:"from"`
	To           int    `json:"to"`
	LastPage     int    `json:"last_page"`
	PerPage      int    `json:"per_page"`
	Total        int    `json:"total"`
	FirstPageURL string `json:"first_page_url"`
	LastPageURL  string `json:"last_page_url"`
	NextPageURL  string `json:"next_page_url"`
	PrevPageURL  string `json:"prev_page_url"`
	Path         string `json:"path"`
}

// PaymentOrder is a top-up order.
type PaymentOrder struct {
	ID int64 `json:"id"`
	// Sum stays a decimal string, e.g. "89.00".
	Sum        string `json:"sum"`
	Currency   string `json:"currency"`
	Stat       int    `json:"stat"`
	FormText   string `json:"form_text"`
	StatusText string `json:"status_text"`
	CreatedAt  string `json:"created_at"`
}

// PaymentHistory is the top-up orders and the available payment methods.
type PaymentHistory struct {
	Orders Paginator[PaymentOrder] `json:"orders"`
	// Paylist is the payment methods by list id; empty when none.
	Paylist  map[string]json.RawMessage `json:"paylist"`
	Domains  []string                   `json:"domains"`
	Forms    map[string]string          `json:"forms"`
	Currency map[string]float64         `json:"currency"`
	Buttons  map[string][]float64       `json:"buttons"`
}

// UnmarshalJSON accepts the maps as `[]`.
func (h *PaymentHistory) UnmarshalJSON(b []byte) error {
	type plain PaymentHistory
	var v struct {
		*plain
		Paylist  phpMap[json.RawMessage] `json:"paylist"`
		Forms    phpMap[string]          `json:"forms"`
		Currency phpMap[float64]         `json:"currency"`
		Buttons  phpMap[[]float64]       `json:"buttons"`
	}
	v.plain = (*plain)(h)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	h.Paylist, h.Forms, h.Currency, h.Buttons = v.Paylist, v.Forms, v.Currency, v.Buttons
	if h.Paylist == nil {
		h.Paylist = map[string]json.RawMessage{}
	}
	return nil
}

// WebhookLog is one webhook delivery attempt.
type WebhookLog struct {
	ID         int64       `json:"id"`
	UserID     int64       `json:"user_id"`
	WebhookURL string      `json:"webhook_url"`
	Type       WebhookType `json:"type"`
	// Params is the payload as sent; ParseWebhook(log.Params) normalises it.
	Params json.RawMessage `json:"params"`
	// Status is "success", "fail" or "pending".
	Status    string `json:"status"`
	Error     string `json:"error"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Balance returns the balance in USD.
func (a *UserAPI) Balance(ctx context.Context) (Balance, error) {
	var res struct {
		Balance  json.RawMessage `json:"balance"`
		Zbalance json.RawMessage `json:"zbalance"`
		Income   json.RawMessage `json:"income"`
	}
	if err := a.do(ctx, "getBalance.php", url.Values{"income": {"1"}}, nil, &res); err != nil {
		return Balance{}, err
	}
	out := Balance{}
	out.Balance, _ = num(res.Balance)
	out.Zbalance, _ = num(res.Zbalance)
	if income, ok := num(res.Income); ok {
		out.Income = &income
	}
	return out, nil
}

// Profile returns the account profile.
func (a *UserAPI) Profile(ctx context.Context) (Profile, error) {
	var res struct {
		Profile Profile `json:"profile"`
	}
	err := a.do(ctx, "getProfile.php", nil, nil, &res)
	return res.Profile, err
}

// PaymentHistory returns a page of top-up orders (newest first) and the available payment methods.
func (a *UserAPI) PaymentHistory(ctx context.Context, page int) (PaymentHistory, error) {
	var res PaymentHistory
	err := a.do(ctx, "getPaymentHistory.php", url.Values{"page": {itoa(page)}}, nil, &res)
	return res, err
}

// SetWebhookURL sets the URL that receives SMS webhooks. An invalid URL returns *APIError VALIDATION_ERROR.
func (a *UserAPI) SetWebhookURL(ctx context.Context, webhookURL string) error {
	body := map[string]any{"profile": map[string]any{"webhook_url": webhookURL}}
	return a.do(ctx, "profile.php", nil, body, nil)
}

// ClearWebhookURL removes the webhook URL (also from operations in progress); it sends webhook_url: null.
func (a *UserAPI) ClearWebhookURL(ctx context.Context) error {
	body := map[string]any{"profile": map[string]any{"webhook_url": nil}}
	return a.do(ctx, "profile.php", nil, body, nil)
}

// WebhookLogs returns a page of the webhook delivery log, newest first, 10 per page.
func (a *UserAPI) WebhookLogs(ctx context.Context, page int) (Paginator[WebhookLog], error) {
	var res struct {
		Data Paginator[WebhookLog] `json:"data"`
	}
	err := a.do(ctx, "webhook-logs", url.Values{"page": {itoa(page)}}, nil, &res)
	return res.Data, err
}
