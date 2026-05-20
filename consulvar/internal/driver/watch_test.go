package driver

import (
	"context"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"github.com/hashicorp/consul/api"
	"gocloud.dev/runtimevar"
)

func TestWatchVariable_InitialValue(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("hello"))
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	s, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s)

	v, err := s.Value()
	require.NoError(t, err)
	assert.Equal(t, "hello", v)

	var kv *api.KVPair
	require.True(t, s.As(&kv))
	assert.Equal(t, uint64(1), kv.ModifyIndex)
}
