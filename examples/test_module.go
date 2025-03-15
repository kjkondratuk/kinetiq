//go:build wasip1

package main

import (
	"context"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"log"
)

func main() {}

func init() {
	v1.RegisterModuleService(MyPlugin{HostFunctionsService: v1.NewHostFunctionsService()})
}

type MyPlugin struct {
	v1.HostFunctionsService
}

func (m MyPlugin) Process(ctx context.Context, request *v1.ProcessRequest) (*v1.ProcessResponse, error) {
	log.Printf("Processing SOME OTHER request: %s - %s : %s - %s", "key", request.Key, "value", string(request.Value))
	resp := v1.ProcessResponse{
		Key:     request.Key,
		Value:   request.Value,
		Headers: request.Headers,
	}
	return &resp, nil
}
