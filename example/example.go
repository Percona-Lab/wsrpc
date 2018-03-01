package example

import (
	"context"

	"github.com/Percona-Lab/wsrpc/example/api"
)

type EchoServer struct{}

func (server *EchoServer) Echo(ctx context.Context, req *api.EchoRequest) (*api.EchoResponse, error) {
	res := &api.EchoResponse{
		Data: "Response for " + req.Data,
	}
	return res, nil
}

// check interface
var _ api.EchoServiceServer = (*EchoServer)(nil)
