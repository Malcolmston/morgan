package morgan

import (
	"net"
	"net/http"
	"time"
)

// Clfdate formats t in Common Log Format, for example "27/Nov/2024:06:21:42
// +0000". The value is always expressed in UTC with a fixed +0000 offset, exactly
// matching the :date[clf] token and the Node.js morgan clfdate helper. It is
// exported so callers building custom format functions can reuse the identical
// timestamp rendering.
func Clfdate(t time.Time) string {
	return clfdate(t)
}

// ClientIP resolves the client IP address for a request using the same
// precedence morgan applies to the :remote-addr token: the first value of the
// X-Forwarded-For header when present, otherwise the host portion of
// r.RemoteAddr. The returned string has any port stripped. An empty string is
// returned when neither source yields an address (including a nil request).
//
// X-Forwarded-For may contain a comma-separated list of proxies; the left-most
// (original client) entry is returned, trimmed of surrounding whitespace.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip := fwd
		if i := indexByte(fwd, ','); i >= 0 {
			ip = fwd[:i]
		}
		return trimSpace(ip)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// RequestURL returns the request target rendered exactly as the :url token: the
// URL path, with "?" and the raw query string appended when a query is present. A
// nil request or nil URL yields an empty string. It lets custom format functions
// reproduce morgan's :url without duplicating the assembly logic.
func RequestURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	url := r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}
	return url
}

// RequestProtocol reports the scheme under which a request was received: "https"
// when the connection is TLS-terminated (r.TLS != nil) or when a trusted proxy
// set X-Forwarded-Proto to "https", and "http" otherwise. It mirrors the
// protocol reasoning used by Express req.protocol behind a proxy and backs the
// :protocol token.
func RequestProtocol(r *http.Request) string {
	if r == nil {
		return ""
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		if i := indexByte(proto, ','); i >= 0 {
			proto = proto[:i]
		}
		return trimSpace(proto)
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// indexByte reports the index of the first occurrence of c in s, or -1.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// trimSpace removes leading and trailing ASCII spaces and tabs from s without
// importing strings, keeping the helper allocation-light for the hot logging path.
func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
