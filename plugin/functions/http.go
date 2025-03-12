package functions

import (
	"context"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"io"
	"log"
	"net/http"
)

func (p PluginFunctions) HttpGet(ctx context.Context, request *v1.HttpGetRequest) (*v1.HttpGetResponse, error) {
	log.Printf("Performing http get request: %s - %s", "url", request.Url)
	resp, err := http.Get(request.Url)
	if err != nil {
		return &v1.HttpGetResponse{}, err
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return &v1.HttpGetResponse{}, err
	}

	return &v1.HttpGetResponse{Response: buf}, nil
}
