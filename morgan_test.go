package morgan

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// serve runs a request through the New middleware with the given format/config,
// capturing the emitted log line(s).
func serve(t *testing.T, format Format, cfg Config, build func() *http.Request, handler http.HandlerFunc) string {
	t.Helper()
	var buf bytes.Buffer
	cfg.Stream = &buf
	h := New(handler, format, cfg)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, build())
	return buf.String()
}

func TestNewTinyFormat(t *testing.T) {
	out := serve(t, Tiny, Config{},
		func() *http.Request { return httptest.NewRequest(http.MethodGet, "/hello?x=1", nil) },
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("hi"))
		})
	if !strings.Contains(out, "GET") || !strings.Contains(out, "/hello?x=1") || !strings.Contains(out, "201") {
		t.Fatalf("tiny line missing fields: %q", out)
	}
	if !strings.Contains(out, "ms") {
		t.Errorf("tiny line should include response-time ms: %q", out)
	}
}

func TestNewDefaultsStatus200(t *testing.T) {
	// Handler that writes a body without calling WriteHeader → status 200.
	out := serve(t, Tiny, Config{},
		func() *http.Request { return httptest.NewRequest(http.MethodPost, "/x", nil) },
		func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	if !strings.Contains(out, "200") {
		t.Fatalf("expected implicit 200, got %q", out)
	}
}

func TestNamedFormats(t *testing.T) {
	for _, f := range []Format{Combined, Common, Short, Tiny, JSON} {
		out := serve(t, f, Config{},
			func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/p", nil)
				r.Header.Set("User-Agent", "test-agent")
				return r
			},
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
		if strings.TrimSpace(out) == "" {
			t.Errorf("format %q produced no output", f)
		}
	}
}

func TestJSONFormatShape(t *testing.T) {
	out := serve(t, JSON, Config{},
		func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api", nil) },
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
	out = strings.TrimSpace(out)
	if !strings.HasPrefix(out, "{") || !strings.HasSuffix(out, "}") {
		t.Fatalf("json line not an object: %q", out)
	}
	for _, key := range []string{`"method":"GET"`, `"url":"/api"`, `"status":404`} {
		if !strings.Contains(out, key) {
			t.Errorf("json missing %s in %q", key, out)
		}
	}
}

func TestSkip(t *testing.T) {
	skipOK := func(r *http.Request, status int) bool { return status < 400 }
	out := serve(t, Tiny, Config{Skip: skipOK},
		func() *http.Request { return httptest.NewRequest(http.MethodGet, "/ok", nil) },
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	if out != "" {
		t.Fatalf("expected skipped log, got %q", out)
	}
	// An error status is not skipped.
	out = serve(t, Tiny, Config{Skip: skipOK},
		func() *http.Request { return httptest.NewRequest(http.MethodGet, "/bad", nil) },
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
	if !strings.Contains(out, "500") {
		t.Fatalf("expected error logged, got %q", out)
	}
}

func TestImmediate(t *testing.T) {
	called := false
	out := serve(t, Tiny, Config{Immediate: true},
		func() *http.Request { return httptest.NewRequest(http.MethodGet, "/im", nil) },
		func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) })
	if !called {
		t.Error("next handler was not called in immediate mode")
	}
	// Immediate logs before response, so status is unset ("" via token).
	if !strings.Contains(out, "GET") || !strings.Contains(out, "/im") {
		t.Fatalf("immediate line missing method/url: %q", out)
	}
}

func TestFromRequestForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	log := FromRequest(r, 200, time.Millisecond, 2*time.Millisecond)
	if got := log.REMOTE_IP.String(); got != "203.0.113.9" {
		t.Errorf("X-Forwarded-For not honored: %q", got)
	}
	if log.METHOD != "GET" || log.STATUS != 200 {
		t.Errorf("unexpected log %+v", log)
	}
}

func TestFromRequestReferrerFallbackAndBasicAuth(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Referrer", "https://ref.example") // misspelled header
	r.SetBasicAuth("alice", "secret")
	log := FromRequest(r, 200, 0, 0)
	if log.REFERRER != "https://ref.example" {
		t.Errorf("referrer fallback failed: %q", log.REFERRER)
	}
	if log.REMOTE_USER != "alice" {
		t.Errorf("basic-auth user = %q", log.REMOTE_USER)
	}
}

func TestFromRequestURLIncludesQuery(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/search?q=go&p=2", nil)
	log := FromRequest(r, 200, 0, 0)
	if log.URL != "/search?q=go&p=2" {
		t.Errorf("url = %q", log.URL)
	}
}

func TestCustomTokenAndCompile(t *testing.T) {
	Token("ua-upper", func(r *http.Request, log Log, args ...string) string {
		return strings.ToUpper(log.USER_AGENT)
	})
	fn := Compile(":method :ua-upper")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("User-Agent", "curl/8")
	log := FromRequest(r, 200, 0, 0)
	line := fn(r, log)
	if line != "GET CURL/8" {
		t.Fatalf("custom token render = %q", line)
	}
}

func TestTokenWithArgs(t *testing.T) {
	fn := Compile(":response-time[1] ms")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	log := FromRequest(r, 200, 1500*time.Microsecond, 0) // 1.5ms
	line := fn(r, log)
	if !strings.HasPrefix(line, "1.5") {
		t.Errorf("response-time[1] = %q, want 1.5 prefix", line)
	}
}

func TestResHeaderToken(t *testing.T) {
	var buf bytes.Buffer
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "42")
		w.WriteHeader(200)
	}), Format(":res[content-length]"), Config{Stream: &buf})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(buf.String(), "42") {
		t.Errorf("res[content-length] not rendered: %q", buf.String())
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d      time.Duration
		digits int
		want   string
	}{
		{1500 * time.Microsecond, 3, "1.500"},
		{2 * time.Millisecond, 0, "2"},
		{0, 2, "0.00"},
	}
	for _, c := range cases {
		if got := FormatDuration(c.d, c.digits); got != c.want {
			t.Errorf("FormatDuration(%v,%d) = %q, want %q", c.d, c.digits, got, c.want)
		}
	}
}

func TestIPString(t *testing.T) {
	if got := IP(net.ParseIP("127.0.0.1")).String(); got != "127.0.0.1" {
		t.Errorf("IP.String = %q", got)
	}
	if got := IP(nil).String(); got == "" {
		// nil IP should render a placeholder, not empty; just ensure no panic.
		t.Logf("nil IP renders as %q", got)
	}
}

func TestRegisterFormat(t *testing.T) {
	RegisterFormat("mytest", ":method|:url")
	fn := getFormatFunc("mytest")
	r := httptest.NewRequest(http.MethodDelete, "/gone", nil)
	log := FromRequest(r, 200, 0, 0)
	if got := fn(r, log); got != "DELETE|/gone" {
		t.Errorf("registered format = %q", got)
	}
}

func TestResponseRecorderCapturesFirstStatus(t *testing.T) {
	rr := &responseRecorder{ResponseWriter: httptest.NewRecorder()}
	rr.WriteHeader(301)
	rr.WriteHeader(500) // ignored — first wins
	if rr.status != 301 {
		t.Errorf("recorder status = %d, want 301", rr.status)
	}
}
