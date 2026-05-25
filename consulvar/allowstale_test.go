package consulvar_test

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/runtimevar"

	"github.com/peczenyj/runtimevar-consul/consulvar"
)

// staleRecordingServer serves a single KV pair and records whether any request
// carried the Consul "stale" query parameter.
func staleRecordingServer(t *testing.T) (client *api.Client, sawStale *atomic.Bool) {
	t.Helper()
	sawStale = &atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("stale") {
			sawStale.Store(true)
		}
		w.Header().Set("X-Consul-Index", "1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"Key":"k","CreateIndex":1,"ModifyIndex":1,"Value":"`+
			base64.StdEncoding.EncodeToString([]byte("v"))+`"}]`)
	}))
	t.Cleanup(srv.Close)

	cfg := api.DefaultConfig()
	cfg.Address = strings.TrimPrefix(srv.URL, "http://")
	c, err := api.NewClient(cfg)
	require.NoError(t, err)
	return c, sawStale
}

// https://github.com/peczenyj/runtimevar-consul/issues/26
// AllowStale set on Options must reach the Consul query as ?stale.
func TestOpenVariable_ForwardsAllowStale_Issue26(t *testing.T) {
	client, sawStale := staleRecordingServer(t)

	v, err := consulvar.OpenVariable(client, "k", &consulvar.Options{
		Decoder:    runtimevar.StringDecoder,
		AllowStale: true,
	})
	require.NoError(t, err)
	defer v.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	snap, err := v.Watch(ctx)
	require.NoError(t, err)
	assert.Equal(t, "v", snap.Value.(string))
	assert.True(t, sawStale.Load(), "Options.AllowStale must reach the Consul query as ?stale")
}
