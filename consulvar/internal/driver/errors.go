package driver

import (
	"context"
	"errors"

	"gocloud.dev/gcerrors"
)

// errKeyNotFound is the sentinel returned (wrapped) when Consul reports the
// watched key does not exist. It is not exported; callers should inspect the
// gcerrors.ErrorCode of the user-facing error.
var errKeyNotFound = errors.New("key not found")

// ErrorAs returns false. Consul's api package does not expose typed errors,
// so there's no useful conversion to offer.
func (w *Watcher) ErrorAs(_ error, _ any) bool { return false }

// ErrorCode maps internal errors to gcerrors codes for the runtimevar.Variable.
func (w *Watcher) ErrorCode(err error) gcerrors.ErrorCode {
	switch {
	case err == nil:
		return gcerrors.OK
	case errors.Is(err, context.Canceled):
		return gcerrors.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return gcerrors.DeadlineExceeded
	case errors.Is(err, errKeyNotFound):
		return gcerrors.NotFound
	default:
		return gcerrors.Unknown
	}
}
