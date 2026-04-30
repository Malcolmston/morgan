package morgan

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// FormatFunc is a compiled format function that renders a single log line.
// An empty return value signals that the line should be skipped.
type FormatFunc func(r *http.Request, log Log) string

// segment is a parsed piece of a format string: literal text or a token reference.
type segment struct {
	literal string
	token   string
	arg     string
}

// segmentRe matches JS morgan's token syntax: :(name) or :(name)[arg]
// name must be at least two characters, matching :([-\w]{2,})(?:\[([^\]]+)\])?
var segmentRe = regexp.MustCompile(`:([-a-zA-Z0-9_]{2,})(?:\[([^\]]+)\])?`)

func parseFormat(format string) []segment {
	var segs []segment
	last := 0
	for _, m := range segmentRe.FindAllStringSubmatchIndex(format, -1) {
		if m[0] > last {
			segs = append(segs, segment{literal: format[last:m[0]]})
		}
		arg := ""
		if m[4] != -1 {
			arg = format[m[4]:m[5]]
		}
		segs = append(segs, segment{token: format[m[2]:m[3]], arg: arg})
		last = m[1]
	}
	if last < len(format) {
		segs = append(segs, segment{literal: format[last:]})
	}
	return segs
}

// Compile compiles a format string into a FormatFunc. Tokens are resolved from
// the live registry at render time, so tokens registered after Compile is called
// are still visible.
//
// Token syntax: :name or :name[arg]
func Compile(format string) FormatFunc {
	segs := parseFormat(format)
	return func(r *http.Request, log Log) string {
		tokenMu.RLock()
		defer tokenMu.RUnlock()
		var sb strings.Builder
		for _, seg := range segs {
			if seg.token == "" {
				sb.WriteString(seg.literal)
				continue
			}
			fn, ok := tokenMap[seg.token]
			if !ok {
				sb.WriteString("-")
				continue
			}
			var val string
			if seg.arg != "" {
				val = fn(r, log, seg.arg)
			} else {
				val = fn(r, log)
			}
			if val == "" {
				val = "-"
			}
			sb.WriteString(val)
		}
		return sb.String()
	}
}

// --- Format registry ---

var (
	formatMu  sync.RWMutex
	formatMap = map[string]any{}
)

// RegisterFormat registers a named format string.
// Overwrites any existing format with the same name.
func RegisterFormat(name, fmtString string) {
	formatMu.Lock()
	defer formatMu.Unlock()
	formatMap[name] = fmtString
}

// RegisterFormatFunc registers a named format function.
// Overwrites any existing format with the same name.
func RegisterFormatFunc(name string, fn FormatFunc) {
	formatMu.Lock()
	defer formatMu.Unlock()
	formatMap[name] = fn
}

// getFormatFunc looks up nameOrFmt in the format registry. If found as a string
// it is compiled; if found as a FormatFunc it is returned directly. If not found
// the value is treated as a raw format string and compiled.
func getFormatFunc(nameOrFmt string) FormatFunc {
	formatMu.RLock()
	v, ok := formatMap[nameOrFmt]
	formatMu.RUnlock()

	if ok {
		switch f := v.(type) {
		case FormatFunc:
			return f
		case string:
			return Compile(f)
		}
	}
	return Compile(nameOrFmt)
}

// --- Dev format ---
//
// Mirrors the JS implementation: one compiled FormatFunc per ANSI color code,
// chosen by status range and cached for reuse.

var (
	devFns   [5]FormatFunc
	devFnsMu sync.Mutex
)

func devColorIndex(code int) int {
	switch code {
	case 31:
		return 1
	case 33:
		return 2
	case 36:
		return 3
	case 32:
		return 4
	default:
		return 0
	}
}

func devFormatLine(r *http.Request, log Log) string {
	var code int
	switch {
	case log.STATUS >= 500:
		code = 31 // red
	case log.STATUS >= 400:
		code = 33 // yellow
	case log.STATUS >= 300:
		code = 36 // cyan
	case log.STATUS >= 200:
		code = 32 // green
	default:
		code = 0 // no color (1xx)
	}

	idx := devColorIndex(code)
	devFnsMu.Lock()
	if devFns[idx] == nil {
		devFns[idx] = Compile(fmt.Sprintf(
			"\x1b[0m:method :url \x1b[%dm:status\x1b[0m :response-time ms - :res[content-length]\x1b[0m",
			code,
		))
	}
	fn := devFns[idx]
	devFnsMu.Unlock()

	return fn(r, log)
}

func init() {
	RegisterFormat("combined", string(Combined))
	RegisterFormat("common", string(Common))
	RegisterFormat("short", string(Short))
	RegisterFormat("tiny", string(Tiny))
	RegisterFormatFunc("dev", devFormatLine)
}
