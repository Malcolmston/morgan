package morgan

import "sync"

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

func init() {
	RegisterFormat("combined", string(Combined))
	RegisterFormat("common", string(Common))
	RegisterFormat("short", string(Short))
	RegisterFormat("tiny", string(Tiny))
	RegisterFormatFunc("dev", devFormatLine)
}
