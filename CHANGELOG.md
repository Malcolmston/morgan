# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] - 2026-07-18
### Added
- `Skip` ergonomics toward morgan's `skip` option: the `SkipFunc` type plus
  `SkipStatusBelow`, `SkipStatusBetween`, `SkipPaths`, `SkipUserAgents` and
  `CombineSkips` for composing common "only log errors" / drop-health-checks
  policies.
- Request-field helpers mirroring morgan's internal token resolution:
  `ClientIP` (X-Forwarded-For then RemoteAddr), `RequestURL` (path + query),
  `RequestProtocol` (http/https via X-Forwarded-Proto/TLS) and the exported
  `Clfdate` Common Log Format timestamp helper.
- Status classification helpers `StatusCategory` and `StatusColorCode`,
  exposing the response-class reasoning the dev format uses.
- New tokens commonly used in Express morgan formats: `:protocol`, `:host`,
  `:path` and `:query`.

## [1.0.2] - 2026-07-04
### Added
- Automated releases (VERSION-driven tags + GitHub Releases, moving `stable` tag).
- CodeQL, benchmark, dependency-review and stale workflows.
- Expanded coverage tests (dev format, buffered stream, tokens, `Log.String`,
  terminal detection).
### Changed
- CI consolidated into a single matrix workflow (gofmt · vet · `-race` + coverage
  · golangci-lint v2 · govulncheck) on Go 1.23 and 1.24.

## [1.0.1] - 2026-05-01
### Added
- Static GitHub Pages API-documentation site generator (dependency-free
  `go/doc` tool under `docs/gen`).

## [1.0.0]
### Added
- Initial release — morgan HTTP request logger for `net/http`: predefined
  formats (combined, common, dev, short, tiny), a token compiler, and buffered
  streaming.

[Unreleased]: https://github.com/malcolmston/morgan/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/malcolmston/morgan/releases/tag/v1.2.0
[1.0.2]: https://github.com/malcolmston/morgan/releases/tag/v1.0.2
[1.0.1]: https://github.com/malcolmston/morgan/releases/tag/v1.0.1
[1.0.0]: https://github.com/malcolmston/morgan/releases/tag/v1.0.0
