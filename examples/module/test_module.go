//go:build wasip1

package main

import (
	"context"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"log"
)

func main() {}

func init() {
	v1.RegisterModuleService(MyPlugin{})
}

type MyPlugin struct{}

func (m MyPlugin) Process(ctx context.Context, request *v1.ProcessRequest) (*v1.ProcessResponse, error) {
	log.Printf("Processing request: %s - %s : %s - %d", "id", request.Id, "value", request.Value)
	resp := v1.ProcessResponse{ResponseCode: 1, Message: "OK"}
	return &resp, nil
}
