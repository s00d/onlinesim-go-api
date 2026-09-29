package onlinesim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// WebhookType is the service that produced the webhook.
type WebhookType string

const (
	WebhookReceivingSMS WebhookType = "receiving_sms"
	WebhookRentSMS      WebhookType = "rent_sms"
)

// WebhookPayload is the JSON body OnlineSim POSTs to your webhook_url.
type WebhookPayload struct {
	UserID      int64       `json:"user_id"`
	CountryCode int64       `json:"country_code"`
	Number      string      `json:"number"`
	Sender      string      `json:"sender"`
	Message     string      `json:"message"`
	TimeStart   string      `json:"time_start"`
	TimeLeft    int64       `json:"time_left"`
	OperationID int64       `json:"operation_id"`
	WebhookType WebhookType `json:"webhook_type"`
	Code        string      `json:"code"`
}

// ParseWebhookJSON parses an inbound webhook body.
// Accepts a flat payload or { "data": { ... } }.
func ParseWebhookJSON(data []byte) (WebhookPayload, error) {
	data = bytes.TrimSpace(data)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return WebhookPayload{}, fmt.Errorf("invalid webhook payload: %w", err)
	}
	if inner, ok := raw["data"]; ok {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(inner, &probe); err == nil && probe != nil {
			raw = probe
		}
	}
	var p WebhookPayload
	var err error
	if p.UserID, err = flexInt64(raw["user_id"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("user_id: %w", err)
	}
	if p.CountryCode, err = flexInt64(raw["country_code"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("country_code: %w", err)
	}
	if p.Number, err = flexString(raw["number"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("number: %w", err)
	}
	if p.Sender, err = flexString(raw["sender"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("sender: %w", err)
	}
	if p.Message, err = flexString(raw["message"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("message: %w", err)
	}
	if p.TimeStart, err = flexString(raw["time_start"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("time_start: %w", err)
	}
	if p.TimeLeft, err = flexInt64(raw["time_left"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("time_left: %w", err)
	}
	if p.OperationID, err = flexInt64(raw["operation_id"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("operation_id: %w", err)
	}
	wt, err := flexString(raw["webhook_type"])
	if err != nil {
		return WebhookPayload{}, fmt.Errorf("webhook_type: %w", err)
	}
	p.WebhookType = WebhookType(wt)
	if p.Code, err = flexString(raw["code"]); err != nil {
		return WebhookPayload{}, fmt.Errorf("code: %w", err)
	}
	return p, nil
}

func flexString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.String(), nil
	}
	return "", fmt.Errorf("expected string or number, got %s", string(raw))
}

func flexInt64(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, fmt.Errorf("missing")
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strconv.ParseInt(s, 10, 64)
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int64(f), nil
	}
	return 0, fmt.Errorf("expected int, got %s", string(raw))
}
