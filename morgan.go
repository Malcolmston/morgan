package morgan

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// FromRequest populates a Log from an incoming HTTP request and timing information.
// RESPONSE_HEADERS must be set by the caller after the handler runs.
func FromRequest(r *http.Request, status int, responseTime, totalTime time.Duration) Log {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
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
	formatFn := getFormatFunc(string(format))

	out := cfg.Stream
	if out == nil {
		out = os.Stdout
	}
	if cfg.Buffer > 0 {
		out = newBufferStream(out, cfg.Buffer)
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
