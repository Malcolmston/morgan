package morgan

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// TokenFunc is the callback signature for a token handler. It receives the original
// request, the populated Log, and any bracket arguments (e.g. "clf" from :date[clf]),
// and returns the string to render at that position in the log line.
type TokenFunc func(r *http.Request, log Log, args ...string) string

var (
	tokenMu  sync.RWMutex
	tokenMap = map[string]TokenFunc{}
)

// Token registers (or overwrites) a named token available as :name or :name[arg]
// in format strings.
//
// Example:
//
//	morgan.Token("type", func(r *http.Request, log morgan.Log, args ...string) string {
//	    return r.Header.Get("Content-Type")
//	})
func Token(name string, fn TokenFunc) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	tokenMap[name] = fn
}

func init() {
	Token("http-version", func(_ *http.Request, log Log, _ ...string) string {
		return log.HTTPVersion
	})

	Token("method", func(_ *http.Request, log Log, _ ...string) string {
		return log.METHOD
	})

	Token("url", func(_ *http.Request, log Log, _ ...string) string {
		return log.URL
	})

	Token("status", func(_ *http.Request, log Log, _ ...string) string {
		if log.STATUS == 0 {
			return ""
		}
		return fmt.Sprintf("%d", log.STATUS)
	})

	Token("referrer", func(_ *http.Request, log Log, _ ...string) string {
		return log.REFERRER
	})

	Token("remote-addr", func(_ *http.Request, log Log, _ ...string) string {
		return log.REMOTE_IP.String()
	})

	Token("remote-user", func(_ *http.Request, log Log, _ ...string) string {
		return log.REMOTE_USER
	})

	Token("user-agent", func(_ *http.Request, log Log, _ ...string) string {
		return log.USER_AGENT
	})

	Token("pid", func(_ *http.Request, log Log, _ ...string) string {
		return fmt.Sprintf("%d", log.PID)
	})

	// :response-time[digits] — ms from request arrival to response headers written.
	Token("response-time", func(_ *http.Request, log Log, args ...string) string {
		digits := 3
		if len(args) > 0 {
			if d, err := strconv.Atoi(args[0]); err == nil {
				digits = d
			}
		}
		return FormatDuration(log.RESPONSE_TIME, digits)
	})

	// :total-time[digits] — ms from request arrival to response fully written.
	Token("total-time", func(_ *http.Request, log Log, args ...string) string {
		digits := 3
		if len(args) > 0 {
			if d, err := strconv.Atoi(args[0]); err == nil {
				digits = d
			}
		}
		return FormatDuration(log.TOTAL_TIME, digits)
	})

	// :date[format] — time the log line is written.
	//   clf  — "27/Nov/2024:06:21:42 +0000"   (Common Log Format)
	//   iso  — "2024-11-27T06:21:42.000Z"      (ISO 8601 / RFC 3339)
	//   web  — "Wed, 27 Nov 2024 06:21:42 GMT" (RFC 1123, default)
	Token("date", func(_ *http.Request, log Log, args ...string) string {
		t := log.DATE
		if t.IsZero() {
			t = time.Now()
		}
		format := "web"
		if len(args) > 0 {
			format = args[0]
		}
		switch format {
		case "clf":
			return clfdate(t)
		case "iso":
			return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		default: // "web"
			return t.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
		}
	})

	// :req[header] — reads a named header from the original request.
	Token("req", func(r *http.Request, _ Log, args ...string) string {
		if r == nil || len(args) == 0 {
			return ""
		}
		vals := r.Header[http.CanonicalHeaderKey(args[0])]
		if len(vals) == 0 {
			return ""
		}
		// join multiple values, matching JS Array.join(', ')
		result := vals[0]
		for _, v := range vals[1:] {
			result += ", " + v
		}
		return result
	})

	// :res[header] — reads a named header from the response.
	Token("res", func(_ *http.Request, log Log, args ...string) string {
		if len(args) == 0 || log.RESPONSE_HEADERS == nil {
			return ""
		}
		vals := log.RESPONSE_HEADERS[http.CanonicalHeaderKey(args[0])]
		if len(vals) == 0 {
			return ""
		}
		result := vals[0]
		for _, v := range vals[1:] {
			result += ", " + v
		}
		return result
	})
}
