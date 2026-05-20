package driver

import (
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"github.com/hashicorp/consul/api"
	"gocloud.dev/runtimevar"
)

func newTestWatcher(t *testing.T) *Watcher {
	t.Helper()
	client, err := api.NewClient(api.DefaultConfig())
	require.NoError(t, err)
	return NewWatcher(client, "any-key", Config{
		Decoder: runtimevar.BytesDecoder,
	})
}

func TestNewWatcher_NotNil(t *testing.T) {
	w := newTestWatcher(t)
	assert.NotNil(t, w)
}

func TestWatcher_Close_NoOp(t *testing.T) {
	w := newTestWatcher(t)
	assert.NoError(t, w.Close())
}
