//go:build tinygo.wasm

// This file represents the API for both the WASM module client and server

// Code generated by protoc-gen-go-plugin. DO NOT EDIT.
// versions:
// 	protoc-gen-go-plugin v0.1.0
// 	protoc               (unknown)
// source: kinetiq/v1/kinetiq.proto

package v1

import (
	context "context"
	wasm "github.com/knqyf263/go-plugin/wasm"
	_ "unsafe"
)

const ModuleServicePluginAPIVersion = 1

//export module_service_api_version
func _module_service_api_version() uint64 {
	return ModuleServicePluginAPIVersion
}

var moduleService ModuleService

func RegisterModuleService(p ModuleService) {
	moduleService = p
}

//export module_service_process
func _module_service_process(ptr, size uint32) uint64 {
	b := wasm.PtrToByte(ptr, size)
	req := new(ProcessRequest)
	if err := req.UnmarshalVT(b); err != nil {
		return 0
	}
	response, err := moduleService.Process(context.Background(), req)
	if err != nil {
		ptr, size = wasm.ByteToPtr([]byte(err.Error()))
		return (uint64(ptr) << uint64(32)) | uint64(size) |
			// Indicate that this is the error string by setting the 32-th bit, assuming that
			// no data exceeds 31-bit size (2 GiB).
			(1 << 31)
	}

	b, err = response.MarshalVT()
	if err != nil {
		return 0
	}
	ptr, size = wasm.ByteToPtr(b)
	return (uint64(ptr) << uint64(32)) | uint64(size)
}

type hostFunctionsService struct{}

func NewHostFunctionsService() HostFunctionsService {
	return hostFunctionsService{}
}

//go:wasm-module env
//export http_get
//go:linkname _http_get
func _http_get(ptr uint32, size uint32) uint64

func (h hostFunctionsService) HttpGet(ctx context.Context, request *HttpGetRequest) (*HttpGetResponse, error) {
	buf, err := request.MarshalVT()
	if err != nil {
		return nil, err
	}
	ptr, size := wasm.ByteToPtr(buf)
	ptrSize := _http_get(ptr, size)
	wasm.FreePtr(ptr)

	ptr = uint32(ptrSize >> 32)
	size = uint32(ptrSize)
	buf = wasm.PtrToByte(ptr, size)

	response := new(HttpGetResponse)
	if err = response.UnmarshalVT(buf); err != nil {
		return nil, err
	}
	return response, nil
}
