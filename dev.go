package morgan

import (
	"fmt"
	"net/http"
	"sync"
)

// devFns caches one compiled FormatFunc per ANSI color code so that the format
// string is only parsed once per color, mirroring the JS developmentFormatLine[color] cache.
var (
	devFns   [5]FormatFunc
	devFnsMu sync.Mutex
)

func devColorIndex(code int) int {
	switch code {
	case 31:
		return 1 // red   (5xx)
	case 33:
		return 2 // yellow (4xx)
	case 36:
		return 3 // cyan  (3xx)
	case 32:
		return 4 // green (2xx)
	default:
		return 0 // no color (1xx)
	}
}

func devFormatLine(r *http.Request, log Log) string {
	var code int
	switch {
	case log.STATUS >= 500:
		code = 31
	case log.STATUS >= 400:
		code = 33
	case log.STATUS >= 300:
		code = 36
	case log.STATUS >= 200:
		code = 32
	default:
		code = 0
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
