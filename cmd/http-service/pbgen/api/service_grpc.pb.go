// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: api/service.proto

package api

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Cluster_CreateCluster_FullMethodName = "/api.Cluster/CreateCluster"
	Cluster_UpdateCluster_FullMethodName = "/api.Cluster/UpdateCluster"
	Cluster_GetCluster_FullMethodName    = "/api.Cluster/GetCluster"
)

// ClusterClient is the client API for Cluster service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ClusterClient interface {
	CreateCluster(ctx context.Context, in *CreateClusterReq, opts ...grpc.CallOption) (*CreateClusterResp, error)
	UpdateCluster(ctx context.Context, in *UpdateClusterReq, opts ...grpc.CallOption) (*UpdateClusterResp, error)
	GetCluster(ctx context.Context, in *GetClusterReq, opts ...grpc.CallOption) (*GetClusterResp, error)
}

type clusterClient struct {
	cc grpc.ClientConnInterface
}

func NewClusterClient(cc grpc.ClientConnInterface) ClusterClient {
	return &clusterClient{cc}
}

func (c *clusterClient) CreateCluster(ctx context.Context, in *CreateClusterReq, opts ...grpc.CallOption) (*CreateClusterResp, error) {
	out := new(CreateClusterResp)
	err := c.cc.Invoke(ctx, Cluster_CreateCluster_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterClient) UpdateCluster(ctx context.Context, in *UpdateClusterReq, opts ...grpc.CallOption) (*UpdateClusterResp, error) {
	out := new(UpdateClusterResp)
	err := c.cc.Invoke(ctx, Cluster_UpdateCluster_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterClient) GetCluster(ctx context.Context, in *GetClusterReq, opts ...grpc.CallOption) (*GetClusterResp, error) {
	out := new(GetClusterResp)
	err := c.cc.Invoke(ctx, Cluster_GetCluster_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterServer is the server API for Cluster service.
// All implementations must embed UnimplementedClusterServer
// for forward compatibility
type ClusterServer interface {
	CreateCluster(context.Context, *CreateClusterReq) (*CreateClusterResp, error)
	UpdateCluster(context.Context, *UpdateClusterReq) (*UpdateClusterResp, error)
	GetCluster(context.Context, *GetClusterReq) (*GetClusterResp, error)
	mustEmbedUnimplementedClusterServer()
}

// UnimplementedClusterServer must be embedded to have forward compatible implementations.
type UnimplementedClusterServer struct {
}

func (UnimplementedClusterServer) CreateCluster(context.Context, *CreateClusterReq) (*CreateClusterResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateCluster not implemented")
}
func (UnimplementedClusterServer) UpdateCluster(context.Context, *UpdateClusterReq) (*UpdateClusterResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateCluster not implemented")
}
func (UnimplementedClusterServer) GetCluster(context.Context, *GetClusterReq) (*GetClusterResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCluster not implemented")
}
func (UnimplementedClusterServer) mustEmbedUnimplementedClusterServer() {}

// UnsafeClusterServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ClusterServer will
// result in compilation errors.
type UnsafeClusterServer interface {
	mustEmbedUnimplementedClusterServer()
}

func RegisterClusterServer(s grpc.ServiceRegistrar, srv ClusterServer) {
	s.RegisterService(&Cluster_ServiceDesc, srv)
}

func _Cluster_CreateCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateClusterReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).CreateCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Cluster_CreateCluster_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).CreateCluster(ctx, req.(*CreateClusterReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cluster_UpdateCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateClusterReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).UpdateCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Cluster_UpdateCluster_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).UpdateCluster(ctx, req.(*UpdateClusterReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cluster_GetCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetClusterReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).GetCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Cluster_GetCluster_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).GetCluster(ctx, req.(*GetClusterReq))
	}
	return interceptor(ctx, in, info, handler)
}

// Cluster_ServiceDesc is the grpc.ServiceDesc for Cluster service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Cluster_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.Cluster",
	HandlerType: (*ClusterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateCluster",
			Handler:    _Cluster_CreateCluster_Handler,
		},
		{
			MethodName: "UpdateCluster",
			Handler:    _Cluster_UpdateCluster_Handler,
		},
		{
			MethodName: "GetCluster",
			Handler:    _Cluster_GetCluster_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/service.proto",
}
