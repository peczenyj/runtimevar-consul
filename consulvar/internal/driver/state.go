package driver

import (
	"time"

	"github.com/hashicorp/consul/api"
)

// state is the driver.State implementation returned from WatchVariable.
type state struct {
	val any
	err error
	// modifyIndex carries the Consul KV ModifyIndex used to chain blocking
	// queries across WatchVariable calls.
	modifyIndex uint64
	raw         *api.KVPair
	meta        *api.QueryMeta
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
// Session, etc. from the last observed pair. It also supports `**api.QueryMeta`
// to access request metrics. Returns false otherwise.
func (s *state) As(i any) bool {
	if p, ok := i.(**api.KVPair); ok {
		*p = s.raw
		return true
	}
	if p, ok := i.(**api.QueryMeta); ok {
		*p = s.meta
		return true
	}
	return false
}
