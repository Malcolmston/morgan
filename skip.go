package morgan

import (
	"net/http"
	"strings"
)

// SkipFunc decides whether the log line for a request/response pair should be
// suppressed. It has the same signature as Config.Skip and any SkipFunc may be
// assigned directly to that field.
//
// Returning true suppresses the line; returning false logs it. This mirrors the
// skip option of the Node.js morgan library, whose README documents the same
// "return true to skip" convention.
type SkipFunc func(r *http.Request, status int) bool

// SkipStatusBelow returns a SkipFunc that suppresses logging whenever the
// response status code is below min. It is the Go form of morgan's canonical
// "only log errors" example, SkipStatusBelow(400), which skips every 1xx/2xx/3xx
// response and logs only 4xx and 5xx.
//
// A status of 0 (no response written, e.g. under Config.Immediate) is treated as
// below any positive min and is therefore skipped.
func SkipStatusBelow(min int) SkipFunc {
	return func(_ *http.Request, status int) bool {
		return status < min
	}
}

// SkipStatusBetween returns a SkipFunc that suppresses logging whenever the
// response status code is within the inclusive range [lo, hi]. It is useful for
// silencing an uninteresting band of statuses, for example SkipStatusBetween(200,
// 399) to log only redirects-and-errors-excluded... i.e. only 1xx, 4xx and 5xx.
func SkipStatusBetween(lo, hi int) SkipFunc {
	if lo > hi {
		lo, hi = hi, lo
	}
	return func(_ *http.Request, status int) bool {
		return status >= lo && status <= hi
	}
}

// SkipPaths returns a SkipFunc that suppresses logging for any request whose URL
// path exactly matches one of the given paths. It is the standard way to keep
// health-check and readiness endpoints (for example "/healthz" or "/metrics")
// out of an access log.
//
// Matching is performed against r.URL.Path only; query strings are ignored. A nil
// request never matches.
func SkipPaths(paths ...string) SkipFunc {
	set := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		set[p] = struct{}{}
	}
	return func(r *http.Request, _ int) bool {
		if r == nil || r.URL == nil {
			return false
		}
		_, ok := set[r.URL.Path]
		return ok
	}
}

// SkipUserAgents returns a SkipFunc that suppresses logging for any request whose
// User-Agent header contains one of the given case-insensitive substrings. It is
// commonly used to drop noise from uptime monitors, load-balancer probes and
// crawlers, for example SkipUserAgents("kube-probe", "ELB-HealthChecker").
//
// An empty substring is ignored so that it never matches every request.
func SkipUserAgents(substrings ...string) SkipFunc {
	needles := make([]string, 0, len(substrings))
	for _, s := range substrings {
		if s != "" {
			needles = append(needles, strings.ToLower(s))
		}
	}
	return func(r *http.Request, _ int) bool {
		if r == nil {
			return false
		}
		ua := strings.ToLower(r.Header.Get("User-Agent"))
		if ua == "" {
			return false
		}
		for _, n := range needles {
			if strings.Contains(ua, n) {
				return true
			}
		}
		return false
	}
}

// CombineSkips returns a SkipFunc that suppresses logging when any of the given
// SkipFuncs would suppress it (a logical OR). Nil entries are ignored. Calling
// CombineSkips with no functions returns a SkipFunc that never skips.
//
// It lets several independent policies be composed, for example:
//
//	Skip: morgan.CombineSkips(
//	    morgan.SkipPaths("/healthz"),
//	    morgan.SkipStatusBelow(400),
//	)
func CombineSkips(fns ...SkipFunc) SkipFunc {
	return func(r *http.Request, status int) bool {
		for _, fn := range fns {
			if fn != nil && fn(r, status) {
				return true
			}
		}
		return false
	}
}
