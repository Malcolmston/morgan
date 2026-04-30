package tests

import (
	"net/http"
	"testing"

	"github.com/Malcolmston/morgan"
)

func TestCompile_Literal(t *testing.T) {
	fn := morgan.Compile("hello world")
	if got := fn(nil, morgan.Log{}); got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestCompile_Empty(t *testing.T) {
	fn := morgan.Compile("")
	if got := fn(nil, morgan.Log{}); got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestCompile_SingleToken(t *testing.T) {
	fn := morgan.Compile(":method")
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	log := morgan.Log{METHOD: "POST"}
	if got := fn(r, log); got != "POST" {
		t.Errorf("got %q, want %q", got, "POST")
	}
}

func TestCompile_TokenWithArg(t *testing.T) {
	morgan.Token("echo", func(_ *http.Request, _ morgan.Log, args ...string) string {
		if len(args) > 0 {
			return args[0]
		}
		return ""
	})
	fn := morgan.Compile(":echo[hello]")
	if got := fn(nil, morgan.Log{}); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestCompile_UnknownToken_DefaultsDash(t *testing.T) {
	fn := morgan.Compile(":no-such-token")
	if got := fn(nil, morgan.Log{}); got != "-" {
		t.Errorf("got %q, want %q", got, "-")
	}
}

func TestCompile_EmptyTokenReturn_DefaultsDash(t *testing.T) {
	morgan.Token("empty-tok", func(_ *http.Request, _ morgan.Log, _ ...string) string {
		return ""
	})
	fn := morgan.Compile(":empty-tok")
	if got := fn(nil, morgan.Log{}); got != "-" {
		t.Errorf("got %q, want %q", got, "-")
	}
}

func TestCompile_Mixed(t *testing.T) {
	log := morgan.Log{METHOD: "GET", URL: "/foo"}
	fn := morgan.Compile(":method :url")
	if got := fn(nil, log); got != "GET /foo" {
		t.Errorf("got %q, want %q", got, "GET /foo")
	}
}

func TestCompile_AdjacentTokens(t *testing.T) {
	log := morgan.Log{METHOD: "DELETE", STATUS: 204}
	fn := morgan.Compile(":method:status")
	if got := fn(nil, log); got != "DELETE204" {
		t.Errorf("got %q, want %q", got, "DELETE204")
	}
}

func TestCompile_LiteralSurroundingToken(t *testing.T) {
	log := morgan.Log{STATUS: 200}
	fn := morgan.Compile("[status=:status]")
	if got := fn(nil, log); got != "[status=200]" {
		t.Errorf("got %q, want %q", got, "[status=200]")
	}
}

func TestCompile_TokenRegisteredAfterCompile(t *testing.T) {
	fn := morgan.Compile(":late-token")
	// token is registered after Compile — should still resolve
	morgan.Token("late-token", func(_ *http.Request, _ morgan.Log, _ ...string) string {
		return "arrived"
	})
	if got := fn(nil, morgan.Log{}); got != "arrived" {
		t.Errorf("got %q, want %q", got, "arrived")
	}
}
