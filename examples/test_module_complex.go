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

func main() {}

func init() {
	v1.RegisterModuleService(MyPlugin{HostFunctionsService: v1.NewHostFunctionsService()})
}

type MyPlugin struct {
	v1.HostFunctionsService
}

func (m MyPlugin) Process(ctx context.Context, request *v1.ProcessRequest) (*v1.ProcessResponse, error) {
	log.Printf("Processing BRAND request: %s - %s : %s - %s", "key", request.Key, "value", string(request.Value))

	data := samplePayload{}
	err := json.Unmarshal(request.Value, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sample payload in webassembly module: %w", err)
	}

	// Example of calling a host function
	// TODO : need a configuration API to provide/seed values like this, or some ability to load a config file
	r, _ := m.HttpGet(ctx, &v1.HttpGetRequest{Url: "https://stage-brand-service.fetchrewards.com/v2/brands/by-user/679d402a4572edcb3fb66882"})
	brands := UserBrandResponse{}
	err = json.Unmarshal(r.Response, &brands)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal brand response in webassembly module: %w", err)
	}

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

	combinedBytes, _ := json.Marshal(combined)
	log.Printf("Combined Response: %s", combinedBytes)

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
