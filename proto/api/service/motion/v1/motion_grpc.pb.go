// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: proto/api/service/motion/v1/motion.proto

package v1

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

// MotionServiceClient is the client API for MotionService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MotionServiceClient interface {
	PlanAndMove(ctx context.Context, in *PlanAndMoveRequest, opts ...grpc.CallOption) (*PlanAndMoveResponse, error)
	MoveSingleComponent(ctx context.Context, in *MoveSingleComponentRequest, opts ...grpc.CallOption) (*MoveSingleComponentResponse, error)
	GetPose(ctx context.Context, in *GetPoseRequest, opts ...grpc.CallOption) (*GetPoseResponse, error)
}

type motionServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMotionServiceClient(cc grpc.ClientConnInterface) MotionServiceClient {
	return &motionServiceClient{cc}
}

func (c *motionServiceClient) PlanAndMove(ctx context.Context, in *PlanAndMoveRequest, opts ...grpc.CallOption) (*PlanAndMoveResponse, error) {
	out := new(PlanAndMoveResponse)
	err := c.cc.Invoke(ctx, "/proto.api.service.motion.v1.MotionService/PlanAndMove", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *motionServiceClient) MoveSingleComponent(ctx context.Context, in *MoveSingleComponentRequest, opts ...grpc.CallOption) (*MoveSingleComponentResponse, error) {
	out := new(MoveSingleComponentResponse)
	err := c.cc.Invoke(ctx, "/proto.api.service.motion.v1.MotionService/MoveSingleComponent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *motionServiceClient) GetPose(ctx context.Context, in *GetPoseRequest, opts ...grpc.CallOption) (*GetPoseResponse, error) {
	out := new(GetPoseResponse)
	err := c.cc.Invoke(ctx, "/proto.api.service.motion.v1.MotionService/GetPose", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MotionServiceServer is the server API for MotionService service.
// All implementations must embed UnimplementedMotionServiceServer
// for forward compatibility
type MotionServiceServer interface {
	PlanAndMove(context.Context, *PlanAndMoveRequest) (*PlanAndMoveResponse, error)
	MoveSingleComponent(context.Context, *MoveSingleComponentRequest) (*MoveSingleComponentResponse, error)
	GetPose(context.Context, *GetPoseRequest) (*GetPoseResponse, error)
	mustEmbedUnimplementedMotionServiceServer()
}

// UnimplementedMotionServiceServer must be embedded to have forward compatible implementations.
type UnimplementedMotionServiceServer struct {
}

func (UnimplementedMotionServiceServer) PlanAndMove(context.Context, *PlanAndMoveRequest) (*PlanAndMoveResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PlanAndMove not implemented")
}
func (UnimplementedMotionServiceServer) MoveSingleComponent(context.Context, *MoveSingleComponentRequest) (*MoveSingleComponentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MoveSingleComponent not implemented")
}
func (UnimplementedMotionServiceServer) GetPose(context.Context, *GetPoseRequest) (*GetPoseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPose not implemented")
}
func (UnimplementedMotionServiceServer) mustEmbedUnimplementedMotionServiceServer() {}

// UnsafeMotionServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MotionServiceServer will
// result in compilation errors.
type UnsafeMotionServiceServer interface {
	mustEmbedUnimplementedMotionServiceServer()
}

func RegisterMotionServiceServer(s grpc.ServiceRegistrar, srv MotionServiceServer) {
	s.RegisterService(&MotionService_ServiceDesc, srv)
}

func _MotionService_PlanAndMove_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlanAndMoveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MotionServiceServer).PlanAndMove(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.service.motion.v1.MotionService/PlanAndMove",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MotionServiceServer).PlanAndMove(ctx, req.(*PlanAndMoveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MotionService_MoveSingleComponent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MoveSingleComponentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MotionServiceServer).MoveSingleComponent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.service.motion.v1.MotionService/MoveSingleComponent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MotionServiceServer).MoveSingleComponent(ctx, req.(*MoveSingleComponentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MotionService_GetPose_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPoseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MotionServiceServer).GetPose(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.service.motion.v1.MotionService/GetPose",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MotionServiceServer).GetPose(ctx, req.(*GetPoseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// MotionService_ServiceDesc is the grpc.ServiceDesc for MotionService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MotionService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.api.service.motion.v1.MotionService",
	HandlerType: (*MotionServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PlanAndMove",
			Handler:    _MotionService_PlanAndMove_Handler,
		},
		{
			MethodName: "MoveSingleComponent",
			Handler:    _MotionService_MoveSingleComponent_Handler,
		},
		{
			MethodName: "GetPose",
			Handler:    _MotionService_GetPose_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/api/service/motion/v1/motion.proto",
}
