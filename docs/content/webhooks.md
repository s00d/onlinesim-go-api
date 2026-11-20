---
title: Webhooks
description: Receive SMS as HTTP callbacks and parse them with ParseWebhook.
navigation:
  order: 5
---

Set a URL with [`client.User().SetWebhookURL`](/user#setwebhookurl). For each SMS the server POSTs a JSON body to it: `webhook_type` `receiving_sms` for numbers, `rent_sms` for rents. Delivery attempts are listed by [`client.User().WebhookLogs`](/user#webhooklogs).

## ParseWebhook

Parse an inbound webhook body.

```go
func ParseWebhook(data []byte) (WebhookPayload, error)
```

Lenient: numeric fields may arrive as strings, missing or malformed fields become zero values (`""`, `0`), unknown fields are ignored. It returns an error only when the input is not a JSON object.

```go
package main

import (
	"io"
	"log"
	"net/http"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func main() {
	http.HandleFunc("/onlinesim-hook", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hook, err := onlinesim.ParseWebhook(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if hook.WebhookType == onlinesim.WebhookReceivingSMS {
			log.Println(hook.OperationID, hook.Code)
		}
		_, _ = io.WriteString(w, "ok")
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Payload fields

| Field | JSON | Type | Description |
|---|---|---|---|
| `WebhookType` | `webhook_type` | `WebhookType` | `WebhookReceivingSMS`, `WebhookRentSMS`, or `""` for anything else. |
| `OperationID` | `operation_id` | `int64` | `tzid` of the operation or rent. |
| `UserID` | `user_id` | `int64` | Account id. |
| `CountryCode` | `country_code` | `int` | Country code of the number. |
| `Number` | `number` | `string` | Numbers: `+` and the full number (`+79001234567`), or `hidden`. Rent: the full number in digits without `+` (`79001234567`). |
| `Sender` | `sender` | `string` | Sender of the SMS, e.g. `VK`. |
| `Message` | `message` | `string` | Full SMS text. |
| `Code` | `code` | `string` | Code extracted from the SMS; `""` when none was found. |
| `TimeStart` | `time_start` | `string` | Numbers: when the operation was ordered. Rent: when the SMS arrived. `YYYY-MM-DD HH:MM:SS`. |
| `TimeLeft` | `time_left` | `int64` | Time left: seconds for numbers, minutes for rent. |
