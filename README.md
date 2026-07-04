# morgan

[![Build](https://github.com/malcolmston/morgan/actions/workflows/build.yml/badge.svg)](https://github.com/malcolmston/morgan/actions/workflows/build.yml)
[![Test](https://github.com/malcolmston/morgan/actions/workflows/test.yml/badge.svg)](https://github.com/malcolmston/morgan/actions/workflows/test.yml)
[![Lint](https://github.com/malcolmston/morgan/actions/workflows/lint.yml/badge.svg)](https://github.com/malcolmston/morgan/actions/workflows/lint.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/malcolmston/morgan.svg)](https://pkg.go.dev/github.com/malcolmston/morgan)

HTTP request logger middleware for Go — a port of the Node.js [morgan](https://github.com/expressjs/morgan) library.

## Installation

```sh
go get github.com/malcolmston/morgan@latest
```

## Quick start

```go
package main

import (
    "net/http"
    "github.com/malcolmston/morgan"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("hello"))
    })

    http.ListenAndServe(":8080", morgan.New(mux, morgan.Dev, morgan.Config{}))
}
```

```
GET / 200 0.224 ms - 5
```

---

## Predefined formats

Pass a format constant as the second argument to `New`.

| Constant | Example output |
|---|---|
| `morgan.Combined` | `::1 - - [27/Nov/2024:06:21:42 +0000] "GET / HTTP/1.1" 200 5 "-" "curl/8.7.1"` |
| `morgan.Common` | `::1 - - [27/Nov/2024:06:21:42 +0000] "GET / HTTP/1.1" 200 5` |
| `morgan.Dev` | `GET / 200 0.224 ms - 5` (status coloured by range) |
| `morgan.Short` | `::1 - GET / HTTP/1.1 200 5 - 0.224 ms` |
| `morgan.Tiny` | `GET / 200 5 - 0.224 ms` |

`Dev` colours the status code: **green** 2xx · **cyan** 3xx · **yellow** 4xx · **red** 5xx.

---

## Config

```go
morgan.New(mux, morgan.Combined, morgan.Config{
    // Write the log line when the request arrives instead of after the response.
    // Response fields (status, content-length) will be empty.
    Immediate: false,

    // Skip logging when this returns true.
    Skip: func(r *http.Request, status int) bool {
        return status < 400 // only log errors
    },

    // Where to write log lines. Defaults to os.Stdout.
    Stream: os.Stderr,

    // Buffer writes and flush on this interval. Zero = no buffering.
    Buffer: time.Second,
})
```

---

## Tokens

Format strings use `:name` or `:name[arg]` syntax. Unknown tokens render as `-`.

| Token | Description |
|---|---|
| `:date[clf]` | `27/Nov/2024:06:21:42 +0000` — Common Log Format (default) |
| `:date[iso]` | `2024-11-27T06:21:42.000Z` — ISO 8601 |
| `:date[web]` | `Wed, 27 Nov 2024 06:21:42 GMT` — RFC 1123 |
| `:http-version` | HTTP protocol version, e.g. `1.1` |
| `:method` | HTTP method |
| `:pid` | Server process ID |
| `:referrer` | `Referer` / `Referrer` request header |
| `:remote-addr` | Client IP (`X-Forwarded-For` takes precedence) |
| `:remote-user` | Basic auth username |
| `:req[header]` | Any named request header |
| `:res[header]` | Any named response header |
| `:response-time[n]` | ms from request arrival to response headers written (default 3 d.p.) |
| `:status` | HTTP status code |
| `:total-time[n]` | ms from request arrival to response fully written (default 3 d.p.) |
| `:url` | Request path and query string |
| `:user-agent` | `User-Agent` header |

### Custom format string

```go
morgan.New(mux, ":method :url :status :response-time[1] ms", morgan.Config{})
// GET /api/users 200 0.4 ms
```

---

## Custom tokens

```go
morgan.Token("request-id", func(r *http.Request, log morgan.Log, args ...string) string {
    return r.Header.Get("X-Request-ID")
})

morgan.New(mux, ":method :url :status :request-id", morgan.Config{})
```

Calling `Token` with an existing name overwrites it.

---

## Custom named formats

Register a format string by name:

```go
morgan.RegisterFormat("short-id", ":method :url :status :request-id")
morgan.New(mux, "short-id", morgan.Config{})
```

Or register a format function for full control:

```go
morgan.RegisterFormatFunc("json", func(r *http.Request, log morgan.Log) string {
    b, _ := json.Marshal(map[string]any{
        "method": log.METHOD,
        "url":    log.URL,
        "status": log.STATUS,
        "ms":     morgan.FormatDuration(log.TOTAL_TIME, 3),
    })
    return string(b)
})

morgan.New(mux, "json", morgan.Config{})
```

---

## Compile

Pre-parse a format string into a reusable `FormatFunc`:

```go
fn := morgan.Compile(":method :url :status")

line := fn(r, log)
```

Tokens registered after `Compile` is called are still resolved at render time.

---

## License

MIT
