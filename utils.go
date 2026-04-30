package main

import (
	"fmt"
	"time"
)

// FormatDuration formats a duration as milliseconds with the given number of decimal digits.
func FormatDuration(d time.Duration, digits int) string {
	ms := float64(d) / float64(time.Millisecond)
	return fmt.Sprintf("%.*f", digits, ms)
}

// String returns a combined log format line for the request.
func (l Log) String() string {
	status := "-"
	if l.STATUS != 0 {
		status = fmt.Sprintf("%d", l.STATUS)
	}
	return fmt.Sprintf("%s - %s [%s] \"%s %s HTTP/%s\" %s %s",
		l.REMOTE_IP,
		orDash(l.REMOTE_USER),
		time.Now().Format("02/Jan/2006:15:04:05 -0700"),
		l.METHOD,
		l.URL,
		l.HTTPVersion,
		status,
		FormatDuration(l.TOTAL_TIME, 3),
	)
}

// orDash returns the input string if it is not empty; otherwise, it returns a hyphen ("-").
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
