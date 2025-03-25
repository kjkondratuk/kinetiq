//go:build wasip1

package main

import (
	"context"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"log"
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

	// Log the inputs to the processor to show the module is doing something and that output from the module routes
	// output properly.
	log.Printf("Processing request: %s - %s : %s - %s", "key", request.Key, "value", string(request.Value))

	// Format and return the response, simply echoing the source message into the destination
	resp := v1.ProcessResponse{
		Key:     request.Key,
		Value:   request.Value,
		Headers: request.Headers,
	}
	return &resp, nil
}
