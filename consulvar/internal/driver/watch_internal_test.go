package driver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gocloud.dev/gcerrors"
	"gocloud.dev/runtimevar"

	consulmock "github.com/peczenyj/runtimevar-consul/mocks/github.com/peczenyj/runtimevar-consul/consulvar/consulapi"
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

func TestWatchVariable_NotFound_UnrelatedIndexChange_Suppressed(t *testing.T) {
	f := newFakeConsul(t)
	// Key exists then is removed, so the cluster index is non-zero while the
	// key is absent.
	f.SetValue([]byte("v1"))
	f.DeleteKey()
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	s1, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s1)
	_, err := s1.Value()
	require.Error(t, err)
	assert.Equal(t, gcerrors.NotFound, w.ErrorCode(err))

	// An unrelated KV write bumps the cluster index while our key stays absent.
	// The watcher must not re-emit the identical NotFound error.
	f.DeleteKey()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	s2, _ := w.WatchVariable(ctx, s1)
	assert.Nil(t, s2, "still-absent key must not re-emit NotFound on an unrelated index bump")
}

func TestWatchVariable_NotFound_ZeroIndexDoesNotBusyLoop(t *testing.T) {
	f := newFakeConsul(t)
	// A never-existing key: the fake returns X-Consul-Index: 0 on every
	// response. A WaitIndex of 0 is non-blocking, so without the index floor
	// the poll loop spins as fast as HTTP round-trips allow.
	w := NewWatcher(f.client(t), "missing", Config{Decoder: runtimevar.StringDecoder})

	s1, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	s2, _ := w.WatchVariable(ctx, s1)
	assert.Nil(t, s2)

	// With the index floored to 1, each poll blocks (~50ms in the fake), so the
	// 200ms window yields only a handful of requests. A busy loop would issue
	// hundreds to thousands.
	reqs := f.Requests()
	assert.Less(t, len(reqs), 20,
		"zero-index NotFound must block between polls, not busy-loop (got %d requests)", len(reqs))
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

func TestWatchVariable_PermissionDenied(t *testing.T) {
	f := newFakeConsul(t)
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	// Simulate a 403 Forbidden
	f.FailNext(http.StatusForbidden)
	s, _ := w.WatchVariable(context.Background(), nil)

	require.NotNil(t, s)
	_, err := s.Value()
	require.Error(t, err)
	assert.Equal(t, gcerrors.PermissionDenied, w.ErrorCode(err))

	var se api.StatusError
	assert.True(t, w.ErrorAs(err, &se))
	assert.Equal(t, http.StatusForbidden, se.Code)
}

func TestWatchVariable_ResourceExhausted(t *testing.T) {
	f := newFakeConsul(t)
	w := NewWatcher(f.client(t), "k", Config{Decoder: runtimevar.StringDecoder})

	// Simulate a 429 Too Many Requests
	f.FailNext(http.StatusTooManyRequests)
	s, _ := w.WatchVariable(context.Background(), nil)

	require.NotNil(t, s)
	_, err := s.Value()
	require.Error(t, err)
	assert.Equal(t, gcerrors.ResourceExhausted, w.ErrorCode(err))
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

func TestWatchVariable_PropagatesQueryOptions(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("v1"))
	w := NewWatcher(f.client(t), "k", Config{
		Decoder:    runtimevar.StringDecoder,
		Datacenter: "dc1",
		Namespace:  "team-a",
		AllowStale: true,
		WaitTime:   30 * time.Second,
	})

	s, _ := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s)

	reqs := f.Requests()
	require.NotEmpty(t, reqs)
	last := reqs[len(reqs)-1]
	assert.Equal(t, "dc1", last.DC, "datacenter should reach the Consul query")
	assert.Equal(t, "team-a", last.NS, "namespace should reach the Consul query")
	assert.Equal(t, "true", last.Stale, "allow_stale should reach the Consul query")
	assert.NotEmpty(t, last.Wait, "wait_time should reach the Consul query")
}

func TestWatchVariable_Mocked(t *testing.T) {
	mk := consulmock.NewConsulKV(t)
	mc := consulmock.NewConsulClient(t)
	w := &Watcher{
		client:  mc,
		key:     "k",
		decoder: runtimevar.StringDecoder,
	}

	mc.EXPECT().KV().Return(mk)

	expectedValue := []byte("mocked-value")
	// Verify that the WaitIndex is passed correctly through QueryOptions
	mk.EXPECT().Get("k", mock.MatchedBy(func(q *api.QueryOptions) bool {
		return q.WaitIndex == 123
	})).Return(
		&api.KVPair{Key: "k", Value: expectedValue, ModifyIndex: 456},
		&api.QueryMeta{LastIndex: 456},
		nil,
	)

	prev := &state{modifyIndex: 123}
	s, _ := w.WatchVariable(context.Background(), prev)
	require.NotNil(t, s)
	v, err := s.Value()
	require.NoError(t, err)
	assert.Equal(t, "mocked-value", v)

	var meta *api.QueryMeta
	require.True(t, s.As(&meta))
	require.NotNil(t, meta)
	assert.Equal(t, uint64(456), meta.LastIndex)
}

func TestWatchVariable_Mocked_TransportError(t *testing.T) {
	mk := consulmock.NewConsulKV(t)
	mc := consulmock.NewConsulClient(t)
	w := &Watcher{
		client:  mc,
		key:     "k",
		decoder: runtimevar.StringDecoder,
	}

	mc.EXPECT().KV().Return(mk)

	boom := errors.New("network failure")
	mk.EXPECT().Get("k", mock.Anything).Return(nil, nil, boom)

	s, wait := w.WatchVariable(context.Background(), nil)
	require.NotNil(t, s)
	_, err := s.Value()
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, time.Second, wait)
}

// https://github.com/peczenyj/runtimevar-consul/issues/27
// A successful poll must reset the consecutive-failure counter, so a later
// transport error restarts the backoff at 1s rather than escalating.
func TestWatchVariable_BackoffResetsAfterSuccessfulPoll_Issue27(t *testing.T) {
	mk := consulmock.NewConsulKV(t)
	mc := consulmock.NewConsulClient(t)
	w := &Watcher{
		client:  mc,
		key:     "k",
		decoder: runtimevar.StringDecoder,
	}

	mc.EXPECT().KV().Return(mk)

	boom := errors.New("network failure")
	// Call A: transport error -> failures becomes 1, backoff 1s.
	mk.EXPECT().Get("k", mock.Anything).Return(nil, nil, boom).Once()
	// Call B, first poll: a successful but unchanged read (ModifyIndex == prev).
	mk.EXPECT().Get("k", mock.Anything).Return(
		&api.KVPair{Key: "k", Value: []byte("v"), ModifyIndex: 5},
		&api.QueryMeta{LastIndex: 5},
		nil,
	).Once()
	// Call B, second poll: transport error again.
	mk.EXPECT().Get("k", mock.Anything).Return(nil, nil, boom).Once()

	_, waitA := w.WatchVariable(context.Background(), nil)
	require.Equal(t, time.Second, waitA)

	_, waitB := w.WatchVariable(context.Background(), &state{modifyIndex: 5})
	assert.Equal(t, time.Second, waitB,
		"a successful poll must reset backoff; expected 1s, the schedule must not escalate to 2s")
}

func TestWatchVariable_Mocked_IndexReset(t *testing.T) {
	mk := consulmock.NewConsulKV(t)
	mc := consulmock.NewConsulClient(t)
	w := &Watcher{
		client:  mc,
		key:     "k",
		decoder: runtimevar.StringDecoder,
	}

	mc.EXPECT().KV().Return(mk)

	// WaitIndex 1000, but Consul returns LastIndex 5. This is an index reset.
	// Watcher should realize effectivePrev = 0 and return the current value.
	expectedValue := []byte("reset-value")
	mk.EXPECT().Get("k", mock.MatchedBy(func(q *api.QueryOptions) bool {
		return q.WaitIndex == 1000
	})).Return(
		&api.KVPair{Key: "k", Value: expectedValue, ModifyIndex: 5},
		&api.QueryMeta{LastIndex: 5},
		nil,
	)

	prev := &state{modifyIndex: 1000}
	s, _ := w.WatchVariable(context.Background(), prev)
	require.NotNil(t, s)
	v, err := s.Value()
	require.NoError(t, err)
	assert.Equal(t, "reset-value", v)
}
