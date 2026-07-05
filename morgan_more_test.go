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

func TestCompileUnknownToken(t *testing.T) {
	// An unregistered token renders as "-".
	fn := Compile(":method :no-such-token-here")
	log := Log{METHOD: "GET"}
	out := fn(httptest.NewRequest(http.MethodGet, "/", nil), log)
	if out != "GET -" {
		t.Fatalf("unknown token output = %q, want %q", out, "GET -")
	}
}

func TestTokenTotalTimeWithDigits(t *testing.T) {
	fn := Compile(":total-time[1]")
	log := Log{TOTAL_TIME: 5500 * time.Microsecond} // 5.5 ms
	out := fn(nil, log)
	if out != "5.5" {
		t.Fatalf("total-time[1] = %q, want %q", out, "5.5")
	}
}

func TestTokenResponseTimeWithDigits(t *testing.T) {
	fn := Compile(":response-time[0]")
	log := Log{RESPONSE_TIME: 2 * time.Millisecond}
	out := fn(nil, log)
	if out != "2" {
		t.Fatalf("response-time[0] = %q, want %q", out, "2")
	}
}

func TestTokenDateZeroUsesNow(t *testing.T) {
	// A zero DATE falls back to the current time, so the rendered value is a
	// well-formed, non-empty web date.
	fn := Compile(":date[web]")
	out := fn(nil, Log{})
	if !strings.HasSuffix(out, "GMT") || out == "-" {
		t.Fatalf("date with zero time = %q, want a non-empty web date", out)
	}
}

func TestTokenReqNilAndMissingAndMulti(t *testing.T) {
	fnReq := Compile(":req[X-Multi]")

	// Nil request renders empty ("-").
	if got := fnReq(nil, Log{}); got != "-" {
		t.Fatalf(":req with nil request = %q, want %q", got, "-")
	}

	// Missing header renders "-".
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := fnReq(r, Log{}); got != "-" {
		t.Fatalf(":req missing header = %q, want %q", got, "-")
	}

	// Multiple header values are joined with ", ".
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.Header.Add("X-Multi", "one")
	r2.Header.Add("X-Multi", "two")
	if got := fnReq(r2, Log{}); got != "one, two" {
		t.Fatalf(":req multi = %q, want %q", got, "one, two")
	}
}

func TestTokenResNilAndMissingAndMulti(t *testing.T) {
	// No args renders "-".
	if got := Compile(":res")(nil, Log{RESPONSE_HEADERS: http.Header{}}); got != "-" {
		t.Fatalf(":res without arg = %q, want %q", got, "-")
	}

	fnRes := Compile(":res[X-Multi]")

	// Nil response headers render "-".
	if got := fnRes(nil, Log{}); got != "-" {
		t.Fatalf(":res nil headers = %q, want %q", got, "-")
	}

	// Multiple response header values are joined with ", ".
	h := http.Header{}
	h.Add("X-Multi", "a")
	h.Add("X-Multi", "b")
	if got := fnRes(nil, Log{RESPONSE_HEADERS: h}); got != "a, b" {
		t.Fatalf(":res multi = %q, want %q", got, "a, b")
	}
}

func TestTokenStatusAndIncomingEmpty(t *testing.T) {
	// A zero status and a negative INCOMING both render "-".
	fn := Compile(":status|:incoming")
	if got := fn(nil, Log{STATUS: 0, INCOMING: -1}); got != "-|-" {
		t.Fatalf("empty status/incoming = %q, want %q", got, "-|-")
	}
}

func TestNewDevFallbackWhenNotTerminal(t *testing.T) {
	// Dev format written to a non-terminal writer uses the plain (uncoloured)
	// fallback format.
	var buf bytes.Buffer
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}), Dev, Config{Stream: &buf})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/dev", nil))

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("dev fallback should not contain ANSI colour codes: %q", out)
	}
	if !strings.Contains(out, "GET") || !strings.Contains(out, "/dev") || !strings.Contains(out, "418") {
		t.Fatalf("dev fallback missing expected fields: %q", out)
	}
}

func TestNewHandlerWithoutWriteDefaultsStatus200(t *testing.T) {
	// A handler that never writes a status should still be logged with 200.
	var buf bytes.Buffer
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intentionally writes nothing.
	}), Tiny, Config{Stream: &buf})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/silent", nil))

	if !strings.Contains(buf.String(), "200") {
		t.Fatalf("expected status 200 in log, got %q", buf.String())
	}
}

func TestNewDefaultStreamIsStdout(t *testing.T) {
	// With no Stream configured, New writes to os.Stdout. Redirect it so the
	// test can observe the output without polluting the test log.
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), Tiny, Config{})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stdout", nil))

	w.Close()
	os.Stdout = orig
	var out bytes.Buffer
	if _, err := out.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "/stdout") {
		t.Fatalf("default stdout stream missing log line: %q", out.String())
	}
}

func TestIsTerminalStatError(t *testing.T) {
	// Stat on a closed file descriptor fails, so isTerminal reports false.
	f, err := os.CreateTemp(t.TempDir(), "morgan")
	if err != nil {
		t.Fatal(err)
	}
	f.Close() // close before calling, forcing Stat to error
	if isTerminal(f) {
		t.Fatal("isTerminal should be false when Stat fails")
	}
}
