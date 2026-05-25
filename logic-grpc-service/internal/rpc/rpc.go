package rpc

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
)

type Request struct {
	Operation string            `json:"operation"`
	Meta      map[string]string `json:"meta,omitempty"`
	Body      json.RawMessage   `json:"body,omitempty"`
}

type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type LogicServiceServer interface {
	Call(context.Context, *Request) (*Response, error)
}

func RegisterLogicServiceServer(s grpc.ServiceRegistrar, srv LogicServiceServer) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "hr.LogicService",
		HandlerType: (*LogicServiceServer)(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "Call",
			Handler:    callHandler,
		}},
		Streams:  []grpc.StreamDesc{},
		Metadata: "logic.proto",
	}, srv)
}

func callHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LogicServiceServer).Call(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/hr.LogicService/Call"}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(LogicServiceServer).Call(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func OK(data any) *Response {
	raw, _ := json.Marshal(data)
	return &Response{Code: 0, Message: "ok", Data: raw}
}

func Fail(code int, msg string) *Response {
	return &Response{Code: code, Message: msg}
}

func Decode[T any](raw json.RawMessage) (T, error) {
	var v T
	err := json.Unmarshal(raw, &v)
	return v, err
}
