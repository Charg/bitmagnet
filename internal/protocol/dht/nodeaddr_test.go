package dht

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalNodeAddr(t *testing.T) {
	t.Parallel()

	var na NodeAddr

	require.NoError(t, na.UnmarshalBinary([]byte("\x01\x02\x03\x04\x05\x06")))
	assert.Equal(t, "1.2.3.4", na.IP.String())
}

func TestUnmarshalNodeAddrIPv6(t *testing.T) {
	t.Parallel()

	var na NodeAddr

	require.NoError(t, na.UnmarshalBinary([]byte("\x20\x01\x0d\xb8\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x1a\xe1")))
	assert.Equal(t, "2001:db8::1", na.IP.String())
	assert.Equal(t, 6881, na.Port)
}

func TestUnmarshalNodeAddrInvalidLength(t *testing.T) {
	t.Parallel()

	for _, b := range []string{"", "\x01", "\x01\x02", "\x01\x02\x03\x04\x05", "\x01\x02\x03\x04\x05\x06\x07"} {
		var na NodeAddr

		assert.Error(t, na.UnmarshalBinary([]byte(b)), "length %d", len(b))
	}
}
