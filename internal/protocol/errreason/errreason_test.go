package errreason

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err  error
		want string
	}{
		{nil, ""},
		{context.Canceled, Canceled},
		{fmt.Errorf("wrapped: %w", context.DeadlineExceeded), Timeout},
		{&net.OpError{Op: "read", Err: os.ErrDeadlineExceeded}, Timeout},
		{
			&net.OpError{Op: "dial", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}},
			Refused,
		},
		{&net.OpError{Op: "read", Err: &os.SyscallError{Syscall: "read", Err: syscall.ECONNRESET}}, Reset},
		{
			&net.OpError{Op: "dial", Err: &os.SyscallError{Syscall: "connect", Err: syscall.EHOSTUNREACH}},
			Unreachable,
		},
		{fmt.Errorf("failed to read all handshake bytes (0): %w", io.EOF), EOF},
		{io.ErrUnexpectedEOF, EOF},
		{&net.OpError{Op: "read", Err: net.ErrClosed}, Closed},
		{errors.New("peer does not support the extension protocol"), Other},
		// a deadline takes precedence over the closed connection it causes:
		{errors.Join(net.ErrClosed, context.DeadlineExceeded), Timeout},
	}

	for _, c := range cases {
		assert.Equal(t, c.want, Classify(c.err), "%v", c.err)
	}
}
