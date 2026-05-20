package driver

import (
	"context"
	"testing"
	"time"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"github.com/hashicorp/consul/api"
	"gocloud.dev/gcerrors"
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

func TestWatchVariable_DetectsUpdate(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("v1"))
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	s1, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s1)

	f.SetValue([]byte("v2"))
	s2, _ := w.WatchVariable(context.Background(), s1)
	require.NotNil(t, s2)

	v, err := s2.Value()
	require.NoError(t, err)
	assert.Equal(t, "v2", v)
}

func TestWatchVariable_NoChange_Suppressed(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("v1"))
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	s1, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	f.ForceLastIndex(1)
	s2, _ := w.WatchVariable(ctx, s1)
	assert.Nil(t, s2)
}

func TestWatchVariable_KeyMissing_NotFound(t *testing.T) {
	f := newFakeConsul(t)
	w := NewWatcher(f.client(t), "missing", Config{Decoder: runtimevar.StringDecoder})

	s, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s)
	_, err := s.Value()
	require.Error(t, err)
	assert.Equal(t, gcerrors.NotFound, w.ErrorCode(err))
}

func TestWatchVariable_NotFound_StableSameIndex(t *testing.T) {
	f := newFakeConsul(t)
	w := NewWatcher(f.client(t), "missing", Config{Decoder: runtimevar.StringDecoder})

	s1, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	f.ForceLastIndex(0)
	s2, _ := w.WatchVariable(ctx, s1)
	assert.Nil(t, s2)
}
