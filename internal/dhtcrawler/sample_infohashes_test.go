package dhtcrawler

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/protocol/dht"

	"github.com/stretchr/testify/assert"
)

func TestSampleInfoHashesInterval(t *testing.T) {
	t.Parallel()

	// a large interval is honoured but capped
	assert.Equal(t, 21600, sampleInfoHashesInterval(21600, 1000, 20))
	assert.Equal(t, maxSampleInfoHashesInterval, sampleInfoHashesInterval(100000, 10, 10))
	// a short interval and many more hashes than returned allows an early re-query
	assert.Equal(t, 60, sampleInfoHashesInterval(300, 1000, 20))
	assert.Equal(t, 10, sampleInfoHashesInterval(10, 1000, 20))
	// a short interval but few hashes: honour the interval
	assert.Equal(t, 300, sampleInfoHashesInterval(300, 30, 20))
}

func TestSeenNodes(t *testing.T) {
	t.Parallel()

	s := newSeenNodes()
	a := netip.MustParseAddrPort("1.2.3.4:6881")
	assert.False(t, s.testAndAdd(a))
	assert.True(t, s.testAndAdd(a))
	assert.False(t, s.testAndAdd(netip.MustParseAddrPort("1.2.3.4:6882")))
}

func TestAsKRPCError(t *testing.T) {
	t.Parallel()

	methodUnknown := dht.Error{Code: dht.ErrorCodeMethodUnknown, Msg: "method unknown"}

	for name, err := range map[string]error{
		"pointer":         &methodUnknown,
		"value":           methodUnknown,
		"wrapped pointer": fmt.Errorf("query failed: %w", &methodUnknown),
	} {
		krpcErr, ok := asKRPCError(err)
		assert.True(t, ok, name)
		assert.Equal(t, dht.ErrorCodeMethodUnknown, krpcErr.Code, name)
	}

	for name, err := range map[string]error{
		"timeout": context.DeadlineExceeded,
		"other":   errors.New("boom"),
	} {
		_, ok := asKRPCError(err)
		assert.False(t, ok, name)
	}
}
