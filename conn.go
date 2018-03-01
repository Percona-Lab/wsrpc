package wsrpc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const (
	wsHandshakeTimeout = 5 * time.Second
	wsBufSize          = 4096
	wsWriterCap        = 0
	wsReadCap          = 0
)

type Conn struct {
	ws *websocket.Conn
	l  *log.Logger

	toWrite chan *V1Message
	read    chan *V1Message

	rw           sync.RWMutex
	nextStreamID uint64 // odd for client-created streams, even for server-created
	readStreams  map[uint64]chan *V1Message
}

// Dial establishes connection by connecting to HTTP server.
func Dial(addr string) (*Conn, error) {
	d := &websocket.Dialer{
		HandshakeTimeout: wsHandshakeTimeout,
		ReadBufferSize:   wsBufSize,
		WriteBufferSize:  wsBufSize,
	}
	ws, _, err := d.Dial(addr, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", addr)
	}
	return makeConn(ws, 1), nil
}

// Upgrade establishes connection by upgrading incoming HTTP request from the client.
func Upgrade(rw http.ResponseWriter, req *http.Request) (*Conn, error) {
	upgrader := &websocket.Upgrader{
		HandshakeTimeout: wsHandshakeTimeout,
		ReadBufferSize:   wsBufSize,
		WriteBufferSize:  wsBufSize,
	}
	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to upgrade connection from %s", req.RemoteAddr)
	}
	return makeConn(ws, 2), nil
}

func makeConn(ws *websocket.Conn, nextStreamID uint64) *Conn {
	conn := &Conn{
		ws:           ws,
		l:            log.New(os.Stderr, fmt.Sprintf("%s->%s: ", ws.LocalAddr(), ws.RemoteAddr()), log.Flags()),
		toWrite:      make(chan *V1Message, wsWriterCap),
		read:         make(chan *V1Message, wsReadCap),
		nextStreamID: nextStreamID,
		readStreams:  make(map[uint64]chan *V1Message),
	}
	go conn.runWriter()
	go conn.runReader()
	return conn
}

// Invoke method on the other side of connection and get response.
func (conn *Conn) Invoke(path string, arg []byte) ([]byte, error) {
	resCh := make(chan *V1Message)

	conn.rw.Lock()
	streamID := conn.nextStreamID
	conn.nextStreamID += 2
	conn.readStreams[streamID] = resCh
	conn.rw.Unlock()

	req := &V1Message{
		V1MessageHeader: V1MessageHeader{
			StreamID: streamID,
			PathLen:  uint8(len(path)),
		},
		Path: path,
		Arg:  arg,
	}
	if err := conn.Write(context.TODO(), req); err != nil {
		return nil, err
	}

	res := <-resCh

	conn.rw.Lock()
	delete(conn.readStreams, streamID)
	conn.rw.Unlock()

	return res.Arg, nil
}

func (conn *Conn) Read(ctx context.Context) (*V1Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case m := <-conn.read:
		return m, nil
	}
}

func (conn *Conn) Write(ctx context.Context, m *V1Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case conn.toWrite <- m:
		return nil
	}
}

func (conn *Conn) runWriter() {
	for m := range conn.toWrite {
		if err := writeMessage(conn.l, conn.ws, m); err != nil {
			log.Panic(err)
		}
	}
}

func (conn *Conn) runReader() {
	for {
		m, err := readMessage(conn.l, conn.ws)
		if err != nil {
			log.Panic(err)
		}

		conn.rw.RLock()
		resCh := conn.readStreams[m.StreamID]
		conn.rw.RUnlock()
		if resCh != nil {
			// response
			resCh <- m
		} else {
			// request
			conn.read <- m
		}
	}
}
