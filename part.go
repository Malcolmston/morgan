package main

import (
	"net"
	"time"
)

// IP represents an IP address as a slice of bytes, typically used to manage and manipulate IP-related data.
type IP []byte

// String returns the string representation of the IP address.
func (ip IP) String() string {
	return net.IP(ip).String()
}

type Log struct {
	// :http-version: the HTTP version of the request.
	HTTPVersion string
	// :method: The HTTP method of the request.
	METHOD string
	// :pid: The process ID of the Node.js process handling the request.
	PID int
	// :referrer: The Referrer header of the request. Uses the standard mis-spelled Referer header if exists, otherwise Referrer.
	REFERRER string
	// :remote-addr: The remote address of the request.
	REMOTE_IP IP
	// :remote-user: The user authenticated as part of Basic auth for the request.
	REMOTE_USER string
	// :req[header]: The given header of the request. "-" if not present.
	REQUEST_HEADER string
	// :res[header]: The given header of the response. "-" if not present.
	RESPONSE_HEADER string
	// :response-time[digits]: Time between request arriving and response headers written, in milliseconds.
	// Digits specifies decimal precision (default 3, microsecond precision).
	RESPONSE_TIME time.Duration
	// :status: The status code of the response. Empty ("-") if cycle completes before response is sent.
	STATUS int
	// :total-time[digits]: Time between request arriving and response fully written, in milliseconds.
	// Digits specifies decimal precision (default 3, microsecond precision).
	TOTAL_TIME time.Duration
	// :url: The URL of the request.
	URL string
	// :user-agent: The contents of the User-Agent header of the request.
	USER_AGENT string
}
