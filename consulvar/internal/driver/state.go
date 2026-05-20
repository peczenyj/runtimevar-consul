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
