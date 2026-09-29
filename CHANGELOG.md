# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
