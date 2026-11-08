package dhtcrawler

import (
	"net/netip"
	"testing"

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
