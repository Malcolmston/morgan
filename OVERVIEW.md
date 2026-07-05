# morgan — Overview

`morgan` is an HTTP request logger middleware for Go's standard `net/http`,
ported from the Node.js [expressjs/morgan](https://github.com/expressjs/morgan)
library. It wraps any `http.Handler` and writes one log line per request in a
format you choose. It depends only on the Go standard library.

---

## How it works

### The middleware

The single entry point is `New`:

```go
func New(next http.Handler, format Format, cfg Config) http.Handler
```

`New` returns an `http.Handler` that you place in front of your real handler.
On each request it:

1. Records a start timestamp.
2. Wraps the `http.ResponseWriter` in an internal `responseRecorder` that
   captures the status code and the moment the response headers are first
   written. This is what lets morgan distinguish `:response-time` (arrival →
   headers written) from `:total-time` (arrival → response fully written).
3. Calls the wrapped handler.
4. Builds a `Log` value from the request, status and timings via `FromRequest`,
   attaches the response headers, and renders a line with the compiled format
   function.
5. Writes the line to the output stream (default `os.Stdout`).

`Config` tunes this flow: `Immediate` logs on request arrival instead of after
the response (useful when the server may crash mid-request, at the cost of
having no status or content-length); `Skip` suppresses lines for selected
requests (for example, only log errors); `Stream` chooses the destination
writer; and `Buffer` enables batched flushing.

### Predefined formats

A `Format` is either the name of a predefined format or a raw format string.
The predefined formats mirror Node morgan:

| Constant           | Shape                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------- |
| `morgan.Combined`  | Apache combined log format (adds `:referrer` and `:user-agent` to common)             |
| `morgan.Common`    | Apache common log format                                                              |
| `morgan.Dev`       | Concise, `:status` colored by range (green 2xx, cyan 3xx, yellow 4xx, red 5xx)        |
| `morgan.Short`     | Shorter than default; includes `:response-time`                                       |
| `morgan.Tiny`      | Minimal: method, url, status, content-length, response-time                           |
| `morgan.JSON`      | One single-line JSON object per request                                               |

`Combined`, `Common`, `Short` and `Tiny` are plain format strings. `Dev` and
`JSON` are registered as format *functions* because they need logic beyond
string substitution (status-based coloring and JSON encoding). Named formats
live in a registry you can extend with `RegisterFormat` (a format string) and
`RegisterFormatFunc` (a `FormatFunc`).

### The token compiler

Format strings are built from tokens written as `:name` or `:name[arg]` — for
example `:method`, `:status`, `:response-time[1]`, `:res[content-length]`,
`:date[iso]`. `Compile` turns a format string into a reusable `FormatFunc`:

```go
type FormatFunc func(r *http.Request, log Log) string
```

Compilation parses the string once into a slice of segments (literal text vs.
token references) using a single regular expression. At render time each token
is looked up in the live token registry and invoked; literals are copied
through. Because lookup happens at render time, tokens registered with `Token`
*after* `Compile` was called are still resolved. Unknown tokens and empty token
values render as `-`.

You register or override a token with `Token`:

```go
type TokenFunc func(r *http.Request, log Log, args ...string) string
```

The built-in tokens (`:method`, `:url`, `:status`, `:remote-addr`, `:date`,
`:req[header]`, `:res[header]`, `:response-time`, `:total-time`, and more) are
registered the same way in the package's `init`.

### Buffered streaming

When `Config.Buffer > 0`, `New` wraps the output writer in an internal buffered
stream. Written lines are appended to an in-memory buffer, and a timer flushes
the accumulated lines to the underlying writer once per interval. This trades a
little latency for far fewer syscalls under high request volume, matching the
`buffer` option of Node morgan. With `Buffer == 0` (the default), each line is
written straight through.

### Terminal detection

`Dev`'s colors are only meaningful on an interactive terminal. When you select
`Dev` but the output stream is *not* a character device — for example when
stdout is redirected to a file or piped to another process — `New` transparently
falls back to an equivalent uncolored format, so logs stay clean. The detection
is done with the standard library alone (`os.File.Stat` plus
`os.ModeCharDevice`), keeping the dependency footprint at zero.

---

## How to use it

### Default dev logging

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

	// Colored, concise output on a terminal; plain output when redirected.
	handler := morgan.New(mux, morgan.Dev, morgan.Config{})
	http.ListenAndServe(":8080", handler)
}
```

Produces lines like:

```
GET / 200 0.224 ms - 5
```

### A custom format string plus a custom token

```go
package main

import (
	"net/http"

	"github.com/malcolmston/morgan"
)

func main() {
	// Register a token that reads a request header.
	morgan.Token("request-id", func(r *http.Request, log morgan.Log, args ...string) string {
		return r.Header.Get("X-Request-ID")
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	})

	// A raw format string that uses the custom token and a bracket argument.
	format := morgan.Format(":method :url :status :response-time[1] ms :request-id")
	handler := morgan.New(mux, format, morgan.Config{})
	http.ListenAndServe(":8080", handler)
}
```

### Buffered output to a file

```go
package main

import (
	"net/http"
	"os"
	"time"

	"github.com/malcolmston/morgan"
)

func main() {
	f, err := os.OpenFile("access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	handler := morgan.New(mux, morgan.Combined, morgan.Config{
		Stream: f,               // write to the file instead of stdout
		Buffer: 2 * time.Second, // flush batched lines every 2s
		Skip: func(r *http.Request, status int) bool {
			return status < 400 // only record errors
		},
	})
	http.ListenAndServe(":8080", handler)
}
```

You can also pre-compile a format for reuse outside the middleware:

```go
render := morgan.Compile(":method :url :status")
line := render(r, log)
```

---

## Why it's better than its predecessor

This is a from-scratch Go port of Node's `expressjs/morgan`, not a wrapper. Its
advantages come from the Go platform rather than from being a strictly larger
feature set — the comparison below is honest about that.

- **Standard library only.** morgan pulls in no third-party packages: no
  `on-headers`, no `on-finished`, no `basic-auth`, no `depd`. Node morgan's
  dependency tree brings several transitive packages into `node_modules`; here
  the import graph is just `net/http` and friends. Fewer moving parts to audit
  or patch.

- **Single binary, no `node_modules`.** Because it compiles into your Go binary,
  there is nothing to install at deploy time, no runtime `node_modules` to ship,
  and no separate Node runtime to keep current. Distribution is one executable.

- **Works with any `net/http` handler.** It is not tied to a framework. Node
  morgan is written for Express/Connect middleware; this port wraps a bare
  `http.Handler`, so it drops in front of `http.ServeMux`, chi, gorilla/mux, or
  a hand-written handler equally.

- **Type-safe configuration and tokens.** Options are a `Config` struct checked
  at compile time rather than a loosely-typed options object; `Format`,
  `FormatFunc` and `TokenFunc` are named types. Passing the wrong shape of
  callback or option is a build error, not a runtime surprise.

- **Concurrency-safe by construction.** The token and format registries are
  guarded by mutexes, and formats are read from the live registry at render
  time, so registering tokens and serving requests from many goroutines is safe.

### Tradeoffs — being honest

- **Not 100% feature-parity.** This port covers the core: predefined formats,
  the token compiler, custom tokens/formats, immediate mode, skip, streaming and
  buffering. Some niche Node-morgan behaviors and the exact ecosystem of
  community token plugins are not reproduced one-for-one.

- **Recompilation to change logging.** Configuration is code. Adjusting formats
  or tokens means editing and rebuilding the Go program, whereas a Node service
  can sometimes be tweaked without a full toolchain. That is the normal
  compiled-vs-interpreted tradeoff.

- **Timing semantics differ slightly.** `:response-time` and `:total-time` are
  measured via a response-writer wrapper rather than Node's event hooks; the
  numbers are close but not derived from the identical mechanism, so exact
  values may differ from the JavaScript original.
