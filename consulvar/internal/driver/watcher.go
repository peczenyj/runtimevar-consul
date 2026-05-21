package driver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gocloud.dev/runtimevar/driver"
)

// WatchVariable performs one Consul blocking query and returns the resulting
// driver.State. See the package-level comment for the full state machine.
func (w *Watcher) WatchVariable(ctx context.Context, prev driver.State) (driver.State, time.Duration) {
	var waitIndex uint64
	if p, ok := prev.(*state); ok {
		waitIndex = p.modifyIndex
	}

	qo := w.baseQuery
	qo.WaitIndex = waitIndex
	kv, meta, err := w.client.KV().Get(w.key, qo.WithContext(ctx))

	now := time.Now()
	if ctx.Err() != nil {
		return nil, 0
	}
	if err != nil {
		w.failures++
		return &state{err: err, updated: now}, backoff(w.failures)
	}

	// Index-reset safety: per Consul docs, if meta.LastIndex < waitIndex the
	// cluster's index has reset and the comparison is invalid.
	//
	// meta is always non-nil here: api.Client.KV().Get returns a populated
	// *QueryMeta even on 404 (the X-Consul-Index header is still present).
	effectivePrev := waitIndex
	if meta.LastIndex < waitIndex {
		effectivePrev = 0
	}

	if kv == nil {
		nfErr := fmt.Errorf("consulvar: key %q: %w", w.key, errKeyNotFound)
		if p, ok := prev.(*state); ok && errors.Is(p.err, errKeyNotFound) && effectivePrev == meta.LastIndex {
			return nil, 0
		}
		w.failures = 0
		return &state{err: nfErr, modifyIndex: meta.LastIndex, updated: now}, 0
	}

	if effectivePrev == kv.ModifyIndex {
		w.failures = 0
		return nil, 0
	}

	v, decErr := w.decoder.Decode(ctx, kv.Value)
	w.failures = 0
	if decErr != nil {
		return &state{err: decErr, modifyIndex: kv.ModifyIndex, updated: now}, 0
	}
	return &state{val: v, raw: kv, modifyIndex: kv.ModifyIndex, updated: now}, 0
}

// backoff returns the wait time after `failures` consecutive transport errors.
// 1s, 2s, 4s, … capped at 30s.
func backoff(failures int) time.Duration {
	if failures <= 0 {
		return 0
	}
	d := time.Second << (failures - 1)
	if d > 30*time.Second || d <= 0 {
		return 30 * time.Second
	}
	return d
}
