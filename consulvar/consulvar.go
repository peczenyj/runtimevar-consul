// Package consulvar provides a gocloud.dev/runtimevar driver backed by
// HashiCorp Consul KV. It watches a single key with Consul blocking queries
// and surfaces each change as a runtimevar.Snapshot.
//
// # URLs
//
// For runtimevar.OpenVariable, consulvar registers for the "consul" scheme.
// The URL's host and path are joined to form the Consul KV key, so
// "consul://services/auth/db_url" watches the key "services/auth/db_url".
// Connection settings are read from the standard Consul environment variables
// (CONSUL_HTTP_ADDR, CONSUL_HTTP_TOKEN, …) via api.DefaultConfig().
//
// The following query parameters are supported:
//   - decoder:     one of "bytes" (default), "string", or "jsonmap"; a
//     "decrypt+<decoder>" prefix is also accepted. See runtimevar.DecoderByName.
//   - datacenter:  the Consul datacenter to query.
//   - namespace:   the Consul Enterprise namespace.
//   - allow_stale: whether to allow stale reads ("true" or "false").
//   - wait_time:   max blocking-query duration, e.g. "30s"; see time.ParseDuration.
//
// Example:
//
//	v, err := runtimevar.OpenVariable(ctx, "consul://services/auth/db_url?decoder=string")
//
// # As
//
// consulvar exposes the underlying Consul pair through Snapshot.As: pass a
// **github.com/hashicorp/consul/api.KVPair to reach Flags, Session, and the
// other fields of the last observed pair.
//
// You can also pass a **github.com/hashicorp/consul/api.QueryMeta to extract
// Consul query metrics like RequestTime and KnownLeader for successful reads.
package consulvar

import (
	"errors"
	"time"

	"github.com/hashicorp/consul/api"
	"gocloud.dev/runtimevar"

	"github.com/peczenyj/runtimevar-consul/consulvar/internal/driver"
)

// Options sets optional parameters for OpenVariable.
type Options struct {
	// Decoder decodes the raw Consul value into the Snapshot.Value. It defaults
	// to runtimevar.BytesDecoder, which yields the raw []byte.
	Decoder *runtimevar.Decoder

	// Datacenter, if set, selects the Consul datacenter to query.
	Datacenter string

	// Namespace, if set, selects the Consul Enterprise namespace.
	Namespace string

	// AllowStale allows Consul to return stale data from any server, scaling
	// read throughput at the cost of potential consistency delays.
	AllowStale bool

	// WaitTime caps the duration of each blocking query (the 'wait' parameter
	// in Consul KV queries). Zero uses Consul's server-side default (typically
	// 5 minutes).
	WaitTime time.Duration
}

// OpenVariable opens a runtimevar.Variable that watches the Consul KV key using
// blocking queries against client. The caller owns client; closing the
// returned Variable does not close it.
func OpenVariable(client *api.Client, key string, opts *Options) (*runtimevar.Variable, error) {
	if client == nil {
		return nil, errors.New("consulvar: client is required")
	}
	if key == "" {
		return nil, errors.New("consulvar: key is required")
	}
	if opts == nil {
		opts = &Options{}
	}
	decoder := opts.Decoder
	if decoder == nil {
		decoder = runtimevar.BytesDecoder
	}
	w := driver.NewWatcher(client, key, driver.Config{
		Decoder:    decoder,
		Datacenter: opts.Datacenter,
		Namespace:  opts.Namespace,
		WaitTime:   opts.WaitTime,
	})
	return runtimevar.New(w), nil
}
