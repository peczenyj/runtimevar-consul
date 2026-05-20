// Package driver implements the gocloud.dev/runtimevar driver.Watcher
// interface backed by HashiCorp Consul KV blocking queries.
//
// This package is internal to the consulvar module; importers should depend
// on github.com/peczenyj/runtimevar-contrib/consulvar instead.
package driver

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/consul/api"
	"gocloud.dev/gcerrors"
	"gocloud.dev/runtimevar"
)

// Config carries the optional knobs forwarded from the public consulvar
// package. The Decoder field is required.
type Config struct {
	Decoder    *runtimevar.Decoder
	Datacenter string
	Namespace  string
	WaitTime   time.Duration
}

// Watcher implements gocloud.dev/runtimevar/driver.Watcher for a single
// Consul KV key using a blocking query (WaitIndex) per WatchVariable call.
type Watcher struct {
	client    *api.Client
	key       string
	decoder   *runtimevar.Decoder
	baseQuery api.QueryOptions
	// failures tracks consecutive WatchVariable errors for backoff; wired up
	// by the watch loop introduced in a later phase.
	failures int //nolint:unused // populated by upcoming WatchVariable implementation
}

// NewWatcher constructs a Watcher. The caller owns client; Close is a no-op.
func NewWatcher(client *api.Client, key string, cfg Config) *Watcher {
	return &Watcher{
		client:  client,
		key:     key,
		decoder: cfg.Decoder,
		baseQuery: api.QueryOptions{
			Datacenter: cfg.Datacenter,
			Namespace:  cfg.Namespace,
			WaitTime:   cfg.WaitTime,
		},
	}
}

// Close releases watcher resources. The caller-owned *api.Client is left
// alone, so this is a no-op.
func (w *Watcher) Close() error { return nil }

// state is the driver.State implementation returned from WatchVariable.
type state struct {
	val any
	err error
	// modifyIndex carries the Consul KV ModifyIndex used to chain blocking
	// queries; consumed by the watch loop introduced in a later phase.
	modifyIndex uint64 //nolint:unused // populated by upcoming WatchVariable implementation
	raw         *api.KVPair
	updated     time.Time
}

func (s *state) Value() (any, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.val, nil
}

func (s *state) UpdateTime() time.Time { return s.updated }

// As supports `**api.KVPair` so callers can reach the underlying Flags,
// Session, etc. from the last observed pair. Returns false otherwise.
func (s *state) As(i any) bool {
	p, ok := i.(**api.KVPair)
	if !ok {
		return false
	}
	*p = s.raw
	return true
}

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
