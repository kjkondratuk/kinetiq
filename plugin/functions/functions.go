package functions

import (
	"context"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"net/http"
)

type pluginFunctions struct {
	httpClient HttpClient
}

type HttpClient interface {
	Get(url string) (*http.Response, error)
}

type PluginFunctions interface {
	HttpGet(ctx context.Context, request *v1.HttpGetRequest) (*v1.HttpGetResponse, error)
}

func NewDefaultPluginFunctions() PluginFunctions {
	return NewPluginFunctions(http.DefaultClient)
}

func NewPluginFunctions(httpClient HttpClient) PluginFunctions {
	return &pluginFunctions{httpClient: httpClient}
}
