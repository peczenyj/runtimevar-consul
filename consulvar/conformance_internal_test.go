//go:build integration

package consulvar

import (
	"context"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	testcontainer_consul "github.com/testcontainers/testcontainers-go/modules/consul"
	"gocloud.dev/runtimevar"
	rvdriver "gocloud.dev/runtimevar/driver"
	"gocloud.dev/runtimevar/drivertest"

	"github.com/peczenyj/runtimevar-consul/consulvar/internal/driver"
)

type harness struct {
	client *api.Client
}

func (h *harness) MakeWatcher(ctx context.Context, name string, decoder *runtimevar.Decoder) (rvdriver.Watcher, error) {
	return driver.NewWatcher(h.client, name, driver.Config{Decoder: decoder}), nil
}

func (h *harness) CreateVariable(ctx context.Context, name string, val []byte) error {
	return h.UpdateVariable(ctx, name, val)
}

func (h *harness) UpdateVariable(ctx context.Context, name string, val []byte) error {
	_, err := h.client.KV().Put(&api.KVPair{Key: name, Value: val}, nil)
	return err
}

func (h *harness) DeleteVariable(ctx context.Context, name string) error {
	_, err := h.client.KV().Delete(name, nil)
	return err
}

func (h *harness) Close() {}

func (h *harness) Mutable() bool { return true }

func TestConformance(t *testing.T) {
	ctx := context.Background()
	logger := log.TestLogger(t)

	// One Consul container is shared across every conformance sub-test. The
	// suite uses a distinct variable name per test, so a single KV store is
	// sufficient and avoids the cost of booting a container per harness.
	container, err := testcontainer_consul.Run(
		ctx,
		testcontainer_consul.DefaultBaseImage,
		testcontainers.WithLogger(logger),
	)
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("start consul container: %v", err)
	}

	addr, err := container.ApiEndpoint(ctx)
	if err != nil {
		t.Fatalf("consul api endpoint: %v", err)
	}

	cfg := api.DefaultConfig()
	cfg.Address = addr
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("consul client: %v", err)
	}

	newHarness := func(t *testing.T) (drivertest.Harness, error) {
		return &harness{client: client}, nil
	}

	drivertest.RunConformanceTests(t, newHarness, nil)
}
