//go:build js && wasm

// Command morgan (wasm) exposes morgan's pure log-formatting helpers to
// JavaScript. Built with GOOS=js GOARCH=wasm it registers a `__mgo_morgan`
// object on the JS global so the very same Go implementation that formats HTTP
// log lines in Go also runs in the browser or Node.
//
// Only the portable, request-independent parts of morgan are exposed: compiling
// a format string (or named format) and rendering it against a plain log object,
// plus the FormatDuration helper. The net/http middleware (morgan.New, which
// wraps an http.Handler) is intentionally NOT exposed — it isn't portable to
// wasm. See morgan.mjs for the idiomatic JS wrapper.
package main

import (
	"net"
	"net/http"
	"sort"
	"strconv"
	"syscall/js"
	"time"

	"github.com/malcolmston/morgan"
)

// namedFormats maps JS-facing format names to morgan's pure, Compile-able
// format strings. morgan's "dev" and "json" formats are registered internally
// as FormatFuncs (not part of the exported API) and so are not reachable here;
// they are deliberately omitted rather than reimplemented.
var namedFormats = map[string]string{
	"combined": string(morgan.Combined),
	"common":   string(morgan.Common),
	"short":    string(morgan.Short),
	"tiny":     string(morgan.Tiny),
}

// builtinTokens lists the token names morgan registers by default. Kept in sync
// with token.go's init(). Tokens reading only from the Log (method, url, status,
// response-time, …) render fine here; :req reads the original *http.Request,
// which the JS side does not supply, so it renders as "-".
var builtinTokens = []string{
	"http-version", "method", "url", "status", "referrer",
	"remote-addr", "remote-user", "user-agent", "pid",
	"response-time", "total-time", "date", "req", "res", "incoming",
}

func main() {
	obj := js.Global().Get("Object").New()
	obj.Set("format", js.FuncOf(formatFn))
	obj.Set("formatDuration", js.FuncOf(formatDurationFn))
	obj.Set("tokens", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		return toJSArray(builtinTokens)
	}))
	obj.Set("formats", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		names := make([]string, 0, len(namedFormats))
		for n := range namedFormats {
			names = append(names, n)
		}
		sort.Strings(names)
		return toJSArray(names)
	}))
	js.Global().Set("__mgo_morgan", obj)

	select {} // keep the Go runtime alive so the exported funcs stay callable
}

// formatFn(nameOrFormatString, logObj) compiles the given named or raw format
// string and renders it against a morgan.Log built from logObj. Since the chosen
// tokens read from the Log (not a live *http.Request), a nil request is passed.
func formatFn(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return ""
	}
	nameOrFmt := args[0].String()
	fmtStr := nameOrFmt
	if s, ok := namedFormats[nameOrFmt]; ok {
		fmtStr = s
	}
	fn := morgan.Compile(fmtStr)

	var log morgan.Log
	if len(args) > 1 && args[1].Type() == js.TypeObject {
		log = buildLog(args[1])
	}
	return fn(nil, log)
}

// formatDurationFn(ms, digits) formats a millisecond value with the given number
// of decimal digits, mirroring morgan.FormatDuration.
func formatDurationFn(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return ""
	}
	digits := 3
	if len(args) > 1 && args[1].Type() == js.TypeNumber {
		digits = args[1].Int()
	}
	return morgan.FormatDuration(msToDuration(args[0].Float()), digits)
}

// buildLog maps a plain JS log object onto morgan.Log. All fields are optional;
// responseTime/totalTime are given in milliseconds and converted to durations,
// and contentLength is stored as a response Content-Length header (which is how
// the :res[content-length] token reads it).
func buildLog(o js.Value) morgan.Log {
	var log morgan.Log
	log.INCOMING = -1

	if v := o.Get("method"); v.Type() == js.TypeString {
		log.METHOD = v.String()
	}
	if v := o.Get("url"); v.Type() == js.TypeString {
		log.URL = v.String()
	}
	if v := o.Get("httpVersion"); v.Type() == js.TypeString {
		log.HTTPVersion = v.String()
	}
	if v := o.Get("status"); v.Type() == js.TypeNumber {
		log.STATUS = v.Int()
	}
	if v := o.Get("referrer"); v.Type() == js.TypeString {
		log.REFERRER = v.String()
	}
	if v := o.Get("remoteUser"); v.Type() == js.TypeString {
		log.REMOTE_USER = v.String()
	}
	if v := o.Get("userAgent"); v.Type() == js.TypeString {
		log.USER_AGENT = v.String()
	}
	if v := o.Get("remoteAddr"); v.Type() == js.TypeString {
		if ip := net.ParseIP(v.String()); ip != nil {
			log.REMOTE_IP = morgan.IP(ip)
		}
	}
	if v := o.Get("pid"); v.Type() == js.TypeNumber {
		log.PID = v.Int()
	}
	if v := o.Get("responseTime"); v.Type() == js.TypeNumber {
		log.RESPONSE_TIME = msToDuration(v.Float())
	}
	if v := o.Get("totalTime"); v.Type() == js.TypeNumber {
		log.TOTAL_TIME = msToDuration(v.Float())
	}
	if v := o.Get("incoming"); v.Type() == js.TypeNumber {
		log.INCOMING = int64(v.Float())
	}
	if v := o.Get("date"); v.Type() == js.TypeNumber {
		log.DATE = time.UnixMilli(int64(v.Float())).UTC()
	}
	if v := o.Get("contentLength"); !v.IsUndefined() && !v.IsNull() {
		h := http.Header{}
		switch v.Type() {
		case js.TypeNumber:
			h.Set("Content-Length", strconv.Itoa(v.Int()))
		case js.TypeString:
			h.Set("Content-Length", v.String())
		}
		log.RESPONSE_HEADERS = h
	}
	return log
}

// msToDuration converts a floating-point millisecond value to a time.Duration.
func msToDuration(ms float64) time.Duration {
	return time.Duration(ms * float64(time.Millisecond))
}

func toJSArray(vals []string) js.Value {
	arr := js.Global().Get("Array").New(len(vals))
	for i, v := range vals {
		arr.SetIndex(i, v)
	}
	return arr
}
