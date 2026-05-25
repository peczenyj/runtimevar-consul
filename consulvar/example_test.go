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

// ExampleOptions demonstrates using advanced options like ACL tokens and
// strong consistency.
func ExampleOptions() {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	// Use a pre-configured decoder for JSON data.
	var myConfig struct {
		APIKey string `json:"api_key"`
	}
	decoder := runtimevar.NewDecoder(&myConfig, runtimevar.JSONDecode)

	v, err := consulvar.OpenVariable(client, "app/config", &consulvar.Options{
		Decoder:           decoder,
		Token:             "my-secret-token",
		RequireConsistent: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	_ = v
}

// ExampleVariable_ErrorAs demonstrates how to extract the underlying
// Consul StatusError for detailed error inspection.
func ExampleVariable_ErrorAs() {
	ctx := context.Background()
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	v, err := consulvar.OpenVariable(client, "restricted/key", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	_, err = v.Latest(ctx)
	if err != nil {
		var se api.StatusError
		if v.ErrorAs(err, &se) {
			log.Printf("Consul error %d: %s", se.Code, se.Body)
		}
	}
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
