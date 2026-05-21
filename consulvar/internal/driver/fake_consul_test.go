package driver

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConsul is a minimal in-memory stand-in for Consul's /v1/kv endpoint.
// It supports blocking queries via WaitIndex; calling SetValue / DeleteKey
// publishes a new state to any blocked waiter.
type fakeConsul struct {
	srv *httptest.Server

	mu        sync.Mutex
	cond      *sync.Cond
	value     []byte
	exists    bool
	index     uint64
	requests  []recordedRequest
	httpError int

	forcedLastIndex uint64
}

type recordedRequest struct {
	URL       *url.URL
	WaitIndex string
	Wait      string
	DC        string
	NS        string
}

func newFakeConsul(t *testing.T) *fakeConsul {
	t.Helper()
	f := &fakeConsul{}
	f.cond = sync.NewCond(&f.mu)
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeConsul) client(t *testing.T) *api.Client {
	t.Helper()
	cfg := api.DefaultConfig()
	cfg.Address = strings.TrimPrefix(f.srv.URL, "http://")
	c, err := api.NewClient(cfg)
	require.NoError(t, err)
	return c
}

func (f *fakeConsul) SetValue(v []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.value = v
	f.exists = true
	f.index++
	f.cond.Broadcast()
}

func (f *fakeConsul) DeleteKey() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.value = nil
	f.exists = false
	f.index++
	f.cond.Broadcast()
}

func (f *fakeConsul) FailNext(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.httpError = status
	f.cond.Broadcast()
}

func (f *fakeConsul) ForceLastIndex(idx uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forcedLastIndex = idx
}

func (f *fakeConsul) Requests() []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedRequest, len(f.requests))
	copy(out, f.requests)
	return out
}

func (f *fakeConsul) handle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rec := recordedRequest{
		URL:       r.URL,
		WaitIndex: q.Get("index"),
		Wait:      q.Get("wait"),
		DC:        q.Get("dc"),
		NS:        q.Get("ns"),
	}

	f.mu.Lock()
	f.requests = append(f.requests, rec)

	if f.httpError != 0 {
		status := f.httpError
		f.httpError = 0
		f.mu.Unlock()
		http.Error(w, http.StatusText(status), status)
		return
	}

	// Observe request-context cancellation so blocked waiters return promptly
	// when the client closes the connection.
	ctx := r.Context()
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			f.mu.Lock()
			f.cond.Broadcast()
			f.mu.Unlock()
		case <-done:
		}
	}()
	defer close(done)

	clientWaitIndex, _ := strconv.ParseUint(q.Get("index"), 10, 64)
	if clientWaitIndex > 0 && f.index <= clientWaitIndex {
		waitDur := parseDurationOrZero(q.Get("wait"))
		if waitDur <= 0 {
			waitDur = 50 * time.Millisecond
		}
		timer := time.AfterFunc(waitDur, func() {
			f.mu.Lock()
			f.cond.Broadcast()
			f.mu.Unlock()
		})
		deadline := time.Now().Add(waitDur)
		for f.index <= clientWaitIndex && time.Now().Before(deadline) && f.httpError == 0 && ctx.Err() == nil {
			f.cond.Wait()
		}
		timer.Stop()
		if ctx.Err() != nil {
			f.mu.Unlock()
			return
		}
	}

	lastIndex := f.index
	if f.forcedLastIndex != 0 {
		lastIndex = f.forcedLastIndex
		f.forcedLastIndex = 0
	}
	exists := f.exists
	value := append([]byte(nil), f.value...)
	idx := f.index
	httpErr := f.httpError
	if httpErr != 0 {
		f.httpError = 0
	}
	f.mu.Unlock()

	w.Header().Set("X-Consul-Index", strconv.FormatUint(lastIndex, 10))
	if httpErr != 0 {
		http.Error(w, http.StatusText(httpErr), httpErr)
		return
	}
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	pair := struct {
		Key         string `json:"Key"`
		CreateIndex uint64 `json:"CreateIndex"`
		ModifyIndex uint64 `json:"ModifyIndex"`
		LockIndex   uint64 `json:"LockIndex"`
		Flags       uint64 `json:"Flags"`
		Value       string `json:"Value"`
	}{
		Key:         strings.TrimPrefix(r.URL.Path, "/v1/kv/"),
		CreateIndex: 1,
		ModifyIndex: idx,
		Value:       base64.StdEncoding.EncodeToString(value),
	}
	body, _ := json.Marshal(pair)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte("["))
	_, _ = w.Write(body)
	_, _ = w.Write([]byte("]"))
}

func parseDurationOrZero(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func TestFakeConsul_ServesValue(t *testing.T) {
	f := newFakeConsul(t)
	f.SetValue([]byte("hello"))
	c := f.client(t)

	kv, meta, err := c.KV().Get("any-key", nil)
	require.NoError(t, err)
	require.NotNil(t, kv)
	assert.Equal(t, []byte("hello"), kv.Value)
	assert.Equal(t, uint64(1), kv.ModifyIndex)
	assert.Equal(t, uint64(1), meta.LastIndex)
}
