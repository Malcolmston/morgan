package morgan

import "testing"

func TestStatusCategory(t *testing.T) {
	cases := []struct {
		status int
		want   string
	}{
		{0, "unknown"},
		{99, "unknown"},
		{100, "informational"},
		{101, "informational"},
		{200, "success"},
		{204, "success"},
		{301, "redirect"},
		{399, "redirect"},
		{404, "client-error"},
		{451, "client-error"},
		{500, "server-error"},
		{599, "server-error"},
		{600, "unknown"},
	}
	for _, c := range cases {
		if got := StatusCategory(c.status); got != c.want {
			t.Errorf("StatusCategory(%d) = %q, want %q", c.status, got, c.want)
		}
	}
}

func TestStatusColorCode(t *testing.T) {
	cases := []struct {
		status int
		want   int
	}{
		{100, 0},
		{199, 0},
		{200, 32},
		{299, 32},
		{301, 36},
		{404, 33},
		{500, 31},
		{503, 31},
		{0, 0},
	}
	for _, c := range cases {
		if got := StatusColorCode(c.status); got != c.want {
			t.Errorf("StatusColorCode(%d) = %d, want %d", c.status, got, c.want)
		}
	}
}
