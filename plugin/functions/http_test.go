package functions

import (
	"bytes"
	"context"
	"errors"
	"github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

func TestHttpGet(t *testing.T) {
	mockHttpClient := new(MockHttpClient)
	pluginFunc := NewPluginFunctions(mockHttpClient)

	tests := []struct {
		name       string
		request    *v1.HttpGetRequest
		mockSetup  func()
		expectResp *v1.HttpGetResponse
		expectErr  error
	}{
		{
			name: "Successful request",
			request: &v1.HttpGetRequest{
				Url: "http://example.com",
			},
			mockSetup: func() {
				respBody := io.NopCloser(bytes.NewReader([]byte("success response")))
				mockHttpClient.On("Get", "http://example.com").Return(&http.Response{
					Body: respBody,
				}, nil).Once()
			},
			expectResp: &v1.HttpGetResponse{
				Response: []byte("success response"),
			},
			expectErr: nil,
		},
		{
			name: "HTTP client error",
			request: &v1.HttpGetRequest{
				Url: "http://example.com",
			},
			mockSetup: func() {
				mockHttpClient.On("Get", "http://example.com").Return(nil, errors.New("http client error")).Once()
			},
			expectResp: &v1.HttpGetResponse{},
			expectErr:  errors.New("http client error"),
		},
		{
			name: "Error reading response body",
			request: &v1.HttpGetRequest{
				Url: "http://example.com",
			},
			mockSetup: func() {
				respBody := io.NopCloser(&mockErrorReader{})
				mockHttpClient.On("Get", "http://example.com").Return(&http.Response{
					Body: respBody,
				}, nil).Once()
			},
			expectResp: &v1.HttpGetResponse{},
			expectErr:  errors.New("mock read error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			resp, err := pluginFunc.HttpGet(context.Background(), tt.request)
			assert.Equal(t, tt.expectResp, resp)
			assert.Equal(t, tt.expectErr, err)
			mockHttpClient.AssertExpectations(t)
		})
	}
}

type mockErrorReader struct{}

func (m *mockErrorReader) Read(p []byte) (int, error) {
	return 0, errors.New("mock read error")
}

func (m *mockErrorReader) Close() error {
	return nil
}
