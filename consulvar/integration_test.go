//go:build integration

package consulvar_test

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	testcontainer_consul "github.com/testcontainers/testcontainers-go/modules/consul"
	"gocloud.dev/gcerrors"
	"gocloud.dev/runtimevar"

	"github.com/peczenyj/runtimevar-consul/consulvar"
)

const watchTimeout = 30 * time.Second

// startConsul boots a throwaway Consul container and returns its HTTP API
// address plus a client pointed at it. The container is terminated on cleanup.
func startConsul(t *testing.T) (addr string, client *api.Client) {
	t.Helper()
	ctx := context.Background()

	logger := log.TestLogger(t)

	container, err := testcontainer_consul.Run(
		ctx,
		testcontainer_consul.DefaultBaseImage,
		testcontainers.WithLogger(logger),
	)

	testcontainers.CleanupContainer(t, container)

	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	addr, err = container.ApiEndpoint(ctx)
	require.NoError(t, err)

	cfg := api.DefaultConfig()
	cfg.Address = addr
	client, err = api.NewClient(cfg)
	require.NoError(t, err)
	return addr, client
}

func put(t *testing.T, client *api.Client, key, value string) {
	t.Helper()
	_, err := client.KV().Put(&api.KVPair{Key: key, Value: []byte(value)}, nil)
	require.NoError(t, err)
}

func TestIntegration(t *testing.T) {
	addr, client := startConsul(t)

	t.Run("watch reflects initial value and update", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), watchTimeout)
		defer cancel()

		key := "config/app/db_url"
		put(t, client, key, "v1")

		v, err := consulvar.OpenVariable(client, key, &consulvar.Options{Decoder: runtimevar.StringDecoder})
		require.NoError(t, err)
		defer v.Close()

		snap, err := v.Watch(ctx)
		require.NoError(t, err)
		assert.Equal(t, "v1", snap.Value.(string))

		put(t, client, key, "v2")

		snap, err = v.Watch(ctx)
		require.NoError(t, err)
		assert.Equal(t, "v2", snap.Value.(string))
	})

	t.Run("missing key surfaces NotFound", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), watchTimeout)
		defer cancel()

		v, err := consulvar.OpenVariable(client, "config/does/not/exist", nil)
		require.NoError(t, err)
		defer v.Close()

		_, err = v.Watch(ctx)
		require.Error(t, err)
		assert.Equal(t, gcerrors.NotFound, gcerrors.Code(err))
	})

	t.Run("key created after watch begins", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), watchTimeout)
		defer cancel()

		key := "config/created/late"
		v, err := consulvar.OpenVariable(client, key, &consulvar.Options{Decoder: runtimevar.StringDecoder})
		require.NoError(t, err)
		defer v.Close()

		_, err = v.Watch(ctx)
		require.Error(t, err)
		require.Equal(t, gcerrors.NotFound, gcerrors.Code(err))

		put(t, client, key, "now-here")

		snap, err := v.Watch(ctx)
		require.NoError(t, err)
		assert.Equal(t, "now-here", snap.Value.(string))
	})

	t.Run("As exposes the underlying KVPair and QueryMeta", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), watchTimeout)
		defer cancel()

		key := "config/with/flags"
		put(t, client, key, "payload")

		v, err := consulvar.OpenVariable(client, key, nil)
		require.NoError(t, err)
		defer v.Close()

		snap, err := v.Watch(ctx)
		require.NoError(t, err)

		var pair *api.KVPair
		require.True(t, snap.As(&pair))
		require.NotNil(t, pair)
		assert.Equal(t, key, pair.Key)
		assert.Equal(t, []byte("payload"), pair.Value)

		var meta *api.QueryMeta
		require.True(t, snap.As(&meta))
		require.NotNil(t, meta)
		assert.NotZero(t, meta.RequestTime)
		assert.NotZero(t, meta.LastIndex)
	})

	t.Run("URL opener resolves via CONSUL_HTTP_ADDR", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), watchTimeout)
		defer cancel()

		key := "config/url/key"
		put(t, client, key, "from-url")

		t.Setenv("CONSUL_HTTP_ADDR", addr)

		v, err := runtimevar.OpenVariable(ctx, "consul://"+key+"?decoder=string")
		require.NoError(t, err)
		defer v.Close()

		snap, err := v.Watch(ctx)
		require.NoError(t, err)
		assert.Equal(t, "from-url", snap.Value.(string))
	})
}
