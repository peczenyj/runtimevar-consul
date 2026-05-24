package consulvar_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/runtimevar"

	"github.com/peczenyj/runtimevar-consul/consulvar"
)

func TestURLOpener_OpenVariableURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{name: "no params", raw: "consul://a/b"},
		{name: "decoder bytes", raw: "consul://a/b?decoder=bytes"},
		{name: "decoder string", raw: "consul://a/b?decoder=string"},
		{name: "decoder jsonmap", raw: "consul://a/b?decoder=jsonmap"},
		{name: "datacenter, namespace, wait_time, allow_stale", raw: "consul://a/b?datacenter=dc1&namespace=team&wait_time=30s&allow_stale=true"},
		{name: "invalid decoder", raw: "consul://a/b?decoder=nope", wantErr: "unsupported decoder"},
		{name: "invalid wait_time", raw: "consul://a/b?wait_time=soon", wantErr: "invalid wait_time"},
		{name: "invalid allow_stale", raw: "consul://a/b?allow_stale=notbool", wantErr: "invalid allow_stale"},
		{name: "unknown param", raw: "consul://a/b?bogus=1", wantErr: `invalid query parameter "bogus"`},
	}

	o := &consulvar.URLOpener{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.raw)
			require.NoError(t, err)

			v, err := o.OpenVariableURL(context.Background(), u)
			if tt.wantErr != "" {
				assert.Nil(t, v)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.NoError(t, v.Close())
		})
	}
}

func TestURLOpener_DecryptPrefix(t *testing.T) {
	// decrypt+ prefix is handled by runtimevar.DecoderByName.
	// It will fail if no secrets.Keeper is in the context, but we want to
	// ensure our URLOpener passes it through without erroring on the name itself.
	o := &consulvar.URLOpener{}
	u, _ := url.Parse("consul://key?decoder=decrypt+string")
	v, err := o.OpenVariableURL(context.Background(), u)

	// It's expected to fail because we haven't set up a secrets.Keeper,
	// but the error should come from runtimevar, not our parameter validation.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RUNTIMEVAR_KEEPER_URL")
	assert.Nil(t, v)
}

func TestURLOpener_WithFallbackDecoder(t *testing.T) {
	// Test the case where URLOpener has a pre-configured Decoder (improving coverage).
	o := &consulvar.URLOpener{
		Decoder: runtimevar.StringDecoder,
	}
	u, _ := url.Parse("consul://key") // No decoder param, should use fallback
	v, err := o.OpenVariableURL(context.Background(), u)
	require.NoError(t, err)
	require.NotNil(t, v)
	v.Close()
}

// TestOpenVariableURL_SchemeRegistered checks that init() registered the
// "consul" scheme on the default mux so runtimevar.OpenVariable resolves it.
func TestOpenVariableURL_SchemeRegistered(t *testing.T) {
	v, err := runtimevar.OpenVariable(context.Background(), "consul://some/key?decoder=string")
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.NoError(t, v.Close())
}
