package tmdb

import (
	"fmt"
	"regexp"

	"go.uber.org/zap"
)

// The TMDB v3 API only accepts the API key as a query parameter, so it is part of every request URL,
// and appears in the errors and retry warnings that include the URL.
var apiKeyParamPattern = regexp.MustCompile(`(api_key=)[^&\s"'\\]+`)

const redactedValue = "REDACTED"

func redactAPIKey(s string) string {
	return apiKeyParamPattern.ReplaceAllString(s, "${1}"+redactedValue)
}

// redactedError hides the API key in the message of an error, while preserving the wrapped error.
type redactedError struct {
	err error
}

func redactError(err error) error {
	if err == nil {
		return nil
	}

	return redactedError{err: err}
}

func (e redactedError) Error() string {
	return redactAPIKey(e.err.Error())
}

func (e redactedError) Unwrap() error {
	return e.err
}

// redactingLogger is a resty logger that hides the API key in log messages.
type redactingLogger struct {
	logger *zap.SugaredLogger
}

func (l redactingLogger) Errorf(format string, v ...interface{}) {
	l.logger.Error(redactAPIKey(fmt.Sprintf(format, v...)))
}

func (l redactingLogger) Warnf(format string, v ...interface{}) {
	l.logger.Warn(redactAPIKey(fmt.Sprintf(format, v...)))
}

func (l redactingLogger) Debugf(format string, v ...interface{}) {
	l.logger.Debug(redactAPIKey(fmt.Sprintf(format, v...)))
}
