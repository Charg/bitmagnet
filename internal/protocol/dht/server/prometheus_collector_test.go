package server

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol/errreason"
	"github.com/stretchr/testify/assert"
)

func TestQueryErrorReason(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err  error
		want string
	}{
		{context.DeadlineExceeded, errreason.Timeout},
		{&dht.Error{Code: dht.ErrorCodeMethodUnknown, Msg: "method unknown"}, reasonKrpcError},
		{dht.Error{Code: dht.ErrorCodeProtocolError, Msg: "bad"}, reasonKrpcError},
		{fmt.Errorf("%w: return data missing from response", ErrInvalidResponse), reasonInvalidResponse},
		{fmt.Errorf("%w: %w", ErrSendFailed, syscall.EPERM), reasonSendFailed},
		{errors.New("something else"), errreason.Other},
	}

	for _, c := range cases {
		assert.Equal(t, c.want, queryErrorReason(c.err), "%v", c.err)
	}
}
