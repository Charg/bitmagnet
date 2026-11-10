package metainforequester

import "errors"

// Stages of a meta info request, used to label failures.
const (
	StageDial               = "dial"
	StageBtHandshake        = "bt_handshake"
	StageExtensionHandshake = "extension_handshake"
	StageRequestPieces      = "request_pieces"
	StageReadPieces         = "read_pieces"
	StageParse              = "parse"
	StageUnknown            = "unknown"
)

// stageError records the stage at which a meta info request failed.
type stageError struct {
	stage string
	err   error
	// ctxErr is the request context's error at the time of failure, if any: when the request times out, the
	// connection is closed under the in-flight read or write, so the underlying error alone would be misleading.
	ctxErr error
}

func newStageError(stage string, err error, ctxErr error) error {
	return stageError{stage: stage, err: err, ctxErr: ctxErr}
}

func (e stageError) Error() string {
	return e.stage + ": " + e.err.Error()
}

func (e stageError) Unwrap() []error {
	if e.ctxErr != nil {
		return []error{e.err, e.ctxErr}
	}

	return []error{e.err}
}

// errorStage returns the stage at which err occurred.
func errorStage(err error) string {
	var se stageError
	if errors.As(err, &se) {
		return se.stage
	}

	return StageUnknown
}
