package consulvar

import (
	"context"
	"net/url"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestURLOpener_ClientCreationFailure(t *testing.T) {
	o := &URLOpener{
		opener: func(*api.Config) (*api.Client, error) {
			return nil, assert.AnError
		},
	}
	u, _ := url.Parse("consul://key")
	v, err := o.OpenVariableURL(context.Background(), u)
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, v)
}
