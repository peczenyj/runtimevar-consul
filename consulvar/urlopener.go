package consulvar

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"gocloud.dev/runtimevar"
)

// Scheme is the URL scheme consulvar registers its URLOpener under.
const Scheme = "consul"

func init() {
	runtimevar.DefaultURLMux().RegisterVariable(Scheme, &URLOpener{})
}

// URLOpener opens Consul KV URLs like "consul://services/auth/db_url?decoder=string".
// See the package documentation for the supported query parameters.
type URLOpener struct {
	// Client, if set, is reused for every opened URL. When nil, each call builds
	// a client from api.DefaultConfig() (the standard Consul environment
	// variables). The Client is never closed by the returned Variable.
	Client *api.Client

	// Decoder is the fallback decoder used when the URL has no "decoder" query
	// parameter. When nil, runtimevar.BytesDecoder is used.
	Decoder *runtimevar.Decoder

	// Options holds defaults for fields not overridden by the URL.
	Options Options

	mu         sync.Mutex
	lazyClient *api.Client

	// opener is api.NewClient by default, but can be overridden in tests.
	opener func(*api.Config) (*api.Client, error)
}

// OpenVariableURL opens a runtimevar.Variable for the given Consul KV URL.
func (o *URLOpener) OpenVariableURL(ctx context.Context, u *url.URL) (*runtimevar.Variable, error) {
	q := u.Query()

	decoderName := q.Get("decoder")
	q.Del("decoder")
	decoder, err := runtimevar.DecoderByName(ctx, decoderName, o.decoderOrDefault())
	if err != nil {
		return nil, fmt.Errorf("open variable %q: %w", u, err)
	}

	opts := o.Options
	opts.Decoder = decoder
	if dc := q.Get("datacenter"); dc != "" {
		opts.Datacenter = dc
		q.Del("datacenter")
	}
	if ns := q.Get("namespace"); ns != "" {
		opts.Namespace = ns
		q.Del("namespace")
	}
	if stale := q.Get("allow_stale"); stale != "" {
		b, err := strconv.ParseBool(stale)
		if err != nil {
			return nil, fmt.Errorf("open variable %q: invalid allow_stale: %w", u, err)
		}
		opts.AllowStale = b
		q.Del("allow_stale")
	}
	if wt := q.Get("wait_time"); wt != "" {
		d, err := time.ParseDuration(wt)
		if err != nil {
			return nil, fmt.Errorf("open variable %q: invalid wait_time: %w", u, err)
		}
		opts.WaitTime = d
		q.Del("wait_time")
	}
	for param := range q {
		return nil, fmt.Errorf("open variable %q: invalid query parameter %q", u, param)
	}

	key := keyFromURL(u)

	client := o.Client
	if client == nil {
		o.mu.Lock()
		if o.lazyClient == nil {
			opener := o.opener
			if opener == nil {
				opener = api.NewClient
			}
			c, err := opener(api.DefaultConfig())
			if err != nil {
				o.mu.Unlock()
				return nil, fmt.Errorf("open variable %q: %w", u, err)
			}
			o.lazyClient = c
		}
		client = o.lazyClient
		o.mu.Unlock()
	}

	return OpenVariable(client, key, &opts)
}

func (o *URLOpener) decoderOrDefault() *runtimevar.Decoder {
	if o.Decoder != nil {
		return o.Decoder
	}
	return runtimevar.BytesDecoder
}

// keyFromURL joins the URL host and path into a Consul KV key. The leading
// slash from the path is dropped, so "consul://services/auth/db_url" yields
// "services/auth/db_url".
func keyFromURL(u *url.URL) string {
	return path.Join(u.Host, u.Path)
}
