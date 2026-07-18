package morgan

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func reqWithUA(ua, path string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "http://example.com"+path, nil)
	if ua != "" {
		r.Header.Set("User-Agent", ua)
	}
	return r
}

func TestSkipStatusBelow(t *testing.T) {
	skip := SkipStatusBelow(400)
	cases := []struct {
		status int
		want   bool
	}{
		{0, true},
		{200, true},
		{399, true},
		{400, false},
		{500, false},
	}
	for _, c := range cases {
		if got := skip(nil, c.status); got != c.want {
			t.Errorf("SkipStatusBelow(400)(%d) = %v, want %v", c.status, got, c.want)
		}
	}
}

func TestSkipStatusBetween(t *testing.T) {
	skip := SkipStatusBetween(200, 399)
	cases := []struct {
		status int
		want   bool
	}{
		{100, false},
		{200, true},
		{300, true},
		{399, true},
		{400, false},
	}
	for _, c := range cases {
		if got := skip(nil, c.status); got != c.want {
			t.Errorf("SkipStatusBetween(200,399)(%d) = %v, want %v", c.status, got, c.want)
		}
	}
	// Swapped bounds must behave identically.
	if !SkipStatusBetween(399, 200)(nil, 250) {
		t.Error("SkipStatusBetween should normalise swapped bounds")
	}
}

func TestSkipPaths(t *testing.T) {
	skip := SkipPaths("/healthz", "/metrics")
	cases := []struct {
		path string
		want bool
	}{
		{"/healthz", true},
		{"/metrics", true},
		{"/metrics?verbose=1", true}, // query ignored, path matches
		{"/api/users", false},
		{"/health", false},
	}
	for _, c := range cases {
		if got := skip(reqWithUA("", c.path), 200); got != c.want {
			t.Errorf("SkipPaths(%q) = %v, want %v", c.path, got, c.want)
		}
	}
	if skip(nil, 200) {
		t.Error("SkipPaths(nil request) should not match")
	}
}

func TestSkipUserAgents(t *testing.T) {
	skip := SkipUserAgents("kube-probe", "ELB-HealthChecker")
	cases := []struct {
		ua   string
		want bool
	}{
		{"kube-probe/1.27", true},
		{"Mozilla/5.0 ELB-HealthChecker/2.0", true},
		{"elb-healthchecker/2.0", true}, // case-insensitive
		{"curl/8.7.1", false},
		{"", false},
	}
	for _, c := range cases {
		if got := skip(reqWithUA(c.ua, "/"), 200); got != c.want {
			t.Errorf("SkipUserAgents(%q) = %v, want %v", c.ua, got, c.want)
		}
	}
	// Empty needles are ignored (must not match everything).
	if SkipUserAgents("")(reqWithUA("curl", "/"), 200) {
		t.Error("empty substring should not match")
	}
}

func TestCombineSkips(t *testing.T) {
	skip := CombineSkips(
		SkipPaths("/healthz"),
		SkipStatusBelow(400),
		nil, // nil entries ignored
	)
	cases := []struct {
		path   string
		status int
		want   bool
	}{
		{"/healthz", 200, true}, // path matches
		{"/api", 200, true},     // status below 400
		{"/api", 500, false},    // neither
		{"/healthz", 500, true}, // path matches
	}
	for _, c := range cases {
		if got := skip(reqWithUA("", c.path), c.status); got != c.want {
			t.Errorf("CombineSkips(%q,%d) = %v, want %v", c.path, c.status, got, c.want)
		}
	}
	if CombineSkips()(nil, 200) {
		t.Error("CombineSkips() with no funcs should never skip")
	}
}

// TestSkipFuncAssignableToConfig verifies a SkipFunc can be assigned to Config.Skip.
func TestSkipFuncAssignableToConfig(t *testing.T) {
	cfg := Config{Skip: SkipStatusBelow(400)}
	if cfg.Skip == nil {
		t.Fatal("SkipFunc not assignable to Config.Skip")
	}
	if !cfg.Skip(nil, 200) {
		t.Error("assigned skip func behaved unexpectedly")
	}
}
