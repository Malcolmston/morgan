package morgan

import "net/http"

// This file registers additional tokens that Express applications commonly add
// to their morgan format strings. They resolve live from the *http.Request and so
// remain accurate under both immediate and deferred logging.
func init() {
	// :protocol — "http" or "https", honouring X-Forwarded-Proto then r.TLS.
	Token("protocol", func(r *http.Request, _ Log, _ ...string) string {
		return RequestProtocol(r)
	})

	// :host — the Host header (authority) of the request.
	Token("host", func(r *http.Request, _ Log, _ ...string) string {
		if r == nil {
			return ""
		}
		return r.Host
	})

	// :path — the request path only, excluding any query string.
	Token("path", func(r *http.Request, _ Log, _ ...string) string {
		if r == nil || r.URL == nil {
			return ""
		}
		return r.URL.Path
	})

	// :query — the raw query string, excluding the leading "?".
	Token("query", func(r *http.Request, _ Log, _ ...string) string {
		if r == nil || r.URL == nil {
			return ""
		}
		return r.URL.RawQuery
	})
}
