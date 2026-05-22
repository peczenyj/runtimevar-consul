package consulapi

import (
	"github.com/hashicorp/consul/api"
)

// ConsulKV abstracts the HashiCorp Consul KV API.
type ConsulKV interface {
	Get(key string, q *api.QueryOptions) (*api.KVPair, *api.QueryMeta, error)
}

// ConsulClient abstracts the HashiCorp Consul Client API.
type ConsulClient interface {
	KV() ConsulKV
}

var _ ConsulKV = (*api.KV)(nil)
