package example

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Percona-Lab/wsrpc"
	"github.com/Percona-Lab/wsrpc/example/api"
)

func RunTestServer(ctx context.Context, t *testing.T, wg *sync.WaitGroup) *httptest.Server {
	echoServer := new(EchoServer)

	handler := func(rw http.ResponseWriter, req *http.Request) {
		wg.Add(1)
		defer wg.Done()

		conn, err := wsrpc.Upgrade(rw, req)
		if err != nil {
			t.Error(err)
			return
		}

		go func() {
			wg.Add(1)
			defer wg.Done()

			if runErr := api.NewEchoServiceDispatcher(conn, echoServer).Run(); runErr != nil {
				t.Logf("Run exited with %s", runErr)
			}
		}()

		// invoke method on connected client
		client := api.NewEchoServiceClient(conn)
		res, err := client.Echo(&api.EchoRequest{
			Data: "server",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if res.Data != "Response for server" {
			t.Errorf("unexpected response: %q", res.Data)
		}

		<-ctx.Done()
		if err = conn.Close(); err != nil {
			t.Log(err)
		}
	}

	return httptest.NewServer(http.HandlerFunc(handler))
}

func TestEchoServer(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testServer := RunTestServer(ctx, t, &wg)
	defer testServer.Close()

	addr := testServer.Listener.Addr().String()
	logrus.Printf("Server started on %s", addr)

	conn, err := wsrpc.Dial("ws://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	echoServer := new(EchoServer)
	go func() {
		wg.Add(1)
		defer wg.Done()

		if runErr := api.NewEchoServiceDispatcher(conn, echoServer).Run(); runErr != nil {
			t.Logf("Run exited with %s", runErr)
		}
	}()

	client := api.NewEchoServiceClient(conn)
	res, err := client.Echo(&api.EchoRequest{
		Data: "client",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Data != "Response for client" {
		t.Errorf("unexpected response: %q", res.Data)
	}
}

func TestMain(m *testing.M) {
	before := runtime.NumGoroutine()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
	e := m.Run()

	after := runtime.NumGoroutine()
	if before != after {
		debug.SetTraceback("all")
		panic(fmt.Sprintf("started with %d goroutines, have %d now", before, after))
	}

	os.Exit(e)
}
