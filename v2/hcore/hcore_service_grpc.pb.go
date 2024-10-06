// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.28.0
// source: v2/hcore/hcore_service.proto

package hcore

import (
	context "context"
	common "github.com/hiddify/hiddify-core/v2/common"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Core_Start_FullMethodName                 = "/hcore.Core/Start"
	Core_CoreInfoListener_FullMethodName      = "/hcore.Core/CoreInfoListener"
	Core_OutboundsInfo_FullMethodName         = "/hcore.Core/OutboundsInfo"
	Core_MainOutboundsInfo_FullMethodName     = "/hcore.Core/MainOutboundsInfo"
	Core_GetSystemInfo_FullMethodName         = "/hcore.Core/GetSystemInfo"
	Core_Setup_FullMethodName                 = "/hcore.Core/Setup"
	Core_Parse_FullMethodName                 = "/hcore.Core/Parse"
	Core_ChangeHiddifySettings_FullMethodName = "/hcore.Core/ChangeHiddifySettings"
	Core_StartService_FullMethodName          = "/hcore.Core/StartService"
	Core_Stop_FullMethodName                  = "/hcore.Core/Stop"
	Core_Restart_FullMethodName               = "/hcore.Core/Restart"
	Core_SelectOutbound_FullMethodName        = "/hcore.Core/SelectOutbound"
	Core_UrlTest_FullMethodName               = "/hcore.Core/UrlTest"
	Core_GenerateWarpConfig_FullMethodName    = "/hcore.Core/GenerateWarpConfig"
	Core_GetSystemProxyStatus_FullMethodName  = "/hcore.Core/GetSystemProxyStatus"
	Core_SetSystemProxyEnabled_FullMethodName = "/hcore.Core/SetSystemProxyEnabled"
	Core_LogListener_FullMethodName           = "/hcore.Core/LogListener"
)

// CoreClient is the client API for Core service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CoreClient interface {
	Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*CoreInfoResponse, error)
	CoreInfoListener(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[CoreInfoResponse], error)
	OutboundsInfo(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[OutboundGroupList], error)
	MainOutboundsInfo(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[OutboundGroupList], error)
	GetSystemInfo(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[SystemInfo], error)
	Setup(ctx context.Context, in *SetupRequest, opts ...grpc.CallOption) (*common.Response, error)
	Parse(ctx context.Context, in *ParseRequest, opts ...grpc.CallOption) (*ParseResponse, error)
	ChangeHiddifySettings(ctx context.Context, in *ChangeHiddifySettingsRequest, opts ...grpc.CallOption) (*CoreInfoResponse, error)
	// rpc GenerateConfig (GenerateConfigRequest) returns (GenerateConfigResponse);
	StartService(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*CoreInfoResponse, error)
	Stop(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (*CoreInfoResponse, error)
	Restart(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*CoreInfoResponse, error)
	SelectOutbound(ctx context.Context, in *SelectOutboundRequest, opts ...grpc.CallOption) (*common.Response, error)
	UrlTest(ctx context.Context, in *UrlTestRequest, opts ...grpc.CallOption) (*common.Response, error)
	GenerateWarpConfig(ctx context.Context, in *GenerateWarpConfigRequest, opts ...grpc.CallOption) (*WarpGenerationResponse, error)
	GetSystemProxyStatus(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (*SystemProxyStatus, error)
	SetSystemProxyEnabled(ctx context.Context, in *SetSystemProxyEnabledRequest, opts ...grpc.CallOption) (*common.Response, error)
	LogListener(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[LogMessage], error)
}

type coreClient struct {
	cc grpc.ClientConnInterface
}

func NewCoreClient(cc grpc.ClientConnInterface) CoreClient {
	return &coreClient{cc}
}

func (c *coreClient) Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*CoreInfoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CoreInfoResponse)
	err := c.cc.Invoke(ctx, Core_Start_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) CoreInfoListener(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[CoreInfoResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Core_ServiceDesc.Streams[0], Core_CoreInfoListener_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[common.Empty, CoreInfoResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_CoreInfoListenerClient = grpc.ServerStreamingClient[CoreInfoResponse]

func (c *coreClient) OutboundsInfo(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[OutboundGroupList], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Core_ServiceDesc.Streams[1], Core_OutboundsInfo_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[common.Empty, OutboundGroupList]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_OutboundsInfoClient = grpc.ServerStreamingClient[OutboundGroupList]

func (c *coreClient) MainOutboundsInfo(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[OutboundGroupList], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Core_ServiceDesc.Streams[2], Core_MainOutboundsInfo_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[common.Empty, OutboundGroupList]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_MainOutboundsInfoClient = grpc.ServerStreamingClient[OutboundGroupList]

func (c *coreClient) GetSystemInfo(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[SystemInfo], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Core_ServiceDesc.Streams[3], Core_GetSystemInfo_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[common.Empty, SystemInfo]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_GetSystemInfoClient = grpc.ServerStreamingClient[SystemInfo]

func (c *coreClient) Setup(ctx context.Context, in *SetupRequest, opts ...grpc.CallOption) (*common.Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(common.Response)
	err := c.cc.Invoke(ctx, Core_Setup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) Parse(ctx context.Context, in *ParseRequest, opts ...grpc.CallOption) (*ParseResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ParseResponse)
	err := c.cc.Invoke(ctx, Core_Parse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) ChangeHiddifySettings(ctx context.Context, in *ChangeHiddifySettingsRequest, opts ...grpc.CallOption) (*CoreInfoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CoreInfoResponse)
	err := c.cc.Invoke(ctx, Core_ChangeHiddifySettings_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) StartService(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*CoreInfoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CoreInfoResponse)
	err := c.cc.Invoke(ctx, Core_StartService_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) Stop(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (*CoreInfoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CoreInfoResponse)
	err := c.cc.Invoke(ctx, Core_Stop_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) Restart(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*CoreInfoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CoreInfoResponse)
	err := c.cc.Invoke(ctx, Core_Restart_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) SelectOutbound(ctx context.Context, in *SelectOutboundRequest, opts ...grpc.CallOption) (*common.Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(common.Response)
	err := c.cc.Invoke(ctx, Core_SelectOutbound_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) UrlTest(ctx context.Context, in *UrlTestRequest, opts ...grpc.CallOption) (*common.Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(common.Response)
	err := c.cc.Invoke(ctx, Core_UrlTest_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) GenerateWarpConfig(ctx context.Context, in *GenerateWarpConfigRequest, opts ...grpc.CallOption) (*WarpGenerationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(WarpGenerationResponse)
	err := c.cc.Invoke(ctx, Core_GenerateWarpConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) GetSystemProxyStatus(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (*SystemProxyStatus, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SystemProxyStatus)
	err := c.cc.Invoke(ctx, Core_GetSystemProxyStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) SetSystemProxyEnabled(ctx context.Context, in *SetSystemProxyEnabledRequest, opts ...grpc.CallOption) (*common.Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(common.Response)
	err := c.cc.Invoke(ctx, Core_SetSystemProxyEnabled_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) LogListener(ctx context.Context, in *common.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[LogMessage], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Core_ServiceDesc.Streams[4], Core_LogListener_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[common.Empty, LogMessage]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_LogListenerClient = grpc.ServerStreamingClient[LogMessage]

// CoreServer is the server API for Core service.
// All implementations must embed UnimplementedCoreServer
// for forward compatibility.
type CoreServer interface {
	Start(context.Context, *StartRequest) (*CoreInfoResponse, error)
	CoreInfoListener(*common.Empty, grpc.ServerStreamingServer[CoreInfoResponse]) error
	OutboundsInfo(*common.Empty, grpc.ServerStreamingServer[OutboundGroupList]) error
	MainOutboundsInfo(*common.Empty, grpc.ServerStreamingServer[OutboundGroupList]) error
	GetSystemInfo(*common.Empty, grpc.ServerStreamingServer[SystemInfo]) error
	Setup(context.Context, *SetupRequest) (*common.Response, error)
	Parse(context.Context, *ParseRequest) (*ParseResponse, error)
	ChangeHiddifySettings(context.Context, *ChangeHiddifySettingsRequest) (*CoreInfoResponse, error)
	// rpc GenerateConfig (GenerateConfigRequest) returns (GenerateConfigResponse);
	StartService(context.Context, *StartRequest) (*CoreInfoResponse, error)
	Stop(context.Context, *common.Empty) (*CoreInfoResponse, error)
	Restart(context.Context, *StartRequest) (*CoreInfoResponse, error)
	SelectOutbound(context.Context, *SelectOutboundRequest) (*common.Response, error)
	UrlTest(context.Context, *UrlTestRequest) (*common.Response, error)
	GenerateWarpConfig(context.Context, *GenerateWarpConfigRequest) (*WarpGenerationResponse, error)
	GetSystemProxyStatus(context.Context, *common.Empty) (*SystemProxyStatus, error)
	SetSystemProxyEnabled(context.Context, *SetSystemProxyEnabledRequest) (*common.Response, error)
	LogListener(*common.Empty, grpc.ServerStreamingServer[LogMessage]) error
	mustEmbedUnimplementedCoreServer()
}

// UnimplementedCoreServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedCoreServer struct{}

func (UnimplementedCoreServer) Start(context.Context, *StartRequest) (*CoreInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedCoreServer) CoreInfoListener(*common.Empty, grpc.ServerStreamingServer[CoreInfoResponse]) error {
	return status.Errorf(codes.Unimplemented, "method CoreInfoListener not implemented")
}
func (UnimplementedCoreServer) OutboundsInfo(*common.Empty, grpc.ServerStreamingServer[OutboundGroupList]) error {
	return status.Errorf(codes.Unimplemented, "method OutboundsInfo not implemented")
}
func (UnimplementedCoreServer) MainOutboundsInfo(*common.Empty, grpc.ServerStreamingServer[OutboundGroupList]) error {
	return status.Errorf(codes.Unimplemented, "method MainOutboundsInfo not implemented")
}
func (UnimplementedCoreServer) GetSystemInfo(*common.Empty, grpc.ServerStreamingServer[SystemInfo]) error {
	return status.Errorf(codes.Unimplemented, "method GetSystemInfo not implemented")
}
func (UnimplementedCoreServer) Setup(context.Context, *SetupRequest) (*common.Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Setup not implemented")
}
func (UnimplementedCoreServer) Parse(context.Context, *ParseRequest) (*ParseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Parse not implemented")
}
func (UnimplementedCoreServer) ChangeHiddifySettings(context.Context, *ChangeHiddifySettingsRequest) (*CoreInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChangeHiddifySettings not implemented")
}
func (UnimplementedCoreServer) StartService(context.Context, *StartRequest) (*CoreInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartService not implemented")
}
func (UnimplementedCoreServer) Stop(context.Context, *common.Empty) (*CoreInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
func (UnimplementedCoreServer) Restart(context.Context, *StartRequest) (*CoreInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Restart not implemented")
}
func (UnimplementedCoreServer) SelectOutbound(context.Context, *SelectOutboundRequest) (*common.Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SelectOutbound not implemented")
}
func (UnimplementedCoreServer) UrlTest(context.Context, *UrlTestRequest) (*common.Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UrlTest not implemented")
}
func (UnimplementedCoreServer) GenerateWarpConfig(context.Context, *GenerateWarpConfigRequest) (*WarpGenerationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateWarpConfig not implemented")
}
func (UnimplementedCoreServer) GetSystemProxyStatus(context.Context, *common.Empty) (*SystemProxyStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSystemProxyStatus not implemented")
}
func (UnimplementedCoreServer) SetSystemProxyEnabled(context.Context, *SetSystemProxyEnabledRequest) (*common.Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetSystemProxyEnabled not implemented")
}
func (UnimplementedCoreServer) LogListener(*common.Empty, grpc.ServerStreamingServer[LogMessage]) error {
	return status.Errorf(codes.Unimplemented, "method LogListener not implemented")
}
func (UnimplementedCoreServer) mustEmbedUnimplementedCoreServer() {}
func (UnimplementedCoreServer) testEmbeddedByValue()              {}

// UnsafeCoreServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CoreServer will
// result in compilation errors.
type UnsafeCoreServer interface {
	mustEmbedUnimplementedCoreServer()
}

func RegisterCoreServer(s grpc.ServiceRegistrar, srv CoreServer) {
	// If the following call pancis, it indicates UnimplementedCoreServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Core_ServiceDesc, srv)
}

func _Core_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_Start_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).Start(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_CoreInfoListener_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(common.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CoreServer).CoreInfoListener(m, &grpc.GenericServerStream[common.Empty, CoreInfoResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_CoreInfoListenerServer = grpc.ServerStreamingServer[CoreInfoResponse]

func _Core_OutboundsInfo_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(common.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CoreServer).OutboundsInfo(m, &grpc.GenericServerStream[common.Empty, OutboundGroupList]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_OutboundsInfoServer = grpc.ServerStreamingServer[OutboundGroupList]

func _Core_MainOutboundsInfo_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(common.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CoreServer).MainOutboundsInfo(m, &grpc.GenericServerStream[common.Empty, OutboundGroupList]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_MainOutboundsInfoServer = grpc.ServerStreamingServer[OutboundGroupList]

func _Core_GetSystemInfo_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(common.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CoreServer).GetSystemInfo(m, &grpc.GenericServerStream[common.Empty, SystemInfo]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_GetSystemInfoServer = grpc.ServerStreamingServer[SystemInfo]

func _Core_Setup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).Setup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_Setup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).Setup(ctx, req.(*SetupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_Parse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ParseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).Parse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_Parse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).Parse(ctx, req.(*ParseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_ChangeHiddifySettings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ChangeHiddifySettingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).ChangeHiddifySettings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_ChangeHiddifySettings_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).ChangeHiddifySettings(ctx, req.(*ChangeHiddifySettingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_StartService_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).StartService(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_StartService_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).StartService(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(common.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_Stop_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).Stop(ctx, req.(*common.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_Restart_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).Restart(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_Restart_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).Restart(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_SelectOutbound_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SelectOutboundRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).SelectOutbound(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_SelectOutbound_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).SelectOutbound(ctx, req.(*SelectOutboundRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_UrlTest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UrlTestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).UrlTest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_UrlTest_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).UrlTest(ctx, req.(*UrlTestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_GenerateWarpConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GenerateWarpConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).GenerateWarpConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_GenerateWarpConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).GenerateWarpConfig(ctx, req.(*GenerateWarpConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_GetSystemProxyStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(common.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).GetSystemProxyStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_GetSystemProxyStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).GetSystemProxyStatus(ctx, req.(*common.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_SetSystemProxyEnabled_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetSystemProxyEnabledRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).SetSystemProxyEnabled(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Core_SetSystemProxyEnabled_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).SetSystemProxyEnabled(ctx, req.(*SetSystemProxyEnabledRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_LogListener_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(common.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CoreServer).LogListener(m, &grpc.GenericServerStream[common.Empty, LogMessage]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Core_LogListenerServer = grpc.ServerStreamingServer[LogMessage]

// Core_ServiceDesc is the grpc.ServiceDesc for Core service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Core_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "hcore.Core",
	HandlerType: (*CoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Start",
			Handler:    _Core_Start_Handler,
		},
		{
			MethodName: "Setup",
			Handler:    _Core_Setup_Handler,
		},
		{
			MethodName: "Parse",
			Handler:    _Core_Parse_Handler,
		},
		{
			MethodName: "ChangeHiddifySettings",
			Handler:    _Core_ChangeHiddifySettings_Handler,
		},
		{
			MethodName: "StartService",
			Handler:    _Core_StartService_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _Core_Stop_Handler,
		},
		{
			MethodName: "Restart",
			Handler:    _Core_Restart_Handler,
		},
		{
			MethodName: "SelectOutbound",
			Handler:    _Core_SelectOutbound_Handler,
		},
		{
			MethodName: "UrlTest",
			Handler:    _Core_UrlTest_Handler,
		},
		{
			MethodName: "GenerateWarpConfig",
			Handler:    _Core_GenerateWarpConfig_Handler,
		},
		{
			MethodName: "GetSystemProxyStatus",
			Handler:    _Core_GetSystemProxyStatus_Handler,
		},
		{
			MethodName: "SetSystemProxyEnabled",
			Handler:    _Core_SetSystemProxyEnabled_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "CoreInfoListener",
			Handler:       _Core_CoreInfoListener_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "OutboundsInfo",
			Handler:       _Core_OutboundsInfo_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "MainOutboundsInfo",
			Handler:       _Core_MainOutboundsInfo_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetSystemInfo",
			Handler:       _Core_GetSystemInfo_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "LogListener",
			Handler:       _Core_LogListener_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "v2/hcore/hcore_service.proto",
}