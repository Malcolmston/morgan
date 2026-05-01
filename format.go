package morgan

import (
	"fmt"
	"net/http"
	"sync"
)

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

func jsonFormatLine(r *http.Request, log Log) string {
	status := "-"
	if log.STATUS != 0 {
		status = fmt.Sprintf("%d", log.STATUS)
	}
	contentLength := "-"
	if log.RESPONSE_HEADERS != nil {
		if v := log.RESPONSE_HEADERS.Get("Content-Length"); v != "" {
			contentLength = v
		}
	}
	return fmt.Sprintf(
		`{"date":%q,"method":%q,"url":%q,"status":%s,"responseTime":%s,"totalTime":%s,"remoteAddr":%q,"userAgent":%q,"contentLength":%s}`,
		log.DATE.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		log.METHOD,
		log.URL,
		status,
		FormatDuration(log.RESPONSE_TIME, 3),
		FormatDuration(log.TOTAL_TIME, 3),
		log.REMOTE_IP.String(),
		log.USER_AGENT,
		contentLength,
	)
}

func init() {
	RegisterFormat("combined", string(Combined))
	RegisterFormat("common", string(Common))
	RegisterFormat("short", string(Short))
	RegisterFormat("tiny", string(Tiny))
	RegisterFormatFunc("dev", devFormatLine)
	RegisterFormatFunc("json", jsonFormatLine)
}
