package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/malcolmston/morgan"
)

func TestNew_LogsToStream(t *testing.T) {
	var buf strings.Builder
	mw := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }),
		morgan.Tiny,
		morgan.Config{Stream: &buf},
	)

	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/hello", nil))

	line := buf.String()
	if !strings.Contains(line, "GET") || !strings.Contains(line, "/hello") {
		t.Errorf("unexpected log line: %q", line)
	}
}

func TestNew_CapturesStatusCode(t *testing.T) {
	var buf strings.Builder
	mw := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(418) }),
		morgan.Tiny,
		morgan.Config{Stream: &buf},
	)

	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/tea", nil))

	if !strings.Contains(buf.String(), "418") {
		t.Errorf("expected 418 in log, got: %q", buf.String())
	}
}

func TestNew_DefaultStatus200(t *testing.T) {
	var buf strings.Builder
	mw := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// handler writes body without explicit WriteHeader → should be 200
			w.Write([]byte("ok")) //nolint:errcheck
		}),
		morgan.Tiny,
		morgan.Config{Stream: &buf},
	)

	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !strings.Contains(buf.String(), "200") {
		t.Errorf("expected 200 in log, got: %q", buf.String())
	}
}

func TestNew_Skip(t *testing.T) {
	var buf strings.Builder
	mw := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }),
		morgan.Tiny,
		morgan.Config{
			Stream: &buf,
			Skip:   func(_ *http.Request, status int) bool { return status < 400 },
		},
	)

	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ok", nil))
	if buf.Len() != 0 {
		t.Errorf("expected no log for 2xx when Skip set, got: %q", buf.String())
	}

	// 500 should be logged
	mw2 := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }),
		morgan.Tiny,
		morgan.Config{
			Stream: &buf,
			Skip:   func(_ *http.Request, status int) bool { return status < 400 },
		},
	)
	mw2.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/err", nil))
	if !strings.Contains(buf.String(), "500") {
		t.Errorf("expected 500 in log, got: %q", buf.String())
	}
}

func TestNew_Immediate(t *testing.T) {
	var buf strings.Builder
	handlerReached := false
	mw := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// log should already be written before handler runs in immediate mode
			if buf.Len() == 0 {
				t.Error("immediate: log not written before handler ran")
			}
			handlerReached = true
			w.WriteHeader(200)
		}),
		morgan.Tiny,
		morgan.Config{Stream: &buf, Immediate: true},
	)

	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/now", nil))

	if !handlerReached {
		t.Error("handler was never reached")
	}
	if !strings.Contains(buf.String(), "GET") {
		t.Errorf("unexpected log: %q", buf.String())
	}
}

func TestNew_Buffer(t *testing.T) {
	// strings.Builder is not thread-safe; use syncWriter so the timer goroutine
	// that calls bufferStream.flush() and the test goroutine don't race.
	var mu sync.Mutex
	var lines []string
	sw := &syncWriter{mu: &mu, lines: &lines}

	interval := 50 * time.Millisecond
	mw := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }),
		morgan.Tiny,
		morgan.Config{Stream: sw, Buffer: interval},
	)

	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/buf", nil))

	// should not be flushed yet
	mu.Lock()
	immediate := len(lines)
	mu.Unlock()
	if immediate != 0 {
		t.Errorf("buffered: expected no output immediately, got %d line(s)", immediate)
	}

	// wait for the buffer to flush
	time.Sleep(interval * 3)
	mu.Lock()
	count, first := len(lines), ""
	if count > 0 {
		first = lines[0]
	}
	mu.Unlock()

	if count == 0 || !strings.Contains(first, "GET") {
		t.Errorf("buffered: expected log after flush, got %d line(s), first=%q", count, first)
	}
}

func TestNew_Concurrent(t *testing.T) {
	var mu sync.Mutex
	var lines []string
	mw := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }),
		morgan.Tiny,
		morgan.Config{Stream: &syncWriter{mu: &mu, lines: &lines}},
	)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/concurrent", nil))
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(lines) != 50 {
		t.Errorf("expected 50 log lines, got %d", len(lines))
	}
}

func TestNew_QueryString(t *testing.T) {
	var buf strings.Builder
	mw := morgan.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		morgan.Tiny,
		morgan.Config{Stream: &buf},
	)

	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/search?q=go", nil))
	if !strings.Contains(buf.String(), "/search?q=go") {
		t.Errorf("expected query string in log, got: %q", buf.String())
	}
}

func TestFromRequest_PopulatesFields(t *testing.T) {
	r := httptest.NewRequest(http.MethodPut, "/items/1?v=2", nil)
	r.Header.Set("User-Agent", "go-test")
	r.Header.Set("Referer", "https://example.com")

	log := morgan.FromRequest(r, 201, 5*time.Millisecond, 10*time.Millisecond)

	if log.METHOD != "PUT" {
		t.Errorf("METHOD: got %q", log.METHOD)
	}
	if log.URL != "/items/1?v=2" {
		t.Errorf("URL: got %q", log.URL)
	}
	if log.STATUS != 201 {
		t.Errorf("STATUS: got %d", log.STATUS)
	}
	if log.USER_AGENT != "go-test" {
		t.Errorf("USER_AGENT: got %q", log.USER_AGENT)
	}
	if log.REFERRER != "https://example.com" {
		t.Errorf("REFERRER: got %q", log.REFERRER)
	}
	if log.RESPONSE_TIME != 5*time.Millisecond {
		t.Errorf("RESPONSE_TIME: got %v", log.RESPONSE_TIME)
	}
	if log.TOTAL_TIME != 10*time.Millisecond {
		t.Errorf("TOTAL_TIME: got %v", log.TOTAL_TIME)
	}
	if log.HTTPVersion != "1.1" {
		t.Errorf("HTTPVersion: got %q", log.HTTPVersion)
	}
	if log.DATE.IsZero() {
		t.Error("DATE should not be zero")
	}
}

// syncWriter is a thread-safe io.Writer that appends each write as a line.
type syncWriter struct {
	mu    *sync.Mutex
	lines *[]string
}

func (sw *syncWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	*sw.lines = append(*sw.lines, string(p))
	return len(p), nil
}
