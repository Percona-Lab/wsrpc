package example

import (
	"github.com/Percona-Lab/wsrpc/example/api"
)

type EchoServer struct{}

func (server *EchoServer) Echo(req *api.EchoRequest) (*api.EchoResponse, error) {
	res := &api.EchoResponse{
		Data: "Response for " + req.Data,
	}
	return res, nil
}

func (server *EchoServer) Empty(req *api.EmptyRequest) (*api.EmptyResponse, error) {
	return new(api.EmptyResponse), nil
}

// check interfaces
var _ api.EchoServiceServer = (*EchoServer)(nil)
