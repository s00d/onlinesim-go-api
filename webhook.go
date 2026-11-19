package onlinesim

import (
	"encoding/json"
	"errors"
)

// WebhookType is the kind of operation a webhook reports.
type WebhookType string

// Webhook types.
const (
	WebhookReceivingSMS WebhookType = "receiving_sms"
	WebhookRentSMS      WebhookType = "rent_sms"
)

// WebhookPayload is an inbound webhook body as normalised by ParseWebhook.
// Missing or malformed fields are zero values ("" or 0).
type WebhookPayload struct {
	// WebhookType is WebhookReceivingSMS, WebhookRentSMS, or "" for anything else.
	WebhookType WebhookType
	// OperationID is the tzid of the operation or rent.
	OperationID int64
	UserID      int64
	CountryCode int
	// Number is the number, e.g. "+79001234567".
	Number string
	// Sender of the SMS, e.g. "VK".
	Sender string
	// Message is the full SMS text.
	Message string
	// Code is the code extracted from the SMS.
	Code      string
	TimeStart string
	TimeLeft  int64
}

// ParseWebhook parses an inbound webhook body (receiving_sms or rent_sms). It is lenient: numeric fields
// may arrive as strings, missing or malformed fields become zero values, unknown fields are ignored.
// It returns an error only when data is not a JSON object.
func ParseWebhook(data []byte) (WebhookPayload, error) {
	var d map[string]json.RawMessage
	if err := json.Unmarshal(data, &d); err != nil {
		return WebhookPayload{}, errors.New("onlinesim: webhook payload is not a JSON object")
	}
	if d == nil { // JSON null
		return WebhookPayload{}, errors.New("onlinesim: webhook payload is not a JSON object")
	}
	p := WebhookPayload{
		OperationID: int64Of(d["operation_id"]),
		UserID:      int64Of(d["user_id"]),
		CountryCode: int(int64Of(d["country_code"])),
		Number:      str(d["number"]),
		Sender:      str(d["sender"]),
		Message:     str(d["message"]),
		Code:        str(d["code"]),
		TimeStart:   str(d["time_start"]),
		TimeLeft:    int64Of(d["time_left"]),
	}
	if t := WebhookType(str(d["webhook_type"])); t == WebhookReceivingSMS || t == WebhookRentSMS {
		p.WebhookType = t
	}
	return p, nil
}

func int64Of(raw json.RawMessage) int64 {
	n, _ := num(raw)
	return int64(n)
}
