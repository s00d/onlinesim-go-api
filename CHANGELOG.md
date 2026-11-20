# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.1.1] - 2026-11-20

Aligned with the regenerated backend API spec.

### Fixed

- Decoding: `Profile.Number` is `int64` and `Profile.Country` a `string` (they were swapped and failed on real answers),
  decoded leniently (null → zero value); `RentSMS.Code` accepts the integer `0` of a `notify` SMS (normalised to `"0"`)
  instead of failing the whole rent read; rent `sum` accepted as a number or a decimal string.
- `User().PaymentHistory` no longer drops `Paylist` when the server sends it as a JSON array (ids 0, 1, 2…): every map that
  arrives as a JSON list is kept, keyed by index (`"0"`, `"1"`, …).
- `Numbers().WaitCode` retries `*APIError` `TRY_AGAIN_LATER` (HTTP 429 rate limit), `ERROR_UNDEFINED` and
  `INTERVAL_CONCURRENT_REQUESTS_ERROR` until its timeout instead of returning at once.
- Error messages: `"0"` (ban not applied, the operation already has an SMS), `ACCOUNT_CLEANED`, `USER_NOT_ACTIVATED`,
  `ERROR_UNDEFINED`, `FORWARD_NOT_SUPPORTED` added; `ERROR_NO_KEY`, `NO_NUMBER_FOR_FORWARD`, `ACCOUNT_IDENTIFICATION_REQUIRED`,
  `API_DISABLED` (no longer sent) removed; `*NoNumberError` only for `NO_NUMBER`; `ERROR_WRONG_KEY` reads "Wrong or missing API key".
- The mock follows the backend: `setOperationRevise` answers `{"tzid":"N","response":"1"}` and `ERROR_WRONG_TZID` /
  `ERROR_NO_TZID`; `setOperationOk` tzid `0` → `ERROR_NO_TZID`; a missing key is HTTP 200 `ERROR_WRONG_KEY` (456 only for a
  wrong key, also on public endpoints); `getNum` with `number` adds `title`, `response_text`, `guard_interval_remaining_seconds`.
  Rent: `getRentNum` reserves the price (`status` 0) and the first non-`notify` SMS confirms and charges it, unknown country →
  `UNDEFINED_COUNTRY`, `sum` a number in getRentNum/extendRentState and a string in getRentState,
  `guard_interval_remaining_seconds` with `number`; `extendRentState` refusals are all `TRY_AGAIN_LATER`; `closeRentNum`
  unknown tzid → `ERROR_NO_OPERATIONS`, releases the reservation of an unconfirmed rent. Webhook-log params: `code` `""` without
  a code, rent `number` in full digits, rent `time_left` in minutes, `time_start` per type. `profile.php` accepts any
  `scheme://…` URL. `getPaymentHistory` adds `forms`/`buttons`/`domains`; `getFreeList` lists 44 first, clamps `count` to
  10..500, sends ISO-8601 `created_at` and `ignore`/`content`.
- Docs: rent reservation and `Sum`, available = `Balance − max(Zbalance, 0)`, `Ban` answer `"0"`, webhook
  `Number`/`TimeLeft`/`TimeStart`, the `WaitCode` retry policy, the mock's behaviour.

## [3.1.0] - 2026-11-19

### Added

- In-process mock: `mock.New(opts...)` returns a `*mock.Mock`, an `http.RoundTripper` that answers every endpoint the SDK calls
  like the real backend (HTTP 456 for a wrong key, string/int `response`, decimal-string money, `[]` for empty maps) and keeps
  balance, operations and rents — no listener, network, key, money or clock. `m.Client(opts...)` returns a normal
  `*onlinesim.Client` wired to it (`WithHTTPClient(&http.Client{Transport: m})` for a hand-built client). Options:
  `WithAPIKey`, `WithBalance`, `WithPrices`, `WithRentTariffs`, `WithGuardInterval`.
- Mock control API shared with the other OnlineSim SDKs: `AddSMS`, `AddSMSAfter` (lets `WaitCode` run with a no-op `Sleep`),
  `FailNext`, `FailNextTransport` (`OnEndpoint`, `WithStatus`), `SetBalance`, `Balance`, `Operations`, `Rents`, `Requests`, `Reset`.
  Deterministic: counter-based ids and numbers, fixed timestamps. Docs page: https://s00d.github.io/onlinesim-go-api/testing
- CI: `staticcheck`, `govulncheck`, `go mod tidy -diff`, `go mod verify`, and a smoke test that runs the README "Testing" example
  in an empty module against a `git archive` copy of the repository.
- `.github/workflows/release.yml`: on a `vX.Y.Z` tag runs CI, checks the tag against CHANGELOG, creates the GitHub release and
  warms the Go module proxy.

### Changed

- `mock.Start(opts...)` serves the same mock over HTTP (`*mock.Server` embeds `*mock.Mock`; `BaseURL`, `Client`, `Close` as before).
  The mock follows the shared semantics: a real balance (price frozen on order, spent on close after an SMS, refunded without one),
  numbers `+<country>9000000NN`, closed operations reported as `TZ_OVER_OK`, a wrong or missing key → 456 `ERROR_WRONG_KEY`.
- The default mock key is `mock-key` (`mock.DefaultAPIKey`; `mock.APIKey` is a deprecated alias).

### Removed

- Mock helpers replaced by the shared control API: `ScriptSMS`/`SmsScript`/`DefaultSmsScript` (use `AddSMS`/`AddSMSAfter`),
  `FailNoNumber` (`FailNext("NO_NUMBER", OnEndpoint("getNum.php"))`), `FailBalance` (`FailNext(code, OnEndpoint("getBalance.php"))`),
  `AddRentSMS` (`AddSMS`), `SetBalance(balance, zbalance, income)` (`SetBalance(usd)`), and the state persistence
  (`NewBuilder`, `StatePath`, `StatePathFromEnv`, `StatePathEnv`, `ResetState`).

## [3.0.0] - 2026-11-19

Aligned with the shared OnlineSim SDK contract. Module path is now `github.com/s00d/onlinesim-go-api/v3`;
the step-by-step v2 → v3 table is in the docs: https://s00d.github.io/onlinesim-go-api/changelog#migrating-from-v2

### Removed

- `Numbers().Repeat` (getNumRepeat), `Numbers().Service` (getService), `Numbers().ServiceNumber` (getServiceNumber): no backend route.
- `Rent().PortReload`: deprecated no-op on the backend.
- `User().CreateEmpty` (pay/createEmpty).
- `Free().Countries`, `Free().Numbers`, `Free().Messages`: the backend returns empty data for them; use `Free().List`.
- `WithOAuth`, `WithUserAgent`, `WithRateLimit`, `DefaultCountry`, `Client.BaseURL()`, `IntPtr`, `BoolPtr`, `DefaultWaitCodeOptions`,
  `ErrNoOperations`, `RequestErrorMessage`, `ErrorFromAPICode`, `APIError.Temporary`, `PayList`, `Pay`, `Balance.IncomeUSD`.
- Client-side pacing (1 rps, 5 s per `setOperationOk` path) and the automatic retry of `INTERVAL_CONCURRENT_REQUESTS_ERROR`.
- `internal/transport`; the GitHub Wiki links (replaced by the docs site).

### Changed

- The API key is sent as `Authorization: Bearer <key>`, never as the `apikey` query parameter.
- Methods take plain arguments instead of param structs: `Numbers().Get(ctx, service, country, reject...)`,
  `GetWithNumber` (returns `NumberOrder`), `Rent().Get(ctx, country, days, extension)`, `Rent().Extend(ctx, tzid, days)`,
  `User().PaymentHistory(ctx, page)`, `User().WebhookLogs(ctx, page)`, `User().SetWebhookURL(ctx, url)`, `Free().List(ctx, FreeListOptions)`.
  No zero-value substitution: the country is always sent as given.
- `StateParams` → `StateOptions{FullMessage, OrderBy, MsgList}` where every zero field is the default; `orderby` is sent lowercase;
  the result type `StateOne` → `Operation` with `Msg OperationMsg{Text, List}`; `State` returns an empty slice on `ERROR_NO_OPERATIONS`,
  `StateOne` returns `*APIError` `ERROR_NO_OPERATIONS`. Both send `number=1` (guard interval).
- `WaitCodeOptions{Interval, Timeout, KeepOpen, FullMessage, Since, Sleep, Now}` (was `Interval, MaxAttempts, NotEnd, FullMessage`).
- Errors: `*APIError{Code, Message}`, `*NoNumberError` (embeds `APIError`), `*TransportError{Status, Body, Err}`, `ErrTimeout`.
- Typed results: `CountryStat`/`ServiceStat` (were `TariffCountryOne`/`TariffService` with `any` fields), `RentItem` (`Sum float64`,
  `Extend map[string]float64`), `RentTariff`, `Profile` (was `User`), `PaymentHistory`, `Paginator[T]`, `FreeList`, `WebhookLog`.
- `ParseWebhookJSON` → `ParseWebhook`; the `{"data": …}` wrapper is no longer accepted.
- `WithTimeout` is a per-request timeout that applies to an injected `*http.Client` too.
- Rent reads send `pagination=0` and `number=1`; `Rent().Get` always sends `extension` as `1`/`0`. Booleans are `1`/`0` everywhere.
- Mock: strict paths and Bearer key, base URL `…/api/`, `SmsScript.Price` is `float64`, real-API answers (`ERROR_NO_OPERATIONS`, `[]` rent tariffs, validation errors).
- Go 1.22 floor; CI on Go 1.22 and stable with `gofmt -l`, `go vet`, golangci-lint v2, `go test -race`.

### Fixed

- `WaitCode` ignored terminal states (`TZ_OVER_OK`, `ERROR_NO_ITEMS`) and gave up after 10 polls by default; it now uses a 600 s timeout
  and returns `*APIError` at once for a finished operation.
- A non-2xx response lost the API code: the body is now checked before the status (HTTP 456 `ERROR_WRONG_KEY` → `*APIError`).
- Non-JSON responses are `*TransportError` instead of a decode error or silent success.
- `ParseWebhook` no longer fails on a missing or malformed numeric field.
- `ClearWebhookURL` sent `""`; it now sends JSON `null`.
- `getNum` sent `number=true` and `extension`; `number` is a presence flag sent only by `GetWithNumber`.
- `reject` is a comma list of strings (number prefixes), not ints.
- Empty PHP maps `[]` (country `services`, rent `extend`, `paylist`, free `numbers`) decode as empty maps.
- Tests no longer depend on wall-clock timing.

### Added

- `Free().List(ctx, FreeListOptions{Country, Number, Page, Count})`.
- `WaitCodeOptions.Since`, injectable `Sleep` / `Now`; retries transient transport errors until the timeout.
- `VALIDATION_ERROR` mapping for profile validation errors.
- `mock.APIKey`, `Server.AddRentSMS`.
- Offline test suite per method, opt-in integration test (`ONLINESIM_IT_BASE`, `ONLINESIM_IT_KEY`), a test that compiles the docs examples.
- Documentation site (Nuxt Content): https://s00d.github.io/onlinesim-go-api/

## [2.0.2] - 2026-09-29

### Changed

- `DefaultCountry` is **`1` (USA)**. Country `7` (RU) is unavailable/banned for typical API keys.

## [2.0.1] - 2026-09-29

### Fixed

- Live API decode: balance/income as DECIMAL strings, `ERROR_NO_OPERATIONS` → empty state, rent `TariffsOne` empty `[]`, `getPrice` null.
- `getNum` accepts tariff keys with `service_` prefix (strips to slug).
- Client pacing matches OpenAPI: default **1 rps**, `setOperationOk` ≥ **5s**, auto-retry on `INTERVAL_CONCURRENT_REQUESTS_ERROR`.

## [2.0.0] - 2026-09-29

### Breaking

- Module path is now `github.com/s00d/onlinesim-go-api/v2`.
- All methods use idiomatic `(T, error)` and take `context.Context` as the first argument.
- `NewClient` replaced by `New(apiKey string, opts ...Option)`.
- Removed `Get*` type prefixes (`NumbersAPI`, `FreeAPI`, …).
- Removed OnlineSim `Proxy` and `OnlineProxy` APIs (aligned with Rust SDK; OnlineSim proxy purchase is `API_DISABLED`).
- Removed dependency on `github.com/ddliu/go-httpclient`; HTTP is stdlib only.
- Default API base URL is `https://onlinesim.host/api` (no insecure TLS skip).
- Network errors are returned instead of panicking.
- `GetRentParams.Extension` is `*bool` (nil → `true`, matching Laravel/Rust).

### Added

- Functional options: lang, dev_id, oauth, base URL, HTTP client, rate limit, user-agent, timeout.
- Numbers: `GetWithNumber`, `Ban` (`setOperationOk?ban=1`), `Repeat`, `WaitCode`, richer state params.
- Free: `FreeList`; messages use `getFreeMessageList`.
- User: `PaymentHistory`, `CreateEmpty` (`pay/createEmpty`), webhook URL + logs.
- `ParseWebhookJSON` for inbound SMS webhooks.
- Flexible tariff decode (string prices, `services: []`).
- Public `mock` package with optional state persistence.
- Examples: `balance`, `wait_sms`, `mock_sms_flow`.
- GitHub Actions CI (test, vet, lint, build).

### Changed

- Go toolchain requirement: 1.22+.

## [1.0.0] - 2019-08-11

### Added

- Initial project.
