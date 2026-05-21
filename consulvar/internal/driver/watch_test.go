package driver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

func TestWatchVariable_KeyAppears(t *testing.T) {
	f := newFakeConsul(t)
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	s1, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s1)
	_, err := s1.Value()
	require.Error(t, err)

	f.SetValue([]byte("appeared"))
	s2, _ := w.WatchVariable(context.Background(), s1)
	require.NotNil(t, s2)
	v, err := s2.Value()
	require.NoError(t, err)
	assert.Equal(t, "appeared", v)
}

func TestWatchVariable_DecodeError(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("not-json"))

	var dst map[string]any
	dec := runtimevar.NewDecoder(&dst, runtimevar.JSONDecode)
	w := NewWatcher(f.client(t), "k", Config{Decoder: dec})

	s, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s)
	_, err := s.Value()
	require.Error(t, err)

	var je *json.SyntaxError
	require.True(t, errors.As(err, &je), "decode error should unwrap to *json.SyntaxError")
}

func TestWatchVariable_TransportErrorThenRecover(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("v1"))
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	f.FailNext(http.StatusInternalServerError)
	s1, wait1 := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s1)
	_, err := s1.Value()
	require.Error(t, err)
	assert.Equal(t, time.Second, wait1)

	s2, wait2 := w.WatchVariable(context.Background(), s1)
	require.NotNil(t, s2)
	v, err := s2.Value()
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	assert.Equal(t, time.Duration(0), wait2)

	f.FailNext(http.StatusInternalServerError)
	_, wait3 := w.WatchVariable(context.Background(), s2)
	assert.Equal(t, time.Second, wait3)
}

func TestWatchVariable_ContextCancel(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("v1"))
	w := NewWatcher(f.client(t), "k", Config{
		Decoder:  runtimevar.StringDecoder,
		WaitTime: 5 * time.Second,
	})

	s1, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s1)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = w.WatchVariable(ctx, s1)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("WatchVariable did not return after context cancel")
	}
}

func TestWatchVariable_IndexResetEmitsState(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("v1"))
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	prev := &state{val: "old", modifyIndex: 100, updated: time.Now()}

	s, _ := w.WatchVariable(context.Background(), prev)
	require.NotNil(t, s, "expected a fresh state after cluster index reset")
	v, err := s.Value()
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
}
