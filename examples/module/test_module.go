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
	log.Printf("Processing HI SUSHI request: %s - %s : %s - %s", "key", request.Key, "value", string(request.Value))
	// Example of calling a host function
	//r, _ := m.HttpGet(ctx, &v1.HttpGetRequest{Url: "http://google.com"})
	//log.Printf("Response: %s", string(r.Response))
	resp := v1.ProcessResponse{
		Key:     request.Key,
		Value:   request.Value,
		Headers: request.Headers,
	}
	return &resp, nil
}
