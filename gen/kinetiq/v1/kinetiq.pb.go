// This file represents the API for both the WASM module client and server

// Code generated by protoc-gen-go-plugin. DO NOT EDIT.
// versions:
// 	protoc-gen-go-plugin 0.9.0
// 	protoc               (unknown)
// source: kinetiq/v1/kinetiq.proto

package v1

import (
	context "context"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// *** Start WASM Module API ***
type ProcessRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key     []byte     `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value   []byte     `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Headers []*Headers `protobuf:"bytes,3,rep,name=headers,proto3" json:"headers,omitempty"`
}

func (x *ProcessRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ProcessRequest) GetKey() []byte {
	if x != nil {
		return x.Key
	}
	return nil
}

func (x *ProcessRequest) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *ProcessRequest) GetHeaders() []*Headers {
	if x != nil {
		return x.Headers
	}
	return nil
}

type Headers struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *Headers) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *Headers) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *Headers) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

type ProcessResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key     []byte     `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value   []byte     `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Headers []*Headers `protobuf:"bytes,3,rep,name=headers,proto3" json:"headers,omitempty"`
}

func (x *ProcessResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ProcessResponse) GetKey() []byte {
	if x != nil {
		return x.Key
	}
	return nil
}

func (x *ProcessResponse) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *ProcessResponse) GetHeaders() []*Headers {
	if x != nil {
		return x.Headers
	}
	return nil
}

type HttpGetRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Url string `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
}

func (x *HttpGetRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *HttpGetRequest) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

type HttpGetResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Response []byte `protobuf:"bytes,1,opt,name=response,proto3" json:"response,omitempty"`
}

func (x *HttpGetResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *HttpGetResponse) GetResponse() []byte {
	if x != nil {
		return x.Response
	}
	return nil
}

// go:plugin type=plugin version=1
type ModuleService interface {
	Process(context.Context, *ProcessRequest) (*ProcessResponse, error)
}

// go:plugin type=host
type HostFunctionsService interface {
	HttpGet(context.Context, *HttpGetRequest) (*HttpGetResponse, error)
}
