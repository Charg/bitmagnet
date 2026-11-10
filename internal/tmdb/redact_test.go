package tmdb

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const testAPIKey = "0123456789abcdef0123456789abcdef"

func TestRedactAPIKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		`Get "https://api.themoviedb.org/3/search/movie?api_key=REDACTED&query=x": connection reset`,
		redactAPIKey(
			`Get "https://api.themoviedb.org/3/search/movie?api_key=`+testAPIKey+`&query=x": connection reset`,
		),
	)
	assert.Equal(t,
		"https://api.themoviedb.org/3/configuration?api_key=REDACTED",
		redactAPIKey("https://api.themoviedb.org/3/configuration?api_key="+testAPIKey),
	)
	assert.Equal(t, "nothing to redact", redactAPIKey("nothing to redact"))
}

// TestRequesterDoesNotLeakAPIKey makes a request that fails at the transport level, which resty retries
// (logging a warning with the URL on each attempt) before returning a *url.Error that includes the URL.
func TestRequesterDoesNotLeakAPIKey(t *testing.T) {
	t.Parallel()

	// get a port with nothing listening on it:
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core).Sugar()

	r := requesterLogger{
		requester: requester{
			resty: resty.New().
				SetBaseURL("http://"+addr+"/3").
				SetQueryParam("api_key", testAPIKey).
				SetRetryCount(1).
				SetRetryWaitTime(time.Millisecond).
				SetRetryMaxWaitTime(time.Millisecond).
				SetLogger(redactingLogger{logger: logger}),
		},
		logger: logger,
	}

	_, reqErr := r.Request(context.Background(), "/search/movie", map[string]string{"query": "x"}, nil)
	require.Error(t, reqErr)
	assert.NotContains(t, reqErr.Error(), testAPIKey)
	assert.Contains(t, reqErr.Error(), "api_key=REDACTED")

	var urlErr *url.Error

	require.ErrorAs(t, reqErr, &urlErr, "the underlying error should still be unwrappable")

	require.NotZero(t, logs.Len())

	for _, entry := range logs.All() {
		assert.NotContains(t, entry.Message, testAPIKey)

		for _, field := range entry.Context {
			assert.NotContains(t, field.String, testAPIKey)

			if e, ok := field.Interface.(error); ok {
				assert.NotContains(t, e.Error(), testAPIKey)
			}
		}
	}
}
