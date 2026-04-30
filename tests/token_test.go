package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Malcolmston/morgan"
)

// compile1 compiles a single-token format and renders it against r and log.
func compile1(format string, r *http.Request, log morgan.Log) string {
	return morgan.Compile(format)(r, log)
}

func TestToken_Method(t *testing.T) {
	log := morgan.Log{METHOD: "PATCH"}
	if got := compile1(":method", nil, log); got != "PATCH" {
		t.Errorf("got %q", got)
	}
}

func TestToken_URL(t *testing.T) {
	log := morgan.Log{URL: "/api/v1"}
	if got := compile1(":url", nil, log); got != "/api/v1" {
		t.Errorf("got %q", got)
	}
}

func TestToken_Status_Nonzero(t *testing.T) {
	log := morgan.Log{STATUS: 404}
	if got := compile1(":status", nil, log); got != "404" {
		t.Errorf("got %q", got)
	}
}

func TestToken_Status_Zero_DefaultsDash(t *testing.T) {
	log := morgan.Log{STATUS: 0}
	if got := compile1(":status", nil, log); got != "-" {
		t.Errorf("got %q, want %q", got, "-")
	}
}

func TestToken_HTTPVersion(t *testing.T) {
	log := morgan.Log{HTTPVersion: "1.1"}
	if got := compile1(":http-version", nil, log); got != "1.1" {
		t.Errorf("got %q", got)
	}
}

func TestToken_RemoteAddr(t *testing.T) {
	log := morgan.Log{REMOTE_IP: morgan.IP{127, 0, 0, 1}}
	if got := compile1(":remote-addr", nil, log); got != "127.0.0.1" {
		t.Errorf("got %q", got)
	}
}

func TestToken_RemoteUser_Empty_DefaultsDash(t *testing.T) {
	if got := compile1(":remote-user", nil, morgan.Log{}); got != "-" {
		t.Errorf("got %q, want %q", got, "-")
	}
}

func TestToken_RemoteUser(t *testing.T) {
	log := morgan.Log{REMOTE_USER: "alice"}
	if got := compile1(":remote-user", nil, log); got != "alice" {
		t.Errorf("got %q", got)
	}
}

func TestToken_UserAgent(t *testing.T) {
	log := morgan.Log{USER_AGENT: "curl/8.7.1"}
	if got := compile1(":user-agent", nil, log); got != "curl/8.7.1" {
		t.Errorf("got %q", got)
	}
}

func TestToken_ResponseTime_DefaultDigits(t *testing.T) {
	log := morgan.Log{RESPONSE_TIME: 1500 * time.Microsecond}
	got := compile1(":response-time", nil, log)
	if got != "1.500" {
		t.Errorf("got %q, want %q", got, "1.500")
	}
}

func TestToken_ResponseTime_CustomDigits(t *testing.T) {
	log := morgan.Log{RESPONSE_TIME: 1500 * time.Microsecond}
	got := compile1(":response-time[1]", nil, log)
	if got != "1.5" {
		t.Errorf("got %q, want %q", got, "1.5")
	}
}

func TestToken_TotalTime(t *testing.T) {
	log := morgan.Log{TOTAL_TIME: 2 * time.Millisecond}
	got := compile1(":total-time", nil, log)
	if got != "2.000" {
		t.Errorf("got %q, want %q", got, "2.000")
	}
}

func TestToken_Date_CLF(t *testing.T) {
	fixed := time.Date(2024, time.November, 27, 6, 21, 42, 0, time.UTC)
	log := morgan.Log{DATE: fixed}
	got := compile1(":date[clf]", nil, log)
	want := "27/Nov/2024:06:21:42 +0000"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToken_Date_ISO(t *testing.T) {
	fixed := time.Date(2024, time.November, 27, 6, 21, 42, 0, time.UTC)
	log := morgan.Log{DATE: fixed}
	got := compile1(":date[iso]", nil, log)
	want := "2024-11-27T06:21:42.000Z"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToken_Date_Web(t *testing.T) {
	fixed := time.Date(2024, time.November, 27, 6, 21, 42, 0, time.UTC)
	log := morgan.Log{DATE: fixed}
	got := compile1(":date[web]", nil, log)
	want := "Wed, 27 Nov 2024 06:21:42 GMT"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToken_Req_Header(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Request-ID", "abc123")
	got := compile1(":req[x-request-id]", r, morgan.Log{})
	if got != "abc123" {
		t.Errorf("got %q, want %q", got, "abc123")
	}
}

func TestToken_Req_MultiValue(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Accept", "text/html")
	r.Header.Add("Accept", "application/json")
	got := compile1(":req[accept]", r, morgan.Log{})
	if got != "text/html, application/json" {
		t.Errorf("got %q, want %q", got, "text/html, application/json")
	}
}

func TestToken_Req_Missing_DefaultsDash(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	got := compile1(":req[x-no-such-header]", r, morgan.Log{})
	if got != "-" {
		t.Errorf("got %q, want %q", got, "-")
	}
}

func TestToken_Res_Header(t *testing.T) {
	h := make(http.Header)
	h.Set("Content-Length", "42")
	log := morgan.Log{RESPONSE_HEADERS: h}
	got := compile1(":res[content-length]", nil, log)
	if got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}
}

func TestToken_Res_NilHeaders_DefaultsDash(t *testing.T) {
	got := compile1(":res[content-length]", nil, morgan.Log{})
	if got != "-" {
		t.Errorf("got %q, want %q", got, "-")
	}
}

func TestToken_Custom(t *testing.T) {
	morgan.Token("request-id", func(r *http.Request, _ morgan.Log, _ ...string) string {
		if r == nil {
			return ""
		}
		return r.Header.Get("X-Request-ID")
	})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Request-ID", "xyz-789")
	got := compile1(":request-id", r, morgan.Log{})
	if got != "xyz-789" {
		t.Errorf("got %q, want %q", got, "xyz-789")
	}
}

func TestToken_Overwrite(t *testing.T) {
	morgan.Token("overwrite-me", func(_ *http.Request, _ morgan.Log, _ ...string) string { return "v1" })
	morgan.Token("overwrite-me", func(_ *http.Request, _ morgan.Log, _ ...string) string { return "v2" })
	got := compile1(":overwrite-me", nil, morgan.Log{})
	if got != "v2" {
		t.Errorf("got %q, want %q", got, "v2")
	}
}
