package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/malcolmston/morgan"
)

func TestRegisterFormat_StringLookup(t *testing.T) {
	morgan.RegisterFormat("test-fmt", ":method :url")
	// Use the registered name as the format argument to New
	var buf strings.Builder
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	mw := morgan.New(handler, "test-fmt", morgan.Config{Stream: &buf})

	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	mw.ServeHTTP(nopWriter{}, req)

	got := buf.String()
	if !strings.Contains(got, "GET") || !strings.Contains(got, "/ping") {
		t.Errorf("unexpected log line: %q", got)
	}
}

func TestRegisterFormatFunc(t *testing.T) {
	morgan.RegisterFormatFunc("fn-fmt", func(_ *http.Request, log morgan.Log) string {
		return "fn:" + log.METHOD
	})

	var buf strings.Builder
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	mw := morgan.New(handler, "fn-fmt", morgan.Config{Stream: &buf})

	req, _ := http.NewRequest(http.MethodDelete, "/", nil)
	mw.ServeHTTP(nopWriter{}, req)

	if got := strings.TrimSpace(buf.String()); got != "fn:DELETE" {
		t.Errorf("got %q, want %q", got, "fn:DELETE")
	}
}

func TestRegisterFormat_Overwrite(t *testing.T) {
	morgan.RegisterFormat("overwrite-fmt", ":method")
	morgan.RegisterFormat("overwrite-fmt", ":url")

	var buf strings.Builder
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	mw := morgan.New(handler, "overwrite-fmt", morgan.Config{Stream: &buf})

	req, _ := http.NewRequest(http.MethodGet, "/overwritten", nil)
	mw.ServeHTTP(nopWriter{}, req)

	got := strings.TrimSpace(buf.String())
	if got != "/overwritten" {
		t.Errorf("got %q, want %q", got, "/overwritten")
	}
}

func TestPredefinedFormat_Tiny(t *testing.T) {
	var buf strings.Builder
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := morgan.New(handler, morgan.Tiny, morgan.Config{Stream: &buf})
	req, _ := http.NewRequest(http.MethodGet, "/tiny", nil)
	mw.ServeHTTP(nopWriter{}, req)

	line := buf.String()
	for _, want := range []string{"GET", "/tiny", "200"} {
		if !strings.Contains(line, want) {
			t.Errorf("Tiny format: expected %q in %q", want, line)
		}
	}
}

func TestPredefinedFormat_Short(t *testing.T) {
	var buf strings.Builder
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	mw := morgan.New(handler, morgan.Short, morgan.Config{Stream: &buf})
	req, _ := http.NewRequest(http.MethodPost, "/short", nil)
	mw.ServeHTTP(nopWriter{}, req)

	line := buf.String()
	for _, want := range []string{"POST", "/short", "201"} {
		if !strings.Contains(line, want) {
			t.Errorf("Short format: expected %q in %q", want, line)
		}
	}
}

func TestPredefinedFormat_Combined(t *testing.T) {
	var buf strings.Builder
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := morgan.New(handler, morgan.Combined, morgan.Config{Stream: &buf})
	req, _ := http.NewRequest(http.MethodGet, "/combined", nil)
	req.Header.Set("User-Agent", "test-agent")
	mw.ServeHTTP(nopWriter{}, req)

	line := buf.String()
	for _, want := range []string{"GET", "/combined", "200", "test-agent"} {
		if !strings.Contains(line, want) {
			t.Errorf("Combined format: expected %q in %q", want, line)
		}
	}
}

func TestPredefinedFormat_Common(t *testing.T) {
	var buf strings.Builder
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mw := morgan.New(handler, morgan.Common, morgan.Config{Stream: &buf})
	req, _ := http.NewRequest(http.MethodGet, "/common", nil)
	mw.ServeHTTP(nopWriter{}, req)

	line := buf.String()
	for _, want := range []string{"GET", "/common", "204"} {
		if !strings.Contains(line, want) {
			t.Errorf("Common format: expected %q in %q", want, line)
		}
	}
}

// nopWriter is a minimal http.ResponseWriter for tests that don't care about the response.
type nopWriter struct{}

func (nopWriter) Header() http.Header         { return http.Header{} }
func (nopWriter) Write(b []byte) (int, error) { return len(b), nil }
func (nopWriter) WriteHeader(int)             {}
