package consulvar_test

import (
	"context"
	"log"

	"github.com/hashicorp/consul/api"
	"gocloud.dev/runtimevar"

	"github.com/peczenyj/runtimevar-consul/consulvar"
)

// ExampleOpenVariable watches a key with a Consul client you construct and own.
func ExampleOpenVariable() {
	ctx := context.Background()

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	v, err := consulvar.OpenVariable(client, "services/auth/db_url", &consulvar.Options{
		Decoder: runtimevar.StringDecoder,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	snap, err := v.Latest(ctx)
	if err != nil {
		log.Print(err)
		return
	}
	log.Printf("db_url = %s", snap.Value.(string))
}

// Example_openVariableFromURL opens the same key through the URL scheme. A
// blank import of consulvar registers the "consul" scheme; connection settings
// come from the standard Consul environment variables.
func Example_openVariableFromURL() {
	// import _ "github.com/peczenyj/runtimevar-consul/consulvar"

	ctx := context.Background()

	v, err := runtimevar.OpenVariable(ctx, "consul://services/auth/db_url?decoder=string")
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	snap, err := v.Latest(ctx)
	if err != nil {
		log.Print(err)
		return
	}
	log.Printf("db_url = %s", snap.Value.(string))
}
