package morgan

// StatusCategory classifies an HTTP status code into its RFC 7231 response class
// and returns a lowercase, hyphenated label:
//
//	1xx -> "informational"
//	2xx -> "success"
//	3xx -> "redirect"
//	4xx -> "client-error"
//	5xx -> "server-error"
//
// A status of 0 (no response written) or any code outside the 100-599 range
// returns "unknown". This mirrors the response-class reasoning the dev format
// uses to colour the status code, exposed for use in custom tokens and Skip
// functions.
func StatusCategory(status int) string {
	switch {
	case status >= 100 && status < 200:
		return "informational"
	case status >= 200 && status < 300:
		return "success"
	case status >= 300 && status < 400:
		return "redirect"
	case status >= 400 && status < 500:
		return "client-error"
	case status >= 500 && status < 600:
		return "server-error"
	default:
		return "unknown"
	}
}

// StatusColorCode returns the ANSI Select Graphic Rendition (SGR) colour code the
// dev format applies to a status of the given class:
//
//	5xx -> 31 (red)
//	4xx -> 33 (yellow)
//	3xx -> 36 (cyan)
//	2xx -> 32 (green)
//	otherwise -> 0 (no colour)
//
// The returned value is the numeric parameter of an ESC[<code>m sequence, so a
// coloured status can be rendered as "\x1b[" + strconv.Itoa(StatusColorCode(s)) +
// "m". It reproduces exactly the palette documented for morgan.Dev.
func StatusColorCode(status int) int {
	switch {
	case status >= 500:
		return 31
	case status >= 400:
		return 33
	case status >= 300:
		return 36
	case status >= 200:
		return 32
	default:
		return 0
	}
}
