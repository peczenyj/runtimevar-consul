package consulvar

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/runtimevar"
)

func testClient(t *testing.T) *api.Client {
	t.Helper()
	client, err := api.NewClient(api.DefaultConfig())
	require.NoError(t, err)
	return client
}

func TestOpenVariable_RequiresClient(t *testing.T) {
	v, err := OpenVariable(nil, "k", nil)
	assert.Nil(t, v)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")
}

func TestOpenVariable_RequiresKey(t *testing.T) {
	v, err := OpenVariable(testClient(t), "", nil)
	assert.Nil(t, v)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key is required")
}

func TestOpenVariable_NilOptionsUsesDefaults(t *testing.T) {
	v, err := OpenVariable(testClient(t), "some/key", nil)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.NoError(t, v.Close())
}

func TestOpenVariable_WithOptions(t *testing.T) {
	v, err := OpenVariable(testClient(t), "some/key", &Options{
		Decoder:    runtimevar.StringDecoder,
		Datacenter: "dc1",
		Namespace:  "team-a",
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.NoError(t, v.Close())
}
