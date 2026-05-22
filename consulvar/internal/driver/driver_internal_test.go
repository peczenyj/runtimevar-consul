package driver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/gcerrors"
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

func TestState_ValueAndUpdateTime(t *testing.T) {
	now := time.Now()
	s := &state{val: "hello", updated: now}
	v, err := s.Value()
	require.NoError(t, err)
	assert.Equal(t, "hello", v)
	assert.Equal(t, now, s.UpdateTime())
}

func TestState_ValueReturnsErr(t *testing.T) {
	boom := errors.New("boom")
	s := &state{err: boom}
	_, err := s.Value()
	assert.ErrorIs(t, err, boom)
}

func TestState_As_KVPair(t *testing.T) {
	kv := &api.KVPair{Key: "k", Value: []byte("v"), ModifyIndex: 7}
	s := &state{raw: kv}
	var got *api.KVPair
	ok := s.As(&got)
	assert.True(t, ok)
	assert.Same(t, kv, got)
}

func TestState_As_UnsupportedType(t *testing.T) {
	s := &state{raw: &api.KVPair{}}
	var got string
	ok := s.As(&got)
	assert.False(t, ok)
}

func TestErrorAs_AlwaysFalse(t *testing.T) {
	w := newTestWatcher(t)
	var target *api.KVPair
	assert.False(t, w.ErrorAs(errors.New("anything"), &target))
}

func TestBackoff_Table(t *testing.T) {
	cases := []struct {
		name     string
		failures int
		want     time.Duration
	}{
		{"none", 0, 0},
		{"negative", -1, 0},
		{"first", 1, time.Second},
		{"second", 2, 2 * time.Second},
		{"third", 3, 4 * time.Second},
		{"capped", 6, 30 * time.Second},
		{"overflow", 64, 30 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, backoff(tc.failures))
		})
	}
}

func TestNextBlockingIndex(t *testing.T) {
	cases := []struct {
		name string
		idx  uint64
		want uint64
	}{
		{"zero floored to one", 0, 1},
		{"one unchanged", 1, 1},
		{"non-zero unchanged", 42, 42},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, nextBlockingIndex(tc.idx))
		})
	}
}

func TestErrorCode_Table(t *testing.T) {
	w := newTestWatcher(t)
	cases := []struct {
		name string
		err  error
		want gcerrors.ErrorCode
	}{
		{"nil", nil, gcerrors.OK},
		{"canceled", context.Canceled, gcerrors.Canceled},
		{"deadline", context.DeadlineExceeded, gcerrors.DeadlineExceeded},
		{"notfound", errKeyNotFound, gcerrors.NotFound},
		{"notfound-wrapped", errors.Join(errors.New("consulvar: key \"x\""), errKeyNotFound), gcerrors.NotFound},
		{"other", errors.New("boom"), gcerrors.Unknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, w.ErrorCode(tc.err))
		})
	}
}
