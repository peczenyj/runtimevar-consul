package driver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gocloud.dev/runtimevar/driver"
)

// WatchVariable performs Consul blocking queries and returns the resulting
// driver.State. It blocks until the variable changes in a way that yields a
// new value or a different error, or until ctx is done.
func (w *Watcher) WatchVariable(ctx context.Context, prev driver.State) (driver.State, time.Duration) {
	var waitIndex uint64
	var prevErr error
	if p, ok := prev.(*state); ok {
		waitIndex = p.modifyIndex
		prevErr = p.err
	}

	for {
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
		effectivePrev := waitIndex
		if meta.LastIndex < waitIndex {
			effectivePrev = 0
		}

		if kv == nil {
			nfErr := fmt.Errorf("consulvar: key %q: %w", w.key, errKeyNotFound)
			// The key is (still) absent. If we already reported NotFound,
			// suppress re-emitting the identical error even when meta.LastIndex
			// advanced: that index is cluster-wide, so an unrelated KV write
			// bumps it while our key's existence is unchanged. Advance the wait
			// index (ensuring the next query blocks) so the next blocking query
			// waits past the new index.
			if errors.Is(prevErr, errKeyNotFound) {
				waitIndex = nextBlockingIndex(meta.LastIndex)
				continue
			}
			w.failures = 0
			return &state{err: nfErr, modifyIndex: meta.LastIndex, updated: now}, 0
		}

		if effectivePrev == kv.ModifyIndex {
			waitIndex = kv.ModifyIndex
			continue
		}

		v, decErr := w.decoder.Decode(ctx, kv.Value)
		w.failures = 0
		if decErr != nil {
			if prevErr != nil && prevErr.Error() == decErr.Error() {
				waitIndex = kv.ModifyIndex
				continue
			}
			return &state{err: decErr, modifyIndex: kv.ModifyIndex, updated: now}, 0
		}
		return &state{val: v, raw: kv, modifyIndex: kv.ModifyIndex, updated: now}, 0
	}
}

// nextBlockingIndex returns a WaitIndex safe to re-use for the next blocking
// query. Consul's blocking-query protocol requires clients to treat any index
// below 1 as 1: a WaitIndex of 0 makes the query return immediately, which
// would turn this poll loop into a busy spin if a server ever reports
// LastIndex 0 for a missing key.
func nextBlockingIndex(idx uint64) uint64 {
	if idx < 1 {
		return 1
	}
	return idx
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
