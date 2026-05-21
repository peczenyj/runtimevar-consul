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
	// failures counts consecutive transport errors for the exponential backoff
	// schedule. Not synchronized: gocloud.dev/runtimevar calls WatchVariable
	// serially, never concurrently.
	failures int
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
