package morgan

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClfdate(t *testing.T) {
	// 2024-11-27T06:21:42Z
	ts := time.Date(2024, time.November, 27, 6, 21, 42, 0, time.UTC)
	want := "27/Nov/2024:06:21:42 +0000"
	if got := Clfdate(ts); got != want {
		t.Errorf("Clfdate = %q, want %q", got, want)
	}
	// A non-UTC input must be normalised to UTC / +0000.
	loc := time.FixedZone("EST", -5*3600)
	local := time.Date(2024, time.November, 27, 1, 21, 42, 0, loc)
	if got := Clfdate(local); got != want {
		t.Errorf("Clfdate(non-UTC) = %q, want %q", got, want)
	}
}

func TestClientIP(t *testing.T) {
	t.Run("remote-addr host", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "203.0.113.7:54321"
		if got := ClientIP(r); got != "203.0.113.7" {
			t.Errorf("ClientIP = %q, want 203.0.113.7", got)
		}
	})
	t.Run("x-forwarded-for wins", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "10.0.0.1:80"
		r.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.1")
		if got := ClientIP(r); got != "198.51.100.9" {
			t.Errorf("ClientIP = %q, want 198.51.100.9", got)
		}
	})
	if got := ClientIP(nil); got != "" {
		t.Errorf("ClientIP(nil) = %q, want empty", got)
	}
}

func TestRequestURL(t *testing.T) {
	cases := []struct {
		target string
		want   string
	}{
		{"/api/users", "/api/users"},
		{"/api/users?page=2&sort=asc", "/api/users?page=2&sort=asc"},
		{"/", "/"},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, c.target, nil)
		if got := RequestURL(r); got != c.want {
			t.Errorf("RequestURL(%q) = %q, want %q", c.target, got, c.want)
		}
	}
	if got := RequestURL(nil); got != "" {
		t.Errorf("RequestURL(nil) = %q, want empty", got)
	}
}

func TestRequestProtocol(t *testing.T) {
	t.Run("plain http", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if got := RequestProtocol(r); got != "http" {
			t.Errorf("RequestProtocol = %q, want http", got)
		}
	})
	t.Run("tls", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.TLS = &tls.ConnectionState{}
		if got := RequestProtocol(r); got != "https" {
			t.Errorf("RequestProtocol = %q, want https", got)
		}
	})
	t.Run("forwarded proto", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Forwarded-Proto", "https, http")
		if got := RequestProtocol(r); got != "https" {
			t.Errorf("RequestProtocol = %q, want https", got)
		}
	})
	if got := RequestProtocol(nil); got != "" {
		t.Errorf("RequestProtocol(nil) = %q, want empty", got)
	}
}

func TestNewTokensRender(t *testing.T) {
	fn := Compile(":protocol :host :path :query")
	r := httptest.NewRequest(http.MethodGet, "http://example.com/api/x?q=1", nil)
	log := FromRequest(r, 200, 0, 0)
	got := fn(r, log)
	want := "http example.com /api/x q=1"
	if got != want {
		t.Errorf("token render = %q, want %q", got, want)
	}
}

func BenchmarkClientIP(b *testing.B) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.7:54321"
	r.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.1")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ClientIP(r)
	}
}

func BenchmarkClfdate(b *testing.B) {
	ts := time.Date(2024, time.November, 27, 6, 21, 42, 0, time.UTC)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Clfdate(ts)
	}
}
