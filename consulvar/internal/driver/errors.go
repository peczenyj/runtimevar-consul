package driver

import (
	"context"
	"errors"

	"github.com/hashicorp/consul/api"
	"gocloud.dev/gcerrors"
)

// errKeyNotFound is the sentinel returned (wrapped) when Consul reports the
// watched key does not exist. It is not exported; callers should inspect the
// gcerrors.ErrorCode of the user-facing error.
var errKeyNotFound = errors.New("key not found")

// ErrorAs unwrap and checks if err is a target error.
func (w *Watcher) ErrorAs(err error, target any) bool {
	return errors.As(err, target)
}

// ErrorCode maps internal errors to gcerrors codes for the runtimevar.Variable.
func (w *Watcher) ErrorCode(err error) gcerrors.ErrorCode {
	if err == nil {
		return gcerrors.OK
	}

	var se api.StatusError
	if errors.As(err, &se) {
		switch se.Code {
		case 401, 403:
			return gcerrors.PermissionDenied
		case 429:
			return gcerrors.ResourceExhausted
		}
	}

	switch {
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
