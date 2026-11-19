---
title: Free numbers
description: Public numbers and their SMS — client.Free(). No API key needed.
navigation:
  order: 4
---

## List

Countries, the numbers of the selected country and the SMS of the selected number, in one call.

```go
func (f *FreeAPI) List(ctx context.Context, opts FreeListOptions) (FreeList, error)
```

| `FreeListOptions` field | Type | Zero value | Description |
|---|---|---|---|
| `Country` | `int` | server: `44` | Country code; an unknown one falls back to the first available. |
| `Number` | `string` | first number | Number **without** the country code, as the keys of `Numbers`. |
| `Page` | `int` | `1` | Page of `Messages`. |
| `Count` | `int` | server default | SMS per page, 10–500. |

Zero fields are not sent.

`FreeList`:

- `Countries`: `[]FreeCountry{Country, CountryText, CountryOriginal}`
- `Numbers`: `map[string]FreeNumber{Country, CountryOriginal, FullNumber, DataHumans, IsArchive}`, keyed by the short number
- `Messages`: a `Paginator[FreeMessage]` (`{ID, Text, InNumber, MyNumber, Code, CreatedAt, DataHumans}`) plus `Number` and `Country`; `Code` is `-` when no code was found

```go
free := client.Free()
list, err := free.List(ctx, onlinesim.FreeListOptions{Country: 44})
if err != nil {
	return err
}
for short := range list.Numbers {
	page, err := free.List(ctx, onlinesim.FreeListOptions{Country: 44, Number: short, Count: 20})
	if err != nil {
		return err
	}
	for _, sms := range page.Messages.Data {
		fmt.Println(sms.InNumber, sms.Code, sms.Text)
	}
	break
}
```
