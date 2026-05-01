package morgan

import (
	"net/http"
	"regexp"
	"strings"
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
// Token syntax: :name or :name[arg].
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
