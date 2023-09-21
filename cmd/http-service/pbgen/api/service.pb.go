// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: api/service.proto

package api

import (
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type User struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Username string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Password string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
}

func (x *User) Reset() {
	*x = User{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_api_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User.ProtoReflect.Descriptor instead.
func (*User) Descriptor() ([]byte, []int) {
	return file_api_service_proto_rawDescGZIP(), []int{0}
}

func (x *User) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *User) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type Resource struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cpu     uint32  `protobuf:"varint,1,opt,name=cpu,proto3" json:"cpu,omitempty"`
	Memory  uint32  `protobuf:"varint,2,opt,name=memory,proto3" json:"memory,omitempty"`
	Storage *uint32 `protobuf:"varint,3,opt,name=storage,proto3,oneof" json:"storage,omitempty"`
}

func (x *Resource) Reset() {
	*x = Resource{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Resource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Resource) ProtoMessage() {}

func (x *Resource) ProtoReflect() protoreflect.Message {
	mi := &file_api_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Resource.ProtoReflect.Descriptor instead.
func (*Resource) Descriptor() ([]byte, []int) {
	return file_api_service_proto_rawDescGZIP(), []int{1}
}

func (x *Resource) GetCpu() uint32 {
	if x != nil {
		return x.Cpu
	}
	return 0
}

func (x *Resource) GetMemory() uint32 {
	if x != nil {
		return x.Memory
	}
	return 0
}

func (x *Resource) GetStorage() uint32 {
	if x != nil && x.Storage != nil {
		return *x.Storage
	}
	return 0
}

type Component struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Replicas uint32    `protobuf:"varint,1,opt,name=replicas,proto3" json:"replicas,omitempty"`
	Resource *Resource `protobuf:"bytes,2,opt,name=resource,proto3" json:"resource,omitempty"`
	Config   string    `protobuf:"bytes,3,opt,name=config,proto3" json:"config,omitempty"`
	Port     *uint32   `protobuf:"varint,4,opt,name=port,proto3,oneof" json:"port,omitempty"`
}

func (x *Component) Reset() {
	*x = Component{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_service_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Component) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Component) ProtoMessage() {}

func (x *Component) ProtoReflect() protoreflect.Message {
	mi := &file_api_service_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Component.ProtoReflect.Descriptor instead.
func (*Component) Descriptor() ([]byte, []int) {
	return file_api_service_proto_rawDescGZIP(), []int{2}
}

func (x *Component) GetReplicas() uint32 {
	if x != nil {
		return x.Replicas
	}
	return 0
}

func (x *Component) GetResource() *Resource {
	if x != nil {
		return x.Resource
	}
	return nil
}

func (x *Component) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}

func (x *Component) GetPort() uint32 {
	if x != nil && x.Port != nil {
		return *x.Port
	}
	return 0
}

type CreateClusterReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClusterId  string     `protobuf:"bytes,1,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty"`
	Version    string     `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	User       *User      `protobuf:"bytes,3,opt,name=user,proto3,oneof" json:"user,omitempty"`
	Pd         *Component `protobuf:"bytes,4,opt,name=pd,proto3" json:"pd,omitempty"`
	Tikv       *Component `protobuf:"bytes,5,opt,name=tikv,proto3" json:"tikv,omitempty"`
	Tiflash    *Component `protobuf:"bytes,6,opt,name=tiflash,proto3,oneof" json:"tiflash,omitempty"`
	Tidb       *Component `protobuf:"bytes,7,opt,name=tidb,proto3" json:"tidb,omitempty"`
	Promehteus *Component `protobuf:"bytes,8,opt,name=promehteus,proto3,oneof" json:"promehteus,omitempty"`
	Grafana    *Component `protobuf:"bytes,9,opt,name=grafana,proto3,oneof" json:"grafana,omitempty"`
}

func (x *CreateClusterReq) Reset() {
	*x = CreateClusterReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_service_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateClusterReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateClusterReq) ProtoMessage() {}

func (x *CreateClusterReq) ProtoReflect() protoreflect.Message {
	mi := &file_api_service_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateClusterReq.ProtoReflect.Descriptor instead.
func (*CreateClusterReq) Descriptor() ([]byte, []int) {
	return file_api_service_proto_rawDescGZIP(), []int{3}
}

func (x *CreateClusterReq) GetClusterId() string {
	if x != nil {
		return x.ClusterId
	}
	return ""
}

func (x *CreateClusterReq) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *CreateClusterReq) GetUser() *User {
	if x != nil {
		return x.User
	}
	return nil
}

func (x *CreateClusterReq) GetPd() *Component {
	if x != nil {
		return x.Pd
	}
	return nil
}

func (x *CreateClusterReq) GetTikv() *Component {
	if x != nil {
		return x.Tikv
	}
	return nil
}

func (x *CreateClusterReq) GetTiflash() *Component {
	if x != nil {
		return x.Tiflash
	}
	return nil
}

func (x *CreateClusterReq) GetTidb() *Component {
	if x != nil {
		return x.Tidb
	}
	return nil
}

func (x *CreateClusterReq) GetPromehteus() *Component {
	if x != nil {
		return x.Promehteus
	}
	return nil
}

func (x *CreateClusterReq) GetGrafana() *Component {
	if x != nil {
		return x.Grafana
	}
	return nil
}

type CreateClusterResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool    `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message *string `protobuf:"bytes,2,opt,name=message,proto3,oneof" json:"message,omitempty"`
}

func (x *CreateClusterResp) Reset() {
	*x = CreateClusterResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_service_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateClusterResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateClusterResp) ProtoMessage() {}

func (x *CreateClusterResp) ProtoReflect() protoreflect.Message {
	mi := &file_api_service_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateClusterResp.ProtoReflect.Descriptor instead.
func (*CreateClusterResp) Descriptor() ([]byte, []int) {
	return file_api_service_proto_rawDescGZIP(), []int{4}
}

func (x *CreateClusterResp) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *CreateClusterResp) GetMessage() string {
	if x != nil && x.Message != nil {
		return *x.Message
	}
	return ""
}

var File_api_service_proto protoreflect.FileDescriptor

var file_api_service_proto_rawDesc = []byte{
	0x0a, 0x11, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x03, 0x61, 0x70, 0x69, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67,
	0x65, 0x6e, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x61, 0x70, 0x69, 0x76, 0x32, 0x2f, 0x6f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xe9, 0x01, 0x0a, 0x04, 0x55, 0x73, 0x65, 0x72, 0x12,
	0x45, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x42, 0x29, 0x92, 0x41, 0x26, 0x32, 0x1c, 0x54, 0x68, 0x65, 0x20, 0x75, 0x73, 0x65, 0x72,
	0x6e, 0x61, 0x6d, 0x65, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73,
	0x74, 0x65, 0x72, 0x2e, 0x4a, 0x06, 0x22, 0x72, 0x6f, 0x6f, 0x74, 0x22, 0x52, 0x08, 0x75, 0x73,
	0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x47, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f,
	0x72, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x2b, 0x92, 0x41, 0x28, 0x32, 0x1c, 0x54,
	0x68, 0x65, 0x20, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x20, 0x6f, 0x66, 0x20, 0x74,
	0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x4a, 0x08, 0x22, 0x31, 0x32,
	0x33, 0x34, 0x35, 0x36, 0x22, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x3a,
	0x51, 0x92, 0x41, 0x4e, 0x0a, 0x4c, 0x2a, 0x04, 0x55, 0x73, 0x65, 0x72, 0x32, 0x31, 0x55, 0x73,
	0x65, 0x72, 0x20, 0x69, 0x73, 0x20, 0x74, 0x68, 0x65, 0x20, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61,
	0x6d, 0x65, 0x20, 0x61, 0x6e, 0x64, 0x20, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x20,
	0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0xd2,
	0x01, 0x10, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f,
	0x72, 0x64, 0x22, 0xe1, 0x02, 0x0a, 0x08, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12,
	0x43, 0x0a, 0x03, 0x63, 0x70, 0x75, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x42, 0x31, 0x92, 0x41,
	0x2e, 0x32, 0x29, 0x54, 0x68, 0x65, 0x20, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x20, 0x6f, 0x66,
	0x20, 0x43, 0x50, 0x55, 0x20, 0x63, 0x6f, 0x72, 0x65, 0x73, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x65,
	0x61, 0x63, 0x68, 0x20, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x2e, 0x4a, 0x01, 0x32, 0x52,
	0x03, 0x63, 0x70, 0x75, 0x12, 0x4f, 0x0a, 0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0d, 0x42, 0x37, 0x92, 0x41, 0x34, 0x32, 0x2f, 0x54, 0x68, 0x65, 0x20, 0x61,
	0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x20,
	0x66, 0x6f, 0x72, 0x20, 0x65, 0x61, 0x63, 0x68, 0x20, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61,
	0x2e, 0x20, 0x75, 0x6e, 0x69, 0x74, 0x3a, 0x20, 0x47, 0x69, 0x4a, 0x01, 0x34, 0x52, 0x06, 0x6d,
	0x65, 0x6d, 0x6f, 0x72, 0x79, 0x12, 0x59, 0x0a, 0x07, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x42, 0x3a, 0x92, 0x41, 0x37, 0x32, 0x30, 0x54, 0x68, 0x65,
	0x20, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x73, 0x74, 0x6f, 0x72, 0x61,
	0x67, 0x65, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x65, 0x61, 0x63, 0x68, 0x20, 0x72, 0x65, 0x70, 0x6c,
	0x69, 0x63, 0x61, 0x2e, 0x20, 0x75, 0x6e, 0x69, 0x74, 0x3a, 0x20, 0x47, 0x69, 0x4a, 0x03, 0x31,
	0x30, 0x30, 0x48, 0x00, 0x52, 0x07, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x88, 0x01, 0x01,
	0x3a, 0x58, 0x92, 0x41, 0x55, 0x0a, 0x53, 0x2a, 0x08, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x32, 0x3b, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x20, 0x69, 0x73, 0x20, 0x74,
	0x68, 0x65, 0x20, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x20, 0x6f, 0x66, 0x20, 0x74,
	0x68, 0x65, 0x20, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x20, 0x66, 0x6f, 0x72,
	0x20, 0x65, 0x61, 0x63, 0x68, 0x20, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x2e, 0xd2, 0x01,
	0x09, 0x63, 0x70, 0x75, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x42, 0x0a, 0x0a, 0x08, 0x5f, 0x73,
	0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x22, 0xa5, 0x03, 0x0a, 0x09, 0x43, 0x6f, 0x6d, 0x70, 0x6f,
	0x6e, 0x65, 0x6e, 0x74, 0x12, 0x4c, 0x0a, 0x08, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x42, 0x30, 0x92, 0x41, 0x2d, 0x32, 0x28, 0x54, 0x68, 0x65,
	0x20, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x20, 0x6f, 0x66, 0x20, 0x72, 0x65, 0x70, 0x6c, 0x69,
	0x63, 0x61, 0x73, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6f, 0x6d, 0x70, 0x6f,
	0x6e, 0x65, 0x6e, 0x74, 0x2e, 0x4a, 0x01, 0x33, 0x52, 0x08, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63,
	0x61, 0x73, 0x12, 0x5f, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x42, 0x34, 0x92, 0x41, 0x31, 0x32, 0x2f, 0x54, 0x68, 0x65, 0x20, 0x72, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6f,
	0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x65, 0x61, 0x63, 0x68,
	0x20, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x2e, 0x52, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x12, 0x39, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x21, 0x92, 0x41, 0x1e, 0x32, 0x1c, 0x54, 0x68, 0x65, 0x20, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6f, 0x6d, 0x70,
	0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x2e, 0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x54,
	0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0d, 0x42, 0x3b, 0x92, 0x41,
	0x38, 0x32, 0x36, 0x54, 0x68, 0x65, 0x20, 0x70, 0x6f, 0x72, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74,
	0x68, 0x65, 0x20, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x2e, 0x20, 0x55, 0x73,
	0x65, 0x64, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x54, 0x69, 0x44, 0x42, 0x2c, 0x20, 0x47, 0x72, 0x61,
	0x66, 0x61, 0x6e, 0x61, 0x20, 0x65, 0x74, 0x63, 0x2e, 0x48, 0x00, 0x52, 0x04, 0x70, 0x6f, 0x72,
	0x74, 0x88, 0x01, 0x01, 0x3a, 0x4f, 0x92, 0x41, 0x4c, 0x0a, 0x4a, 0x2a, 0x09, 0x43, 0x6f, 0x6d,
	0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x32, 0x2a, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e,
	0x74, 0x20, 0x69, 0x73, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65,
	0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x2e, 0xd2, 0x01, 0x10, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x73, 0x72, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x07, 0x0a, 0x05, 0x5f, 0x70, 0x6f, 0x72, 0x74, 0x22, 0xce,
	0x07, 0x0a, 0x10, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x52, 0x65, 0x71, 0x12, 0x5d, 0x0a, 0x0a, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x3e, 0x92, 0x41, 0x3b, 0x32, 0x25, 0x54, 0x68,
	0x65, 0x20, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x20, 0x49, 0x44, 0x20, 0x6f, 0x72, 0x20, 0x6e,
	0x61, 0x6d, 0x65, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x2e, 0x4a, 0x12, 0x22, 0x74, 0x69, 0x64, 0x62, 0x2d, 0x63, 0x6c, 0x73, 0x75, 0x74,
	0x65, 0x72, 0x2d, 0x31, 0x32, 0x33, 0x22, 0x52, 0x09, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x49, 0x64, 0x12, 0x6e, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x54, 0x92, 0x41, 0x51, 0x32, 0x45, 0x54, 0x68, 0x65, 0x20, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x2e, 0x20, 0x4f, 0x6e, 0x6c, 0x79, 0x20, 0x6f, 0x66, 0x66, 0x69, 0x63,
	0x69, 0x61, 0x6c, 0x20, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x20, 0x61, 0x72, 0x65,
	0x20, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x64, 0x20, 0x6e, 0x6f, 0x77, 0x2e, 0x4a,
	0x08, 0x22, 0x76, 0x37, 0x2e, 0x31, 0x2e, 0x30, 0x22, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x12, 0x52, 0x0a, 0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x09, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x42, 0x2e, 0x92, 0x41, 0x2b,
	0x32, 0x29, 0x54, 0x68, 0x65, 0x20, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x20, 0x61,
	0x6e, 0x64, 0x20, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x20, 0x6f, 0x66, 0x20, 0x74,
	0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x48, 0x00, 0x52, 0x04, 0x75,
	0x73, 0x65, 0x72, 0x88, 0x01, 0x01, 0x12, 0x45, 0x0a, 0x02, 0x70, 0x64, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65,
	0x6e, 0x74, 0x42, 0x25, 0x92, 0x41, 0x22, 0x32, 0x20, 0x54, 0x68, 0x65, 0x20, 0x50, 0x44, 0x20,
	0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65,
	0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x52, 0x02, 0x70, 0x64, 0x12, 0x4b, 0x0a,
	0x04, 0x74, 0x69, 0x6b, 0x76, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x42, 0x27, 0x92, 0x41, 0x24,
	0x32, 0x22, 0x54, 0x68, 0x65, 0x20, 0x54, 0x69, 0x4b, 0x56, 0x20, 0x63, 0x6f, 0x6d, 0x70, 0x6f,
	0x6e, 0x65, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73,
	0x74, 0x65, 0x72, 0x2e, 0x52, 0x04, 0x74, 0x69, 0x6b, 0x76, 0x12, 0x59, 0x0a, 0x07, 0x74, 0x69,
	0x66, 0x6c, 0x61, 0x73, 0x68, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x42, 0x2a, 0x92, 0x41, 0x27,
	0x32, 0x25, 0x54, 0x68, 0x65, 0x20, 0x54, 0x69, 0x46, 0x6c, 0x61, 0x73, 0x68, 0x20, 0x63, 0x6f,
	0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x48, 0x01, 0x52, 0x07, 0x74, 0x69, 0x66, 0x6c, 0x61,
	0x73, 0x68, 0x88, 0x01, 0x01, 0x12, 0x4b, 0x0a, 0x04, 0x74, 0x69, 0x64, 0x62, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e,
	0x65, 0x6e, 0x74, 0x42, 0x27, 0x92, 0x41, 0x24, 0x32, 0x22, 0x54, 0x68, 0x65, 0x20, 0x54, 0x69,
	0x44, 0x42, 0x20, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20,
	0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x52, 0x04, 0x74, 0x69,
	0x64, 0x62, 0x12, 0x62, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x6d, 0x65, 0x68, 0x74, 0x65, 0x75, 0x73,
	0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x43, 0x6f, 0x6d,
	0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x42, 0x2d, 0x92, 0x41, 0x2a, 0x32, 0x28, 0x54, 0x68, 0x65,
	0x20, 0x50, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x20, 0x63, 0x6f, 0x6d, 0x70,
	0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x2e, 0x48, 0x02, 0x52, 0x0a, 0x70, 0x72, 0x6f, 0x6d, 0x65, 0x68, 0x74,
	0x65, 0x75, 0x73, 0x88, 0x01, 0x01, 0x12, 0x59, 0x0a, 0x07, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e,
	0x61, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x43, 0x6f,
	0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x42, 0x2a, 0x92, 0x41, 0x27, 0x32, 0x25, 0x54, 0x68,
	0x65, 0x20, 0x47, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x20, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e,
	0x65, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x2e, 0x48, 0x03, 0x52, 0x07, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x88, 0x01,
	0x01, 0x3a, 0x6c, 0x92, 0x41, 0x69, 0x0a, 0x67, 0x2a, 0x10, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x32, 0x35, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x20, 0x69, 0x73, 0x20,
	0x74, 0x68, 0x65, 0x20, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x20, 0x66, 0x6f, 0x72, 0x20,
	0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6e, 0x67, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x2e, 0xd2, 0x01, 0x1b, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x70, 0x64, 0x74, 0x69, 0x6b, 0x76, 0x74, 0x69, 0x64, 0x62, 0x42,
	0x07, 0x0a, 0x05, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x42, 0x0a, 0x0a, 0x08, 0x5f, 0x74, 0x69, 0x66,
	0x6c, 0x61, 0x73, 0x68, 0x42, 0x0d, 0x0a, 0x0b, 0x5f, 0x70, 0x72, 0x6f, 0x6d, 0x65, 0x68, 0x74,
	0x65, 0x75, 0x73, 0x42, 0x0a, 0x0a, 0x08, 0x5f, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x22,
	0x89, 0x02, 0x0a, 0x11, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x52, 0x65, 0x73, 0x70, 0x12, 0x47, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x42, 0x2d, 0x92, 0x41, 0x2a, 0x32, 0x22, 0x57, 0x68, 0x65,
	0x74, 0x68, 0x65, 0x72, 0x20, 0x74, 0x68, 0x65, 0x20, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x20, 0x69, 0x73, 0x20, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x66, 0x75, 0x6c, 0x2e, 0x4a,
	0x04, 0x74, 0x72, 0x75, 0x65, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x40,
	0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42,
	0x21, 0x92, 0x41, 0x1e, 0x32, 0x1c, 0x54, 0x68, 0x65, 0x20, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x2e, 0x48, 0x00, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x88, 0x01, 0x01,
	0x3a, 0x5d, 0x92, 0x41, 0x5a, 0x0a, 0x58, 0x2a, 0x11, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x32, 0x37, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x20, 0x69, 0x73,
	0x20, 0x74, 0x68, 0x65, 0x20, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x20, 0x66, 0x6f,
	0x72, 0x20, 0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6e, 0x67, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x2e, 0xd2, 0x01, 0x09, 0x62, 0x61, 0x73, 0x65, 0x5f, 0x72, 0x65, 0x73, 0x70, 0x42,
	0x0a, 0x0a, 0x08, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0xc1, 0x01, 0x0a, 0x07,
	0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x12, 0x80, 0x01, 0x0a, 0x0d, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x12, 0x15, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71,
	0x1a, 0x16, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x22, 0x40, 0x92, 0x41, 0x22, 0x12, 0x11, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x20, 0x61, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e,
	0x2a, 0x0d, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x82,
	0xd3, 0xe4, 0x93, 0x02, 0x15, 0x3a, 0x01, 0x2a, 0x22, 0x10, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74,
	0x61, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x73, 0x1a, 0x33, 0x92, 0x41, 0x30, 0x12,
	0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x2c, 0x20, 0x67, 0x65, 0x74, 0x2c, 0x20, 0x6d, 0x6f,
	0x64, 0x69, 0x66, 0x79, 0x2c, 0x20, 0x61, 0x6e, 0x64, 0x20, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65,
	0x20, 0x54, 0x69, 0x44, 0x42, 0x20, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x42,
	0x78, 0x92, 0x41, 0x3e, 0x12, 0x3c, 0x0a, 0x11, 0x54, 0x69, 0x44, 0x42, 0x20, 0x4f, 0x70, 0x65,
	0x72, 0x61, 0x74, 0x6f, 0x72, 0x20, 0x41, 0x50, 0x49, 0x12, 0x1e, 0x54, 0x68, 0x69, 0x73, 0x20,
	0x69, 0x73, 0x20, 0x74, 0x68, 0x65, 0x20, 0x54, 0x69, 0x44, 0x42, 0x20, 0x4f, 0x70, 0x65, 0x72,
	0x61, 0x74, 0x6f, 0x72, 0x20, 0x41, 0x50, 0x49, 0x2e, 0x32, 0x07, 0x76, 0x31, 0x2d, 0x62, 0x65,
	0x74, 0x61, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70,
	0x69, 0x6e, 0x67, 0x63, 0x61, 0x70, 0x2f, 0x74, 0x69, 0x64, 0x62, 0x2d, 0x6f, 0x70, 0x65, 0x72,
	0x61, 0x74, 0x6f, 0x72, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2d, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x2f, 0x61, 0x70, 0x69, 0x3b, 0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_api_service_proto_rawDescOnce sync.Once
	file_api_service_proto_rawDescData = file_api_service_proto_rawDesc
)

func file_api_service_proto_rawDescGZIP() []byte {
	file_api_service_proto_rawDescOnce.Do(func() {
		file_api_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_service_proto_rawDescData)
	})
	return file_api_service_proto_rawDescData
}

var file_api_service_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_api_service_proto_goTypes = []interface{}{
	(*User)(nil),              // 0: api.User
	(*Resource)(nil),          // 1: api.Resource
	(*Component)(nil),         // 2: api.Component
	(*CreateClusterReq)(nil),  // 3: api.CreateClusterReq
	(*CreateClusterResp)(nil), // 4: api.CreateClusterResp
}
var file_api_service_proto_depIdxs = []int32{
	1, // 0: api.Component.resource:type_name -> api.Resource
	0, // 1: api.CreateClusterReq.user:type_name -> api.User
	2, // 2: api.CreateClusterReq.pd:type_name -> api.Component
	2, // 3: api.CreateClusterReq.tikv:type_name -> api.Component
	2, // 4: api.CreateClusterReq.tiflash:type_name -> api.Component
	2, // 5: api.CreateClusterReq.tidb:type_name -> api.Component
	2, // 6: api.CreateClusterReq.promehteus:type_name -> api.Component
	2, // 7: api.CreateClusterReq.grafana:type_name -> api.Component
	3, // 8: api.Cluster.CreateCluster:input_type -> api.CreateClusterReq
	4, // 9: api.Cluster.CreateCluster:output_type -> api.CreateClusterResp
	9, // [9:10] is the sub-list for method output_type
	8, // [8:9] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_api_service_proto_init() }
func file_api_service_proto_init() {
	if File_api_service_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*User); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Resource); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_service_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Component); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_service_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateClusterReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_service_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateClusterResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_api_service_proto_msgTypes[1].OneofWrappers = []interface{}{}
	file_api_service_proto_msgTypes[2].OneofWrappers = []interface{}{}
	file_api_service_proto_msgTypes[3].OneofWrappers = []interface{}{}
	file_api_service_proto_msgTypes[4].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_service_proto_goTypes,
		DependencyIndexes: file_api_service_proto_depIdxs,
		MessageInfos:      file_api_service_proto_msgTypes,
	}.Build()
	File_api_service_proto = out.File
	file_api_service_proto_rawDesc = nil
	file_api_service_proto_goTypes = nil
	file_api_service_proto_depIdxs = nil
}
