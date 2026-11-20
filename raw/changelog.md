# Changelog

> Release notes of onlinesim-go-api and the v2 → v3 migration.

The full history is in [CHANGELOG.md](https://github.com/s00d/onlinesim-go-api/blob/master/CHANGELOG.md).

## 3.1.1

Aligned with the regenerated backend API spec.

- Decoding fixes: `Profile.Number` (`int64`) and `Profile.Country` (`string`) were swapped; `RentSMS.Code` accepts the integer `0` of a `notify` SMS (as `"0"`); rent `Sum` from a number or a decimal string; `PaymentHistory.Paylist` sent as a JSON array is kept (keyed by index).
- `WaitCode` retries `TRY_AGAIN_LATER` (HTTP 429), `ERROR_UNDEFINED` and `INTERVAL_CONCURRENT_REQUESTS_ERROR` until its timeout.
- Error messages: `"0"`, `ACCOUNT_CLEANED`, `USER_NOT_ACTIVATED`, `ERROR_UNDEFINED`, `FORWARD_NOT_SUPPORTED` added; dead codes removed; `*NoNumberError` only for `NO_NUMBER`.
- The mock follows the backend: rent reservation and confirmation by the first non-`notify` SMS, extend/close rules, `setOperationRevise` answers, missing vs wrong key, webhook-log params, free list and payment history shapes. See [Testing](/testing).

## 3.1.0

- New in-process mock: `mock.New(opts...)` is an `http.RoundTripper` that answers like the real API; `m.Client()` returns a normal client wired to it — no listener, network, key, money or clock. `mock.Start(opts...)` serves the same mock over HTTP. See [Testing](/testing).
- Mock control API aligned with the other OnlineSim SDKs: `AddSMS`, `AddSMSAfter`, `FailNext`, `FailNextTransport`, `SetBalance`, `Balance`, `Operations`, `Rents`, `Requests`, `Reset`. It keeps a real balance (price frozen on order, spent on close after an SMS, refunded without one), derives tariffs from its prices and records webhook deliveries.
- The old mock helpers (`ScriptSMS`, `SmsScript`, `FailNoNumber`, `FailBalance`, `AddRentSMS`, the 3-argument `SetBalance`, `NewBuilder`/`StatePath` persistence) are gone; the mock key is now `mock-key`.

## 3.0.0

Module path `github.com/s00d/onlinesim-go-api/v3`. The API follows the shared OnlineSim SDK contract; v2 code does not compile unchanged.

- The key is sent as `Authorization: Bearer <key>`, never in the URL.
- Typed errors: `*APIError`, `*NoNumberError`, `*TransportError`, `ErrTimeout`; a wrong key (HTTP 456) is an `*APIError`.
- `WaitCode` polls every 5 s for up to 10 min, stops at once on a finished operation, retries network errors, takes an injectable `Sleep`/`Now`.
- No client-side pacing or retries; `INTERVAL_CONCURRENT_REQUESTS_ERROR` reaches the caller.
- New: `Free().List` options, `ParseWebhook`, `AddRentSMS` in the mock; the mock checks paths and the key.
- Removed: `Numbers().Repeat` / `Service` / `ServiceNumber`, `Rent().PortReload`, `User().CreateEmpty`, `Free().Countries` / `Numbers` / `Messages`.

## Migrating from v2

Replace the import path, then:

<table>
<thead>
  <tr>
    <th>
      v2
    </th>
    
    <th>
      v3
    </th>
  </tr>
</thead>

<tbody>
  <tr>
    <td>
      <code>
        github.com/s00d/onlinesim-go-api/v2
      </code>
    </td>
    
    <td>
      <code>
        github.com/s00d/onlinesim-go-api/v3
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Numbers().Get(ctx, GetNumberParams{Service, Country, Reject})
      </code>
    </td>
    
    <td>
      <code>
        Numbers().Get(ctx, service, country, reject...)
      </code>
      
       (<code>
        reject
      </code>
      
       are strings)
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        GetWithNumber(ctx, GetNumberParams{…})
      </code>
      
       → <code>
        NumberWithTz
      </code>
    </td>
    
    <td>
      <code>
        GetWithNumber(ctx, service, country, reject...)
      </code>
      
       → <code>
        NumberOrder
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        State(ctx, StateParams{MessageToCode: IntPtr(0), OrderBy: "ASC", …})
      </code>
    </td>
    
    <td>
      <code>
        State(ctx, StateOptions{FullMessage: true, OrderBy: OrderAsc})
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        StateOne
      </code>
      
       type, <code>
        st.MsgString()
      </code>
    </td>
    
    <td>
      <code>
        Operation
      </code>
      
      , <code>
        op.Msg.Text
      </code>
      
       (or <code>
        op.Msg.List
      </code>
      
       with <code>
        MsgList
      </code>
      
      )
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        WaitCode(ctx, tzid, WaitCodeOptions{Interval, MaxAttempts, NotEnd})
      </code>
      
      , <code>
        DefaultWaitCodeOptions()
      </code>
    </td>
    
    <td>
      <code>
        WaitCode(ctx, tzid, WaitCodeOptions{Interval, Timeout, KeepOpen, Since})
      </code>
      
      ; zero = 5 s / 600 s
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        ErrNoOperations
      </code>
    </td>
    
    <td>
      <code>
        State
      </code>
      
       returns an empty slice; <code>
        StateOne
      </code>
      
       returns <code>
        *APIError
      </code>
      
       <code>
        ERROR_NO_OPERATIONS
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        TariffCountryOne
      </code>
      
      , <code>
        TariffService
      </code>
    </td>
    
    <td>
      <code>
        CountryStat
      </code>
      
      , <code>
        ServiceStat
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Rent().Get(ctx, GetRentParams{Country, Days, Extension: BoolPtr(false)})
      </code>
    </td>
    
    <td>
      <code>
        Rent().Get(ctx, country, days, extension)
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Rent().Extend(ctx, tzid, 0)
      </code>
      
       (defaulted to 1)
    </td>
    
    <td>
      <code>
        Rent().Extend(ctx, tzid, days)
      </code>
      
      ; pass the days
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        RentItem.Sum
      </code>
      
      , <code>
        Extend
      </code>
      
      , <code>
        DayExtend
      </code>
      
       as <code>
        any
      </code>
    </td>
    
    <td>
      <code>
        float64
      </code>
      
      , <code>
        map[string]float64
      </code>
      
      , <code>
        float64
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        User().Profile
      </code>
      
       → <code>
        User
      </code>
    </td>
    
    <td>
      → <code>
        Profile
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        User().PaymentHistory(ctx)
      </code>
      
       → <code>
        PayList
      </code>
    </td>
    
    <td>
      <code>
        User().PaymentHistory(ctx, page)
      </code>
      
       → <code>
        PaymentHistory
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        User().SetWebhookURL(ctx, &url)
      </code>
      
       / <code>
        (ctx, nil)
      </code>
    </td>
    
    <td>
      <code>
        SetWebhookURL(ctx, url)
      </code>
      
       / <code>
        ClearWebhookURL(ctx)
      </code>
      
       (sends <code>
        null
      </code>
      
      )
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        User().WebhookLogs(ctx, int64)
      </code>
      
       → <code>
        WebhookLogsPage
      </code>
    </td>
    
    <td>
      <code>
        WebhookLogs(ctx, int)
      </code>
      
       → <code>
        Paginator[WebhookLog]
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Free().FreeList(ctx)
      </code>
    </td>
    
    <td>
      <code>
        Free().List(ctx, FreeListOptions{})
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        ParseWebhookJSON(b)
      </code>
    </td>
    
    <td>
      <code>
        ParseWebhook(b)
      </code>
      
      ; no <code>
        {"data":…}
      </code>
      
       wrapper, missing fields are zero values, not errors
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        NoNumberError{Code}
      </code>
    </td>
    
    <td>
      <code>
        *NoNumberError
      </code>
      
       embeds <code>
        APIError
      </code>
      
      ; <code>
        errors.As
      </code>
      
       matches both
    </td>
  </tr>
  
  <tr>
    <td>
      HTTP errors as plain <code>
        error
      </code>
    </td>
    
    <td>
      <code>
        *TransportError
      </code>
      
       with <code>
        Status
      </code>
      
      , <code>
        Body
      </code>
      
      , <code>
        Err
      </code>
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        WithOAuth
      </code>
      
      , <code>
        WithUserAgent
      </code>
      
      , <code>
        WithRateLimit
      </code>
    </td>
    
    <td>
      removed (Bearer key; wrap the transport for throttling)
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        WithDevID
      </code>
      
      , <code>
        WithLang
      </code>
      
      , <code>
        WithBaseURL
      </code>
      
      , <code>
        WithHTTPClient
      </code>
      
      , <code>
        WithTimeout
      </code>
    </td>
    
    <td>
      same; the timeout now applies per request, also to an injected client
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        DefaultCountry
      </code>
      
      , country <code>
        0
      </code>
      
       → default
    </td>
    
    <td>
      removed: pass the country (<code>
        7
      </code>
      
      , <code>
        44
      </code>
      
      , …)
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        RequestErrorMessage
      </code>
      
      , <code>
        ErrorFromAPICode
      </code>
      
      , <code>
        APIError.Temporary
      </code>
    </td>
    
    <td>
      removed; <code>
        APIError.Message
      </code>
      
       carries the text
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        mock.SmsScript.Price
      </code>
      
       (string)
    </td>
    
    <td>
      <code>
        float64
      </code>
      
      ; <code>
        mock.APIKey
      </code>
      
      ; <code>
        BaseURL()
      </code>
      
       ends with <code>
        /api/
      </code>
    </td>
  </tr>
</tbody>
</table>
