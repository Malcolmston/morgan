# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-04-30

### Added

- `morgan.JSON` format — built-in structured logging as a single-line JSON object with fields `date`, `method`, `url`, `status`, `responseTime`, `totalTime`, `remoteAddr`, `userAgent`, and `contentLength`
- `:incoming` token — incoming request body size in bytes from `Content-Length`; stored on `Log.INCOMING` and returns `"-"` when the header is absent
- `.golangci.yml` — pins linter configuration for consistent CI lint behaviour across golangci-lint versions

### Fixed

- `Dev` format no longer emits ANSI colour codes when the output stream is not a terminal (e.g. piped to a file or log aggregator)

### Changed

- Added `golang.org/x/term` dependency for TTY detection

---

## [1.0.0] - 2026-04-30

### Added

- `New` HTTP middleware wrapping any `http.Handler`
- Predefined formats: `Combined`, `Common`, `Dev`, `Short`, `Tiny`
- `Dev` format with ANSI status colouring (green 2xx, cyan 3xx, yellow 4xx, red 5xx)
- Token system with `Token()` for registering custom tokens
- Built-in tokens: `:method`, `:url`, `:status`, `:date[clf|iso|web]`, `:remote-addr`, `:remote-user`, `:referrer`, `:user-agent`, `:http-version`, `:pid`, `:req[header]`, `:res[header]`, `:response-time[n]`, `:total-time[n]`
- `Compile()` to pre-parse format strings into reusable `FormatFunc` values
- `RegisterFormat()` and `RegisterFormatFunc()` for named custom formats
- `Config` options: `Immediate`, `Skip`, `Stream`, `Buffer`
- `FromRequest()` to populate a `Log` from a live `*http.Request`
- `X-Forwarded-For` support for proxied remote addresses
- Basic auth username extraction for `:remote-user`
- Write buffering via `Config.Buffer` with timer-based flushing
- CI workflows for build, test (with race detector and coverage), lint, deploy, and publish
