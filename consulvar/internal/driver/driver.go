// Package driver implements the gocloud.dev/runtimevar driver.Watcher
// interface backed by HashiCorp Consul KV blocking queries.
//
// This package is internal to the consulvar module; importers should depend
// on github.com/peczenyj/runtimevar-contrib/consulvar instead.
package driver

import (
	"time"

	"github.com/hashicorp/consul/api"
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
	failures  int
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
	val         any
	err         error
	modifyIndex uint64
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
