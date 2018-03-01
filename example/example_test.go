package example

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Percona-Lab/wsrpc"
	"github.com/Percona-Lab/wsrpc/example/api"
)

func RunTestServer(t *testing.T) *httptest.Server {
	echoServer := new(EchoServer)

	handler := func(rw http.ResponseWriter, req *http.Request) {
		conn, err := wsrpc.Upgrade(rw, req)
		if err != nil {
			t.Error(err)
		}

		go api.NewEchoServiceDispatcher(conn, echoServer).Run(req.Context())

		client := api.NewEchoServiceClient(conn)
		res, err := client.Echo(context.TODO(), &api.EchoRequest{
			Data: "server",
		})
		if err != nil {
			t.Error(err)
		}
		if res.Data != "Response for server" {
			t.Errorf("unexpected response: %q", res.Data)
		}

		<-req.Context().Done()
		t.Log("handler done")
	}

	return httptest.NewServer(http.HandlerFunc(handler))
}

func TestEcho(t *testing.T) {
	testServer := RunTestServer(t)
	defer testServer.Close()
	addr := testServer.Listener.Addr().String()
	log.Printf("Server started on %s", addr)

	conn, err := wsrpc.Dial("ws://" + addr)
	if err != nil {
		t.Fatal(err)
	}

	echoServer := new(EchoServer)
	go api.NewEchoServiceDispatcher(conn, echoServer).Run(context.TODO())

	client := api.NewEchoServiceClient(conn)
	res, err := client.Echo(context.TODO(), &api.EchoRequest{
		Data: "client",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Data != "Response for client" {
		t.Errorf("unexpected response: %q", res.Data)
	}
}
