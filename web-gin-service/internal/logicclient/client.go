package logicclient

import (
	"context"
	"encoding/json"
	"errors"

	"final_homework/web-gin-service/internal/grpcjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
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

type Client struct{ cc *grpc.ClientConn }

func New(addr string) (*Client, error) {
	encoding.RegisterCodec(grpcjson.Codec{})
	cc, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcjson.Codec{})))
	if err != nil {
		return nil, err
	}
	return &Client{cc: cc}, nil
}

func (c *Client) Call(ctx context.Context, op string, meta map[string]string, body any) (*Response, error) {
	raw, _ := json.Marshal(body)
	var out Response
	err := c.cc.Invoke(ctx, "/hr.LogicService/Call", &Request{Operation: op, Meta: meta, Body: raw}, &out)
	if err != nil {
		return nil, err
	}
	if out.Code != 0 {
		return &out, errors.New(out.Message)
	}
	return &out, nil
}

func Decode[T any](resp *Response) (T, error) {
	var v T
	err := json.Unmarshal(resp.Data, &v)
	return v, err
}
