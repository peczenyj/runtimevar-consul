package consulvar

import (
	"context"
	"net/url"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"gocloud.dev/runtimevar"
)

func TestKeyFromURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"host and path", "consul://services/auth/db_url", "services/auth/db_url"},
		{"host only", "consul://my-key", "my-key"},
		{"trailing slash trimmed", "consul://a/b/", "a/b"},
		{"query ignored", "consul://a/b?decoder=string", "a/b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.want, keyFromURL(u))
		})
	}
}

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
		{name: "datacenter, namespace, wait_time", raw: "consul://a/b?datacenter=dc1&namespace=team&wait_time=30s"},
		{name: "invalid decoder", raw: "consul://a/b?decoder=nope", wantErr: "unsupported decoder"},
		{name: "invalid wait_time", raw: "consul://a/b?wait_time=soon", wantErr: "invalid wait_time"},
		{name: "unknown param", raw: "consul://a/b?bogus=1", wantErr: `invalid query parameter "bogus"`},
	}

	o := &URLOpener{}
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

// TestOpenVariableURL_SchemeRegistered checks that init() registered the
// "consul" scheme on the default mux so runtimevar.OpenVariable resolves it.
func TestOpenVariableURL_SchemeRegistered(t *testing.T) {
	v, err := runtimevar.OpenVariable(context.Background(), "consul://some/key?decoder=string")
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.NoError(t, v.Close())
}
