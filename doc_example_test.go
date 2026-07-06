package morgan_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/malcolmston/morgan"
)

// ExampleNew demonstrates the most common use of the package: wrapping an
// existing http.Handler with a predefined format. Here the Tiny format is used,
// which renders the method, URL, status, response content length and response
// time. The middleware is driven with an httptest request and recorder so the
// example is fully self-contained and does not need a live network listener.
// Log output is redirected into a bytes buffer via Config.Stream instead of the
// default os.Stdout so the test can inspect it. The takeaway is that a single
// call to New turns any handler into a logged handler without changing the
// handler itself. Because the Tiny format includes a non-deterministic
// response-time value, the example asserts only that a stable substring is
// present rather than printing the whole line.
func ExampleNew() {
	var buf strings.Builder

	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	logged := morgan.New(app, morgan.Tiny, morgan.Config{Stream: &buf})

	req := httptest.NewRequest(http.MethodGet, "/hi", nil)
	rec := httptest.NewRecorder()
	logged.ServeHTTP(rec, req)

	fmt.Println(strings.Contains(buf.String(), "GET /hi 200"))
	// Output: true
}

// ExampleToken shows how to extend morgan with a custom token and then use that
// token inside a raw format string. A token named "app-version" is registered
// with Token; its handler ignores the request and returns a fixed string, but a
// real handler could read anything from the request or Log. The format string
// combines the built-in :method and :url tokens with the new :app-version
// token, and New compiles it because the string is not a registered format
// name. Tokens are resolved from the live registry at render time, so
// registering the token before the request is served is sufficient. The
// takeaway is that any per-request value you can compute can become a first
// class token; because this format contains no timing or date tokens, its
// output is deterministic and can be asserted exactly.
func ExampleToken() {
	morgan.Token("app-version", func(_ *http.Request, _ morgan.Log, _ ...string) string {
		return "1.0.0"
	})

	var buf strings.Builder

	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	logged := morgan.New(app, morgan.Format(":method :url :app-version"), morgan.Config{Stream: &buf})

	req := httptest.NewRequest(http.MethodGet, "/widgets", nil)
	rec := httptest.NewRecorder()
	logged.ServeHTTP(rec, req)

	fmt.Print(buf.String())
	// Output: GET /widgets 1.0.0
}

// ExampleConfig_skip demonstrates directing output to a custom io.Writer while
// suppressing routine requests with a Skip function. The Skip callback receives
// the request and the observed status code and returns true to drop the line,
// so returning status < 400 here means only client and server errors are
// logged. Two requests are driven through the same middleware: a successful one
// that Skip suppresses, and a not-found one that is kept. The example inspects
// the buffer after each request to show the filtering taking effect. The
// takeaway is that Config.Stream and Config.Skip together let you send access
// logs anywhere an io.Writer can reach while logging only the requests you care
// about, mirroring the stream and skip options of the Node morgan middleware.
func ExampleConfig_skip() {
	var buf strings.Builder

	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, "ok")
	})

	cfg := morgan.Config{
		Stream: &buf,
		Skip: func(_ *http.Request, status int) bool {
			return status < 400
		},
	}
	logged := morgan.New(app, morgan.Tiny, cfg)

	okReq := httptest.NewRequest(http.MethodGet, "/ok", nil)
	logged.ServeHTTP(httptest.NewRecorder(), okReq)
	fmt.Println("logged after success:", buf.Len() > 0)

	missReq := httptest.NewRequest(http.MethodGet, "/missing", nil)
	logged.ServeHTTP(httptest.NewRecorder(), missReq)
	fmt.Println("has 404 line:", strings.Contains(buf.String(), "GET /missing 404"))
	// Output:
	// logged after success: false
	// has 404 line: true
}
