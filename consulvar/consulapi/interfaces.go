// Package consulapi defines narrow interfaces over the HashiCorp Consul
// client (github.com/hashicorp/consul/api) that the driver depends on. They
// exist to decouple the driver from the concrete client and to enable mocking
// in tests; see the generated mocks under mocks/.
package consulapi

import (
	"github.com/hashicorp/consul/api"
)

var _ ConsulKV = (*api.KV)(nil)

// ConsulKV abstracts the HashiCorp Consul KV API.
type ConsulKV interface {
	Get(key string, q *api.QueryOptions) (*api.KVPair, *api.QueryMeta, error)
}

// ConsulClient abstracts the HashiCorp Consul Client API.
type ConsulClient interface {
	KV() ConsulKV
}
