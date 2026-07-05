// Package morgan is an HTTP request logger middleware for net/http, ported from
// the Node.js expressjs/morgan library (the request logger most commonly seen
// mounted on Express applications). Like its namesake, morgan produces one
// access-log line per HTTP request and derives the fields of that line by
// expanding named tokens — :method, :url, :status, :response-time and so on —
// against the request and the response it observed. The goal of the port is
// behavioural parity: the same format names, the same token syntax, and the
// same Common Log Format output, expressed idiomatically in Go and depending
// only on the standard library.
//
// Use morgan when you want structured, human- or machine-readable access logs
// for an HTTP service without adopting a heavier logging framework. It answers
// the operational questions an access log exists to answer — who called what,
// when, with which method and path, what status came back, how large the
// response was, and how long it took — and it does so uniformly for every
// handler it wraps. Typical uses are request auditing, latency spot-checks,
// feeding an existing log pipeline, or simply having readable local output
// during development.
//
// The central entry point is New, which wraps any http.Handler and returns a
// new http.Handler that logs each request as it passes through. Internally New
// substitutes a small responseRecorder for the real http.ResponseWriter so that
// the status code, the response headers (and therefore the reported content
// length), and the moment headers are flushed can be captured without the
// wrapped handler being aware of it. Timing is measured around the call to the
// inner handler, giving both :response-time (arrival to headers written) and
// :total-time (arrival to fully written). Once the handler returns, the
// collected values are assembled into a Log and the active FormatFunc renders
// the final line.
//
// Output is controlled by a Format, which is either a predefined constant or a
// raw format string. The predefined formats mirror Node morgan exactly:
// Combined (Apache combined log format, the fullest line, including referrer
// and user agent), Common (Apache common log format), Dev (concise output whose
// :status is colored by response class — green 2xx, cyan 3xx, yellow 4xx, red
// 5xx — for terminal use), Short (like common plus response time and without
// the bracketed date), and Tiny (the minimal method/url/status/length/time
// line). This port additionally offers JSON, which emits one JSON object per
// request. Raw format strings use :token and :token[arg] placeholders; tokens
// such as :req[header] and :res[header] take a bracket argument, and a missing
// or empty token value renders as "-". Formats are compiled into a FormatFunc
// by Compile, tokens are resolved from a live registry at render time, and the
// token set can be extended at runtime with Token. Named formats can be added
// with RegisterFormat (a format string) or RegisterFormatFunc (a FormatFunc).
//
// Behaviour beyond the format is configured through Config, and its options are
// deliberately close to the Node API. Config.Stream sets the destination writer
// (the analogue of morgan's stream option, defaulting to os.Stdout);
// Config.Buffer enables interval-based buffered flushing (morgan's buffer
// option); Config.Immediate writes the line on request arrival rather than on
// response completion, trading away response fields such as status and length
// for a guarantee the request is logged even if the handler never returns; and
// Config.Skip is the equivalent of morgan's skip function, receiving the
// request and the observed status and returning true to suppress the line.
// Custom tokens (Token) and the compiled-format machinery (Compile, FormatFunc)
// round out the parity. The most notable difference from the Node library is
// naturally the type system: tokens receive a strongly typed Log rather than
// loosely typed request/response objects, and Go's io.Writer stands in for a
// Node writable stream.
package morgan

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// FromRequest populates a Log from an incoming HTTP request and timing information.
// RESPONSE_HEADERS must be set by the caller after the handler runs.
func FromRequest(r *http.Request, status int, responseTime, totalTime time.Duration) Log {
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remoteIP = host
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		remoteIP = fwd
	}

	referrer := r.Header.Get("Referer")
	if referrer == "" {
		referrer = r.Header.Get("Referrer")
	}

	url := r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	return Log{
		DATE:           time.Now(),
		HTTPVersion:    fmt.Sprintf("%d.%d", r.ProtoMajor, r.ProtoMinor),
		METHOD:         r.Method,
		PID:            os.Getpid(),
		REFERRER:       referrer,
		REMOTE_IP:      IP(net.ParseIP(remoteIP)),
		REMOTE_USER:    remoteUser(r),
		REQUEST_HEADER: r.Header.Get("Authorization"),
		STATUS:         status,
		RESPONSE_TIME:  responseTime,
		TOTAL_TIME:     totalTime,
		URL:            url,
		USER_AGENT:     r.Header.Get("User-Agent"),
		INCOMING:       r.ContentLength,
	}
}

func remoteUser(r *http.Request) string {
	if user, _, ok := r.BasicAuth(); ok {
		return user
	}
	return ""
}

// responseRecorder wraps http.ResponseWriter to capture the status code and the
// moment response headers are first written (used to distinguish RESPONSE_TIME
// from TOTAL_TIME).
type responseRecorder struct {
	http.ResponseWriter
	status     int
	headerTime time.Time
	written    bool
}

func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.written {
		rr.status = code
		rr.headerTime = time.Now()
		rr.written = true
	}
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if !rr.written {
		rr.WriteHeader(http.StatusOK)
	}
	return rr.ResponseWriter.Write(b)
}

// New returns an http.Handler middleware that logs each request using the given
// format and configuration. format may be a named format ("combined", "dev", …)
// or a raw format string with :token syntax.
//
// Example:
//
//	mux := http.NewServeMux()
//	http.ListenAndServe(":8080", morgan.New(mux, morgan.Dev, morgan.Config{}))
func New(next http.Handler, format Format, cfg Config) http.Handler {
	rawOut := cfg.Stream
	if rawOut == nil {
		rawOut = os.Stdout
	}

	formatFn := getFormatFunc(string(format))
	// Dev colours are only useful in a real terminal; fall back to plain output
	// when piping to a file or another process.
	if format == Dev && !isTerminal(rawOut) {
		formatFn = Compile(":method :url :status :response-time ms - :res[content-length]")
	}

	out := io.Writer(rawOut)
	if cfg.Buffer > 0 {
		out = newBufferStream(rawOut, cfg.Buffer)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if cfg.Immediate {
			log := FromRequest(r, 0, 0, 0)
			if cfg.Skip == nil || !cfg.Skip(r, 0) {
				if line := formatFn(r, log); line != "" {
					fmt.Fprintln(out, line)
				}
			}
			next.ServeHTTP(w, r)
			return
		}

		rr := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rr, r)

		if !rr.written {
			rr.WriteHeader(http.StatusOK)
		}

		log := FromRequest(r, rr.status, rr.headerTime.Sub(start), time.Since(start))
		log.RESPONSE_HEADERS = w.Header()

		if cfg.Skip != nil && cfg.Skip(r, rr.status) {
			return
		}

		if line := formatFn(r, log); line != "" {
			fmt.Fprintln(out, line)
		}
	})
}
