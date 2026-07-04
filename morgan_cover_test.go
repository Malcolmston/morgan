package morgan

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// fixedDate is a deterministic timestamp used across the coverage tests:
// 05 Mar 2024 06:21:42 UTC (day < 10 exercises pad2's zero-pad branch).
var fixedDate = time.Date(2024, time.March, 5, 6, 21, 42, 0, time.UTC)

func TestDevFormatLineColors(t *testing.T) {
	fn := getFormatFunc("dev")
	cases := []struct {
		status int
		code   string // expected ANSI colour code for :status
	}{
		{500, "\x1b[31m"}, // red
		{404, "\x1b[33m"}, // yellow
		{304, "\x1b[36m"}, // cyan
		{200, "\x1b[32m"}, // green
		{100, "\x1b[0m"},  // none (1xx) — no colour code, plain reset
	}
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	for _, c := range cases {
		log := Log{METHOD: "GET", URL: "/x", STATUS: c.status}
		out := fn(r, log)
		if !strings.Contains(out, "GET") || !strings.Contains(out, "/x") {
			t.Errorf("status %d: missing method/url: %q", c.status, out)
		}
		if c.status >= 200 && !strings.Contains(out, c.code) {
			t.Errorf("status %d: expected colour %q in %q", c.status, c.code, out)
		}
	}
}

func TestDevColorIndex(t *testing.T) {
	for code, want := range map[int]int{31: 1, 33: 2, 36: 3, 32: 4, 0: 0, 99: 0} {
		if got := devColorIndex(code); got != want {
			t.Errorf("devColorIndex(%d) = %d, want %d", code, got, want)
		}
	}
}

func TestLogStringCombined(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Length", "128")
	log := Log{
		DATE:             fixedDate,
		REMOTE_USER:      "alice",
		METHOD:           "GET",
		URL:              "/a",
		HTTPVersion:      "1.1",
		STATUS:           200,
		RESPONSE_HEADERS: h,
		REFERRER:         "https://ref",
		USER_AGENT:       "curl/8",
	}
	s := log.String()
	for _, want := range []string{"alice", "05/Mar/2024:06:21:42 +0000", `"GET /a HTTP/1.1"`, "200", "128", "https://ref", "curl/8"} {
		if !strings.Contains(s, want) {
			t.Errorf("Log.String missing %q in:\n%s", want, s)
		}
	}
}

func TestLogStringDashesForEmpty(t *testing.T) {
	// No status, no headers, no user/referrer/agent → dashes everywhere.
	log := Log{DATE: fixedDate, METHOD: "GET", URL: "/", HTTPVersion: "1.1"}
	s := log.String()
	if !strings.Contains(s, "- - [") { // remote-user dash
		t.Errorf("expected dash for empty remote-user: %s", s)
	}
	if !strings.Contains(s, `"" "-"`) && !strings.Contains(s, `"-" "-"`) {
		// referrer & user-agent render as "-"
		if !strings.Contains(s, `"-"`) {
			t.Errorf("expected dashes for empty referrer/agent: %s", s)
		}
	}
	if strings.Contains(s, " 0 ") {
		t.Errorf("zero status should render as '-', got: %s", s)
	}
}

func TestOrDash(t *testing.T) {
	if orDash("") != "-" {
		t.Error("orDash(\"\") should be -")
	}
	if orDash("x") != "x" {
		t.Error("orDash passthrough failed")
	}
}

func TestPad2(t *testing.T) {
	if pad2(5) != "05" || pad2(25) != "25" {
		t.Errorf("pad2 = %q,%q", pad2(5), pad2(25))
	}
}

func TestTokensRendered(t *testing.T) {
	resp := http.Header{}
	resp.Set("Content-Length", "42")
	log := Log{
		DATE:             fixedDate,
		HTTPVersion:      "1.1",
		METHOD:           "POST",
		URL:              "/u",
		PID:              4242,
		REFERRER:         "https://r",
		REMOTE_USER:      "bob",
		USER_AGENT:       "UA",
		STATUS:           201,
		RESPONSE_TIME:    2 * time.Millisecond,
		TOTAL_TIME:       5 * time.Millisecond,
		RESPONSE_HEADERS: resp,
		INCOMING:         17,
	}
	r := httptest.NewRequest(http.MethodPost, "/u", nil)
	r.Header.Set("X-Trace", "abc")

	fn := Compile(":http-version|:method|:url|:status|:referrer|:remote-user|:user-agent|:pid|" +
		":response-time|:total-time|:date[clf]|:date[iso]|:date[web]|:req[X-Trace]|:res[content-length]|:incoming")
	out := fn(r, log)
	for _, want := range []string{"1.1", "POST", "/u", "201", "https://r", "bob", "UA", "4242",
		"05/Mar/2024:06:21:42 +0000", "2024-03-05T06:21:42.000Z", "abc", "42", "17"} {
		if !strings.Contains(out, want) {
			t.Errorf("token output missing %q in:\n%s", want, out)
		}
	}
}

func TestBufferStreamFlushes(t *testing.T) {
	var buf bytes.Buffer
	bs := newBufferStream(&buf, 20*time.Millisecond)
	bs.Write([]byte("a\n"))
	bs.Write([]byte("b\n"))
	if buf.Len() != 0 {
		t.Fatal("buffer flushed too early")
	}
	time.Sleep(60 * time.Millisecond)
	if got := buf.String(); got != "a\nb\n" {
		t.Fatalf("buffered output = %q, want %q", got, "a\nb\n")
	}
	// A second write after a flush re-arms the timer.
	bs.Write([]byte("c\n"))
	time.Sleep(60 * time.Millisecond)
	if got := buf.String(); got != "a\nb\nc\n" {
		t.Fatalf("second flush output = %q", got)
	}
}

func TestBufferedMiddlewareEndToEnd(t *testing.T) {
	var buf bytes.Buffer
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }),
		Tiny, Config{Stream: &buf, Buffer: 20 * time.Millisecond})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/buf", nil))
	time.Sleep(60 * time.Millisecond)
	if !strings.Contains(buf.String(), "/buf") {
		t.Fatalf("buffered middleware did not flush log: %q", buf.String())
	}
}

func TestIsTerminal(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("a bytes.Buffer is not a terminal")
	}
	// A regular file is an *os.File but not a character device.
	f, err := os.CreateTemp(t.TempDir(), "morgan")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("a regular file should not be reported as a terminal")
	}
}

func TestJSONFormatContentLength(t *testing.T) {
	resp := http.Header{}
	resp.Set("Content-Length", "9")
	log := Log{METHOD: "GET", URL: "/j", STATUS: 200, RESPONSE_HEADERS: resp, DATE: fixedDate}
	out := jsonFormatLine(httptest.NewRequest(http.MethodGet, "/j", nil), log)
	if !strings.Contains(out, `"contentLength":9`) {
		t.Errorf("json contentLength missing: %s", out)
	}
}
