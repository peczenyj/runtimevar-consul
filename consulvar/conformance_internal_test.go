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
	closer func()
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

func (h *harness) Close() {
	if h.closer != nil {
		h.closer()
	}
}

func (h *harness) Mutable() bool { return true }

func TestConformance(t *testing.T) {
	newHarness := func(t *testing.T) (drivertest.Harness, error) {
		ctx := context.Background()
		logger := log.TestLogger(t)

		container, err := testcontainer_consul.Run(
			ctx,
			testcontainer_consul.DefaultBaseImage,
			testcontainers.WithLogger(logger),
		)
		if err != nil {
			return nil, err
		}

		addr, err := container.ApiEndpoint(ctx)
		if err != nil {
			_ = container.Terminate(ctx)
			return nil, err
		}

		cfg := api.DefaultConfig()
		cfg.Address = addr
		client, err := api.NewClient(cfg)
		if err != nil {
			_ = container.Terminate(ctx)
			return nil, err
		}

		return &harness{
			client: client,
			closer: func() {
				_ = container.Terminate(context.Background())
			},
		}, nil
	}

	drivertest.RunConformanceTests(t, newHarness, nil)
}
