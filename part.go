package morgan

import (
	"io"
	"net"
	"net/http"
	"time"
)

// IP represents an IP address as a slice of bytes.
type IP []byte

func (ip IP) String() string {
	return net.IP(ip).String()
}

// Format is a predefined or custom log format string. Tokens use :name or :name[arg] syntax.
type Format string

const (
	// Combined is the standard Apache combined log format.
	//   :remote-addr - :remote-user [:date[clf]] ":method :url HTTP/:http-version" :status :res[content-length] ":referrer" ":user-agent"
	Combined Format = `:remote-addr - :remote-user [:date[clf]] ":method :url HTTP/:http-version" :status :res[content-length] ":referrer" ":user-agent"`

	// Common is the standard Apache common log format.
	//   :remote-addr - :remote-user [:date[clf]] ":method :url HTTP/:http-version" :status :res[content-length]
	Common Format = `:remote-addr - :remote-user [:date[clf]] ":method :url HTTP/:http-version" :status :res[content-length]`

	// Dev is concise output colored by response status, for development use.
	// :status is green for 2xx, cyan for 3xx, yellow for 4xx, red for 5xx, uncolored for 1xx.
	Dev Format = "dev"

	// Short is shorter than default, also including response time.
	//   :remote-addr :remote-user :method :url HTTP/:http-version :status :res[content-length] - :response-time ms
	Short Format = `:remote-addr :remote-user :method :url HTTP/:http-version :status :res[content-length] - :response-time ms`

	// Tiny is the minimal output.
	//   :method :url :status :res[content-length] - :response-time ms
	Tiny Format = `:method :url :status :res[content-length] - :response-time ms`

	// JSON emits each request as a single-line JSON object.
	JSON Format = "json"
)

// Config holds the options for a morgan middleware instance.
type Config struct {
	// Immediate writes the log line on request instead of response.
	// The request will be logged even if the server crashes, but response
	// data (status code, content length, etc.) cannot be logged.
	Immediate bool

	// Skip determines whether to skip logging for a given request/response pair.
	// Example — only log errors:
	//   Skip: func(r *http.Request, status int) bool { return status < 400 }
	Skip func(r *http.Request, status int) bool

	// Stream is the output writer for log lines. Defaults to os.Stdout.
	Stream io.Writer

	// Buffer enables write buffering. Log lines are flushed to Stream in
	// batches at this interval. Zero disables buffering.
	Buffer time.Duration
}

// Log holds all token values captured for a single HTTP request/response cycle.
type Log struct {
	// :date — time the log line is written; use :date[clf], :date[iso], or :date[web].
	DATE time.Time
	// :http-version — HTTP protocol version (e.g. "1.1").
	HTTPVersion string
	// :method — HTTP method of the request.
	METHOD string
	// :pid — process ID of the server process.
	PID int
	// :referrer — Referrer header (falls back to mis-spelled Referer).
	REFERRER string
	// :remote-addr — remote address of the request.
	REMOTE_IP IP
	// :remote-user — user authenticated via Basic auth.
	REMOTE_USER string
	// :req[header] — a specific request header; read live from *http.Request via the token.
	REQUEST_HEADER string
	// :res[header] — response headers captured after the handler runs.
	RESPONSE_HEADERS http.Header
	// :response-time[digits] — time from request arrival to response headers written, in ms.
	RESPONSE_TIME time.Duration
	// :status — HTTP status code; "-" if the cycle completed before a response was sent.
	STATUS int
	// :total-time[digits] — time from request arrival to response fully written, in ms.
	TOTAL_TIME time.Duration
	// :url — request URL path (with query string if present).
	URL string
	// :user-agent — User-Agent header.
	USER_AGENT string
	// :incoming — size of the incoming request body in bytes (-1 if unknown).
	INCOMING int64
}
