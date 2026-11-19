---
title: User
description: Balance, profile, payments and the webhook URL — client.User().
navigation:
  order: 3
---

## Balance

Balance in USD.

```go
func (u *UserAPI) Balance(ctx context.Context) (Balance, error)
```

`Balance`: `{Balance, Zbalance float64; Income *float64}`. `Balance` includes the sum frozen by numbers in progress (`Zbalance`): available = `Balance − Zbalance`. `Income` is the partner income, `nil` when the server sends none. The server caches the answer for 60 s.

```go
b, err := client.User().Balance(ctx)
if err != nil {
	return err
}
available := b.Balance - b.Zbalance
fmt.Println(available)
```

Errors: `*APIError` `ERROR_WRONG_KEY`, `ACCOUNT_BLOCKED`, `API_ACCESS_DISABLED`, `API_ACCESS_IP`.

## Profile

The account profile.

```go
func (u *UserAPI) Profile(ctx context.Context) (Profile, error)
```

`Profile`: `{ID, Name, Username, Email, Locale, Currency, NumberCountry, NumberReject, WebhookURL, APIAccess, …}`. `APIAccess` and `NumberReject` take several shapes and stay `json.RawMessage`.

```go
p, err := client.User().Profile(ctx)
fmt.Println(p.Email, p.WebhookURL, err)
```

## PaymentHistory

Top-up orders (paginated, newest first) and the available payment methods.

```go
func (u *UserAPI) PaymentHistory(ctx context.Context, page int) (PaymentHistory, error)
```

`PaymentHistory`: `{Orders Paginator[PaymentOrder]; Paylist; Domains; Forms; Currency; Buttons}`. `PaymentOrder.Sum` stays a decimal string (`"89.00"`).

```go
h, err := client.User().PaymentHistory(ctx, 1)
if err != nil {
	return err
}
fmt.Printf("%d orders, page %d/%d\n", h.Orders.Total, h.Orders.CurrentPage, h.Orders.LastPage)
```

## SetWebhookURL

Set the URL that receives SMS webhooks (see [Webhooks](/webhooks)).

```go
func (u *UserAPI) SetWebhookURL(ctx context.Context, webhookURL string) error
```

```go
err := client.User().SetWebhookURL(ctx, "https://example.com/onlinesim-hook")
fmt.Println(err)
```

Errors: `*APIError` `VALIDATION_ERROR` for an invalid URL; `Message` holds the server's field messages.

## ClearWebhookURL

Remove the webhook URL, also from operations in progress. Sends `webhook_url: null`.

```go
func (u *UserAPI) ClearWebhookURL(ctx context.Context) error
```

```go
err := client.User().ClearWebhookURL(ctx)
fmt.Println(err)
```

## WebhookLogs

Webhook delivery log, newest first, 10 per page.

```go
func (u *UserAPI) WebhookLogs(ctx context.Context, page int) (Paginator[WebhookLog], error)
```

`WebhookLog`: `{ID, UserID, WebhookURL, Type, Params, Status, Error, CreatedAt, UpdatedAt}` — `Type` is `WebhookReceivingSMS` or `WebhookRentSMS`, `Status` is `success`, `fail` or `pending`, `Params` the payload that was sent (`json.RawMessage`; `ParseWebhook(log.Params)` normalises it).

```go
logs, err := client.User().WebhookLogs(ctx, 1)
if err != nil {
	return err
}
for _, l := range logs.Data {
	fmt.Println(l.CreatedAt, l.Status, l.Error)
}
```
