package morgan

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// isTerminal reports whether w is a file descriptor attached to a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// FormatDuration formats a duration as milliseconds with the given number of decimal digits.
func FormatDuration(d time.Duration, digits int) string {
	ms := float64(d) / float64(time.Millisecond)
	return fmt.Sprintf("%.*f", digits, ms)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// clfdate formats t in Common Log Format: "27/Nov/2024:06:21:42 +0000"
// Always expressed in UTC with a fixed +0000 offset, matching the JS implementation.
func clfdate(t time.Time) string {
	u := t.UTC()
	return fmt.Sprintf("%s/%s/%d:%s +0000",
		pad2(u.Day()),
		u.Month().String()[:3],
		u.Year(),
		u.Format("15:04:05"),
	)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%d", n)
}

// String returns the log line in Combined format.
func (l Log) String() string {
	status := "-"
	if l.STATUS != 0 {
		status = fmt.Sprintf("%d", l.STATUS)
	}
	contentLength := "-"
	if l.RESPONSE_HEADERS != nil {
		contentLength = orDash(l.RESPONSE_HEADERS.Get("Content-Length"))
	}
	return fmt.Sprintf(`%s - %s [%s] "%s %s HTTP/%s" %s %s "%s" "%s"`,
		l.REMOTE_IP,
		orDash(l.REMOTE_USER),
		clfdate(l.DATE),
		l.METHOD,
		l.URL,
		l.HTTPVersion,
		status,
		contentLength,
		orDash(l.REFERRER),
		orDash(l.USER_AGENT),
	)
}

// bufferStream batches writes and flushes to the underlying writer at a fixed interval,
// matching the JS morgan buffer option behaviour.
type bufferStream struct {
	mu       sync.Mutex
	buf      []string
	timer    *time.Timer
	out      io.Writer
	interval time.Duration
}

func newBufferStream(out io.Writer, interval time.Duration) io.Writer {
	return &bufferStream{out: out, interval: interval}
}

func (bs *bufferStream) Write(p []byte) (int, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.buf = append(bs.buf, string(p))
	if bs.timer == nil {
		bs.timer = time.AfterFunc(bs.interval, bs.flush)
	}
	return len(p), nil
}

func (bs *bufferStream) flush() {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if len(bs.buf) > 0 {
		bs.out.Write([]byte(strings.Join(bs.buf, ""))) //nolint:errcheck
		bs.buf = bs.buf[:0]
	}
	bs.timer = nil
}
