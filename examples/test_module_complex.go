//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"log"
	"math/rand"
)

func main() {
	// Empty, but this would be executed after init when the module is loaded
}

func init() {
	v1.RegisterModuleService(MyPlugin{HostFunctionsService: v1.NewHostFunctionsService()})
}

// MyPlugin is a v1.HostFunctionsService because it provides host functions (from the above call to register the
// module) **AND** a v1.ModuleService because it implements the interface in the Process function below.
type MyPlugin struct {
	v1.HostFunctionsService
}

// Process : the MyPlugin implementation of the message processor plugin API.
//
//	inputs:
//	    - ctx - context.Context - a context which can be terminated or cancelled by the server
//	    - request - *v1.ProcessRequest - a record which this WASM module can process
//	outputs:
//	    - *v1.ProcessResponse - the response data to publish to the configured destination
//	    - error - any error that occurred during processing that resulted in the
func (m MyPlugin) Process(ctx context.Context, request *v1.ProcessRequest) (*v1.ProcessResponse, error) {
	log.Printf("Processing BRAND request: %s - %s : %s - %s", "key", request.Key, "value", string(request.Value))

	// Unmarshal a json payload from the source topic, returning an error if it's malformatted
	data := samplePayload{}
	err := json.Unmarshal(request.Value, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sample payload in webassembly module: %w", err)
	}

	// Execute an HTTP get request using the provided host function.  This allows the WASM module to invoke logic on the
	// host and is necessary because the WASM environment cannot run parallel tasks required for things like web clients
	// and servers.  In this case, we perform a simple, unparameterized get request against a RESTful service.
	//
	// **Note** : this url will not actually work and is provided only as an example
	r, _ := m.HttpGet(ctx, &v1.HttpGetRequest{Url: "https://some.url.com/brands"})
	// TODO : need a configuration API to provide/seed values like this, or some ability to load a config file

	// Unmarshal data out of the HTTP response
	brands := UserBrandResponse{}
	err = json.Unmarshal(r.Response, &brands)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal brand response in webassembly module: %w", err)
	}

	// Select a random available brand from the HTTP response, if any are present and combine the source data with it
	var combined combinedResponse
	if len(brands) > 0 {
		selectedBrand := brands[rand.Intn(len(brands))]
		log.Printf("Selected brand: %+v", selectedBrand)
		combined = combinedResponse{
			samplePayload: data,
			Brand:         selectedBrand,
		}
	} else {
		log.Println("No brands available to select.")
		combined = combinedResponse{
			samplePayload: data,
			Brand:         Brand{},
		}
	}

	// Marshal the combined payload back into json so we can send it as a message to the destination
	combinedBytes, _ := json.Marshal(combined)
	log.Printf("Combined Response: %s", combinedBytes)

	// Compose and return the response
	resp := v1.ProcessResponse{
		Key:     request.Key,
		Value:   combinedBytes,
		Headers: request.Headers,
	}
	return &resp, nil
}

type combinedResponse struct {
	samplePayload
	Brand Brand `json:"brand"`
}

type samplePayload struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

type UserBrandResponse []Brand

type Brand struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	RomanceText string `json:"romanceText"`
}
