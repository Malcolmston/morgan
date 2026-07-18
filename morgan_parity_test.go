package morgan

// This file encodes known-answer vectors taken directly from the upstream
// Node.js expressjs/morgan test suite (test/morgan.js) and the token/format
// definitions in its index.js. Each TestParity* function asserts the Go port
// reproduces the exact strings the original library asserts, so the two stay in
// behavioural parity. Vectors are exercised at the token / Compile layer and,
// where the behaviour is middleware-level (immediate mode, skip), through
// net/http/httptest, keeping every assertion deterministic.

import (
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// newRequest builds a request with the given method, target and headers.
func newParityRequest(method, target string, headers map[string]string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

// render compiles format and renders it against r and log, matching how New
// produces a line.
func render(format string, r *http.Request, log Log) string {
	return Compile(format)(r, log)
}

// maskColor replaces every ESC[<n>m SGR sequence with _color_<n>_, mirroring the
// createColorLineStream helper in the upstream dev-format tests.
var colorRe = regexp.MustCompile("\x1b\\[([0-9]+)m")

func maskColor(s string) string {
	return colorRe.ReplaceAllString(s, "_color_${1}_")
}

// clfRe / webRe / isoRe mirror the regexes the upstream :date tests assert with.
var (
	clfRe = regexp.MustCompile(`^\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2} \+0000$`)
	webRe = regexp.MustCompile(`^\w{3}, \d{2} \w{3} \d{4} \d{2}:\d{2}:\d{2} GMT$`)
	isoRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$`)
)

// TestParityDate covers the upstream :date token tests, including the
// "blank for unknown format" case that must render "-".
func TestParityDate(t *testing.T) {
	fixed := time.Date(2024, time.November, 27, 6, 21, 42, 0, time.UTC)
	log := Log{DATE: fixed}
	r := newParityRequest("GET", "/", nil)

	if got := render(":date", r, log); !webRe.MatchString(got) {
		t.Errorf(":date web default = %q, want web format", got)
	}
	if got := render(":date[web]", r, log); !webRe.MatchString(got) {
		t.Errorf(":date[web] = %q, want web format", got)
	}
	if got := render(":date[clf]", r, log); !clfRe.MatchString(got) {
		t.Errorf(":date[clf] = %q, want clf format", got)
	}
	if got := render(":date[iso]", r, log); !isoRe.MatchString(got) {
		t.Errorf(":date[iso] = %q, want iso format", got)
	}
	// Upstream: unknown format falls through to undefined -> "-".
	if got := render(":date[unknown]", r, log); got != "-" {
		t.Errorf(":date[unknown] = %q, want %q", got, "-")
	}
	// Exact CLF rendering for the fixed instant.
	if got := render(":date[clf]", r, log); got != "27/Nov/2024:06:21:42 +0000" {
		t.Errorf(":date[clf] exact = %q", got)
	}
}

// TestParityHTTPVersion mirrors ':http-version should be 1.0 or 1.1'.
func TestParityHTTPVersion(t *testing.T) {
	log := Log{HTTPVersion: "1.1"}
	if got := render(":http-version", newParityRequest("GET", "/", nil), log); !regexp.MustCompile(`^1\.[01]$`).MatchString(got) {
		t.Errorf(":http-version = %q", got)
	}
}

// TestParityReq covers ':req should get request properties' and array headers.
func TestParityReq(t *testing.T) {
	r := newParityRequest("GET", "/", map[string]string{"X-From-String": "me"})
	if got := render(":req[x-from-string]", r, Log{}); got != "me" {
		t.Errorf(":req[x-from-string] = %q, want %q", got, "me")
	}

	r2 := newParityRequest("GET", "/", nil)
	r2.Header["Set-Cookie"] = []string{"foo=bar", "fizz=buzz"}
	if got := render(":req[set-cookie]", r2, Log{}); got != "foo=bar, fizz=buzz" {
		t.Errorf(":req array = %q, want %q", got, "foo=bar, fizz=buzz")
	}
}

// TestParityRes covers ':res should get response properties', array headers, and
// the empty-before-response case.
func TestParityRes(t *testing.T) {
	h := http.Header{}
	h.Set("X-Sent", "true")
	if got := render(":res[x-sent]", newParityRequest("GET", "/", nil), Log{RESPONSE_HEADERS: h}); got != "true" {
		t.Errorf(":res[x-sent] = %q, want %q", got, "true")
	}

	h2 := http.Header{"X-Keys": []string{"foo", "bar"}}
	if got := render(":res[x-keys]", newParityRequest("GET", "/", nil), Log{RESPONSE_HEADERS: h2}); got != "foo, bar" {
		t.Errorf(":res array = %q, want %q", got, "foo, bar")
	}

	// No response headers captured yet -> "-".
	if got := render(":res[x-sent]", newParityRequest("GET", "/", nil), Log{}); got != "-" {
		t.Errorf(":res before response = %q, want %q", got, "-")
	}
}

// TestParityRemoteUser covers the full :remote-user escaping suite from upstream,
// which guards against log-injection via a hostile Authorization header.
func TestParityRemoteUser(t *testing.T) {
	cases := []struct {
		name string
		user string // raw Basic-auth username
		want string
	}{
		{"empty", "", "-"},
		{"basic", "tj", "tj"},
		{"crlf", "evil\r\ninjected", `evil\r\ninjected`},
		{"null", "evil\x00null", `evil\u0000null`},
		{"tab-esc", "evil\t\x1buser", `evil\t\u001buser`},
		{"backslash", `evil\nuser`, `evil\\nuser`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := render(":remote-user", newParityRequest("GET", "/", nil), Log{REMOTE_USER: tc.user})
			if got != tc.want {
				t.Errorf(":remote-user(%q) = %q, want %q", tc.user, got, tc.want)
			}
			if strings.ContainsAny(got, "\r\n\x00\t\x1b") {
				t.Errorf(":remote-user(%q) leaked a control character: %q", tc.user, got)
			}
		})
	}
}

// TestParityRemoteUserFromAuthHeader confirms the escaping applies end-to-end
// when the username arrives via a real Basic Authorization header (base64
// vectors copied from the upstream tests).
func TestParityRemoteUserFromAuthHeader(t *testing.T) {
	cases := []struct {
		auth string
		want string
	}{
		{"Basic dGo6", "tj"}, // tj:
		{"Basic ZXZpbA0KaW5qZWN0ZWQ6eA==", `evil\r\ninjected`}, // evil\r\ninjected:x
		{"Basic ZXZpbABudWxsOng=", `evil\u0000null`},           // evil\x00null:x
		{"Basic ZXZpbAkbdXNlcjp4", `evil\t\u001buser`},         // evil\t\x1buser:x
		{"Basic ZXZpbFxudXNlcjp4", `evil\\nuser`},              // evil\nuser:x (literal backslash)
		{"Basic Og==", "-"},                                    // : (empty user)
	}
	for _, tc := range cases {
		r := newParityRequest("GET", "/", map[string]string{"Authorization": tc.auth})
		log := FromRequest(r, 200, time.Millisecond, time.Millisecond)
		if got := render(":remote-user", r, log); got != tc.want {
			t.Errorf("auth %q -> :remote-user = %q, want %q", tc.auth, got, tc.want)
		}
	}
}

// TestParityStatus covers ':status should get response status' and the
// before-response ("-") case.
func TestParityStatus(t *testing.T) {
	if got := render(":status", newParityRequest("GET", "/", nil), Log{STATUS: 200}); got != "200" {
		t.Errorf(":status = %q, want %q", got, "200")
	}
	if got := render(":status", newParityRequest("GET", "/", nil), Log{STATUS: 0}); got != "-" {
		t.Errorf(":status before response = %q, want %q", got, "-")
	}
}

// TestParityResponseTime covers digit counts and the missing-timing case.
func TestParityResponseTime(t *testing.T) {
	log := Log{RESPONSE_TIME: 1500 * time.Microsecond} // 1.5 ms
	r := newParityRequest("GET", "/", nil)

	if got := render(":response-time", r, log); !regexp.MustCompile(`^[0-9]+\.[0-9]{3}$`).MatchString(got) {
		t.Errorf(":response-time default digits = %q", got)
	}
	if got := render(":response-time[5]", r, log); !regexp.MustCompile(`^[0-9]+\.[0-9]{5}$`).MatchString(got) {
		t.Errorf(":response-time[5] = %q", got)
	}
	if got := render(":response-time[0]", r, log); !regexp.MustCompile(`^[0-9]+$`).MatchString(got) {
		t.Errorf(":response-time[0] = %q", got)
	}
	if got := render(":response-time", r, log); got != "1.500" {
		t.Errorf(":response-time exact = %q, want %q", got, "1.500")
	}
	// Negative sentinel (missing timing) -> "-".
	if got := render(":response-time", r, Log{RESPONSE_TIME: -1}); got != "-" {
		t.Errorf(":response-time missing = %q, want %q", got, "-")
	}
}

// TestParityTotalTime mirrors TestParityResponseTime for :total-time.
func TestParityTotalTime(t *testing.T) {
	log := Log{TOTAL_TIME: 2500 * time.Microsecond} // 2.5 ms
	r := newParityRequest("GET", "/", nil)
	if got := render(":total-time[0]", r, log); !regexp.MustCompile(`^[0-9]+$`).MatchString(got) {
		t.Errorf(":total-time[0] = %q", got)
	}
	if got := render(":total-time", r, log); got != "2.500" {
		t.Errorf(":total-time exact = %q, want %q", got, "2.500")
	}
	if got := render(":total-time", r, Log{TOTAL_TIME: -1}); got != "-" {
		t.Errorf(":total-time missing = %q, want %q", got, "-")
	}
}

// TestParityURL covers ':url should get request URL', including query strings.
func TestParityURL(t *testing.T) {
	if got := render(":url", newParityRequest("GET", "/foo", nil), Log{URL: "/foo"}); got != "/foo" {
		t.Errorf(":url = %q, want %q", got, "/foo")
	}
	log := FromRequest(newParityRequest("GET", "/foo?bar=baz", nil), 200, 0, 0)
	if got := render(":url", newParityRequest("GET", "/foo?bar=baz", nil), log); got != "/foo?bar=baz" {
		t.Errorf(":url with query = %q, want %q", got, "/foo?bar=baz")
	}
}

// TestParityFormatString covers the "a string" format tests: plain tokens, text
// mixed with tokens, and special characters.
func TestParityFormatString(t *testing.T) {
	log := Log{METHOD: "GET", URL: "/"}
	r := newParityRequest("GET", "/", nil)

	if got := render(":method :url", r, log); got != "GET /" {
		t.Errorf("%q, want %q", got, "GET /")
	}
	if got := render("method=:method url=:url", r, log); got != "method=GET url=/" {
		t.Errorf("%q, want %q", got, "method=GET url=/")
	}

	// 'LOCAL\:remote-user ":method :url" :status' with user "tobi", status 200.
	log2 := Log{METHOD: "GET", URL: "/", STATUS: 200, REMOTE_USER: "tobi"}
	if got := render(`LOCAL\:remote-user ":method :url" :status`, r, log2); got != `LOCAL\tobi "GET /" 200` {
		t.Errorf("special chars = %q, want %q", got, `LOCAL\tobi "GET /" 200`)
	}
}

// TestParityUnknownToken mirrors the compiler behaviour where an unregistered
// token renders "-" (via the `|| "-"` fallback in upstream compile()).
func TestParityUnknownToken(t *testing.T) {
	if got := render(":does-not-exist", newParityRequest("GET", "/", nil), Log{}); got != "-" {
		t.Errorf(":does-not-exist = %q, want %q", got, "-")
	}
}

// TestParityPredefinedFormats covers combined/common/short/tiny against the
// masked-timestamp expectations from upstream. The remote address is fixed so
// the whole line is deterministic.
func TestParityPredefinedFormats(t *testing.T) {
	tsRe := regexp.MustCompile(`\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2} \+0000`)
	timerRe := regexp.MustCompile(`[0-9]+\.[0-9]{3}`)

	base := Log{
		DATE:          time.Date(2024, time.November, 27, 6, 21, 42, 0, time.UTC),
		HTTPVersion:   "1.1",
		METHOD:        "GET",
		URL:           "/",
		STATUS:        200,
		REMOTE_IP:     IP(mustIP("127.0.0.1")),
		REMOTE_USER:   "tj",
		REFERRER:      "http://localhost/",
		USER_AGENT:    "my-ua",
		RESPONSE_TIME: 500 * time.Microsecond,
	}
	r := newParityRequest("GET", "/", nil)

	combined := tsRe.ReplaceAllString(render(string(Combined), r, base), "_timestamp_")
	if want := `127.0.0.1 - tj [_timestamp_] "GET / HTTP/1.1" 200 - "http://localhost/" "my-ua"`; combined != want {
		t.Errorf("combined = %q, want %q", combined, want)
	}

	common := tsRe.ReplaceAllString(render(string(Common), r, base), "_timestamp_")
	if want := `127.0.0.1 - tj [_timestamp_] "GET / HTTP/1.1" 200 -`; common != want {
		t.Errorf("common = %q, want %q", common, want)
	}

	short := timerRe.ReplaceAllString(render(string(Short), r, base), "_timer_")
	if want := `127.0.0.1 tj GET / HTTP/1.1 200 - - _timer_ ms`; short != want {
		t.Errorf("short = %q, want %q", short, want)
	}

	tiny := timerRe.ReplaceAllString(render(string(Tiny), r, base), "_timer_")
	if want := `GET / 200 - - _timer_ ms`; tiny != want {
		t.Errorf("tiny = %q, want %q", tiny, want)
	}
}

// mustIP parses a dotted-quad for the fixture; it never fails for the literals used.
func mustIP(s string) []byte {
	return net.ParseIP(s)
}

// TestParityDevColors covers the five dev-format color cases. The colored output
// is masked to _color_<n>_ exactly as the upstream createColorLineStream does.
func TestParityDevColors(t *testing.T) {
	cases := []struct {
		status int
		prefix string
	}{
		{102, "_color_0_GET / _color_0_102_color_0_"},
		{200, "_color_0_GET / _color_32_200_color_0_"},
		{300, "_color_0_GET / _color_36_300_color_0_"},
		{400, "_color_0_GET / _color_33_400_color_0_"},
		{500, "_color_0_GET / _color_31_500_color_0_"},
	}
	r := newParityRequest("GET", "/", nil)
	for _, tc := range cases {
		log := Log{METHOD: "GET", URL: "/", STATUS: tc.status, RESPONSE_TIME: time.Millisecond}
		line := maskColor(devFormatLine(r, log))
		if !strings.HasPrefix(line, tc.prefix) {
			t.Errorf("dev %d: %q does not start with %q", tc.status, line, tc.prefix)
		}
		if !strings.HasSuffix(line, "_color_0_") {
			t.Errorf("dev %d: %q does not end with _color_0_", tc.status, line)
		}
	}
}

// TestParityImmediate covers the "with immediate option" tests: :res,
// :response-time and :status all render "-" because no response has started.
func TestParityImmediate(t *testing.T) {
	cases := []struct {
		format string
		want   string
	}{
		{":method :url :res[x-sent]", "GET / -"},
		{":method :url :response-time", "GET / -"},
		{":method :url :status", "GET / -"},
	}
	for _, tc := range cases {
		var buf strings.Builder
		var logged bool
		h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Immediate mode must log before the handler writes the response.
			if !logged {
				logged = true
			}
			w.WriteHeader(200)
		}), Format(tc.format), Config{Immediate: true, Stream: &buf})

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, newParityRequest("GET", "/", nil))

		got := strings.TrimRight(buf.String(), "\n")
		if got != tc.want {
			t.Errorf("immediate %q = %q, want %q", tc.format, got, tc.want)
		}
	}
}

// TestParitySkip covers the "with skip option" tests: skipping by request URL and
// by response status suppresses the line entirely.
func TestParitySkip(t *testing.T) {
	// Skip based on request URL.
	var buf strings.Builder
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), Common, Config{
		Stream: &buf,
		Skip: func(r *http.Request, status int) bool {
			return strings.Contains(r.URL.RawQuery, "skip=true")
		},
	})
	h.ServeHTTP(httptest.NewRecorder(), newParityRequest("GET", "/?skip=true", nil))
	if buf.Len() != 0 {
		t.Errorf("skip-by-request logged %q, want empty", buf.String())
	}

	// Skip based on response status.
	var buf2 strings.Builder
	h2 := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), Common, Config{
		Stream: &buf2,
		Skip:   func(r *http.Request, status int) bool { return status == 200 },
	})
	h2.ServeHTTP(httptest.NewRecorder(), newParityRequest("GET", "/", nil))
	if buf2.Len() != 0 {
		t.Errorf("skip-by-status logged %q, want empty", buf2.String())
	}
}
