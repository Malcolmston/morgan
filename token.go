package morgan

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// TokenFunc is the signature for a token handler. It receives the original request,
// the populated Log, and any bracket arguments (e.g. "clf" from :date[clf]).
type TokenFunc func(r *http.Request, log Log, args ...string) string

var (
	tokenMu  sync.RWMutex
	tokenMap = map[string]TokenFunc{}
)

// Token registers (or overwrites) a named token available as :name or :name[arg] in
// format strings. The callback should return the string value to render at that position.
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

var tokenRe = regexp.MustCompile(`:([a-zA-Z][a-zA-Z0-9_-]*)(?:\[([^\]]*)\])?`)

// Render substitutes all :token and :token[arg] references in format with their resolved
// values. Unknown tokens are left as-is.
func Render(format Format, r *http.Request, log Log) string {
	tokenMu.RLock()
	defer tokenMu.RUnlock()
	return tokenRe.ReplaceAllStringFunc(string(format), func(match string) string {
		sub := tokenRe.FindStringSubmatch(match)
		name, arg := sub[1], sub[2]
		fn, ok := tokenMap[name]
		if !ok {
			return match
		}
		if arg != "" {
			return fn(r, log, arg)
		}
		return fn(r, log)
	})
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
			return "-"
		}
		return fmt.Sprintf("%d", log.STATUS)
	})

	Token("referrer", func(_ *http.Request, log Log, _ ...string) string {
		return orDash(log.REFERRER)
	})

	Token("remote-addr", func(_ *http.Request, log Log, _ ...string) string {
		return log.REMOTE_IP.String()
	})

	Token("remote-user", func(_ *http.Request, log Log, _ ...string) string {
		return orDash(log.REMOTE_USER)
	})

	Token("user-agent", func(_ *http.Request, log Log, _ ...string) string {
		return orDash(log.USER_AGENT)
	})

	Token("pid", func(_ *http.Request, log Log, _ ...string) string {
		return fmt.Sprintf("%d", log.PID)
	})

	Token("response-time", func(_ *http.Request, log Log, args ...string) string {
		digits := 3
		if len(args) > 0 {
			if d, err := strconv.Atoi(args[0]); err == nil {
				digits = d
			}
		}
		return FormatDuration(log.RESPONSE_TIME, digits)
	})

	Token("total-time", func(_ *http.Request, log Log, args ...string) string {
		digits := 3
		if len(args) > 0 {
			if d, err := strconv.Atoi(args[0]); err == nil {
				digits = d
			}
		}
		return FormatDuration(log.TOTAL_TIME, digits)
	})

	// :date[format] — current date/time when the log line is written.
	// Available formats:
	//   clf  — "10/Oct/2000:13:55:36 +0000"  (Common Log Format, default)
	//   iso  — "2000-10-10T13:55:36.000Z"    (ISO 8601 / RFC 3339)
	//   web  — "Tue, 10 Oct 2000 13:55:36 GMT" (RFC 1123)
	// If no format is given, web is used.
	Token("date", func(_ *http.Request, log Log, args ...string) string {
		t := log.DATE
		if t.IsZero() {
			t = time.Now()
		}
		if len(args) == 0 {
			return t.UTC().Format(time.RFC1123)
		}
		switch args[0] {
		case "clf":
			return t.Format("02/Jan/2006:15:04:05 -0700")
		case "iso":
			return t.UTC().Format(time.RFC3339Nano)
		case "web":
			return t.UTC().Format(time.RFC1123)
		default:
			return t.UTC().Format(time.RFC1123)
		}
	})

	// :req[header] — reads a named header from the original request.
	Token("req", func(r *http.Request, _ Log, args ...string) string {
		if len(args) == 0 {
			return "-"
		}
		return orDash(r.Header.Get(args[0]))
	})

	// :res[header] — reads a named header from the response.
	Token("res", func(_ *http.Request, log Log, args ...string) string {
		if len(args) == 0 || log.RESPONSE_HEADERS == nil {
			return "-"
		}
		return orDash(log.RESPONSE_HEADERS.Get(args[0]))
	})
}
