// Package errreason classifies network and protocol errors into a small, fixed set of reasons,
// suitable for use as a metric label.
package errreason

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"syscall"
)

const (
	Canceled    = "canceled"
	Timeout     = "timeout"
	Refused     = "refused"
	Reset       = "reset"
	Unreachable = "unreachable"
	EOF         = "eof"
	Closed      = "closed"
	Other       = "other"
)

// Classify returns the reason for err, or an empty string if err is nil.
func Classify(err error) string {
	var netErr net.Error

	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.Canceled):
		return Canceled
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, os.ErrDeadlineExceeded),
		errors.As(err, &netErr) && netErr.Timeout():
		return Timeout
	case errors.Is(err, syscall.ECONNREFUSED):
		return Refused
	case errors.Is(err, syscall.ECONNRESET), errors.Is(err, syscall.EPIPE):
		return Reset
	case errors.Is(err, syscall.EHOSTUNREACH), errors.Is(err, syscall.ENETUNREACH):
		return Unreachable
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return EOF
	case errors.Is(err, net.ErrClosed):
		return Closed
	default:
		return Other
	}
}
