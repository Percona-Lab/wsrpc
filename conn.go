package wsrpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	wsHandshakeTimeout = 5 * time.Second
	wsWriteTimeout     = time.Second
	wsPingInterval     = 30 * time.Second
	wsBufSize          = 4096
	wsReadCap          = 0
)

var (
	errConnectionClosed = errors.New("wsrpc: WebSocket connection closed")
)

// Conn.
//
// All exported Conn methods are safe for concurrent usage.
//
// Connection termination
//
// Conn may decide to terminate underlying WebSocket connection when:
//  * runReader exits due to read errors, timeouts, unmarshalling errors, etc.;
//  * Write exits due to write errors, timeouts, marshalling errors, etc.;
//  * TODO runPinger exits due to write errors, timeouts;
//  * TODO pongs are not received for some time.
//
// In that case it just calls stop(error) with appropriate error.
type Conn struct {
	ws *websocket.Conn
	l  *logrus.Entry

	stopOnce sync.Once
	stopErr  error
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	writeM sync.Mutex

	read chan *Message

	readRW           sync.RWMutex
	readNextStreamID uint64 // odd for client-created streams, even for server-created
	readStreams      map[uint64]chan *Message
}

// Dial establishes connection by connecting to HTTP server.
func Dial(addr string, headers http.Header) (*Conn, http.Header, error) {
	d := &websocket.Dialer{
		HandshakeTimeout: wsHandshakeTimeout,
		ReadBufferSize:   wsBufSize,
		WriteBufferSize:  wsBufSize,
	}
	ws, resp, err := d.Dial(addr, headers)
	if err != nil {
		var respHeaders http.Header
		if resp != nil {
			respHeaders = resp.Header
			b, _ := httputil.DumpResponse(resp, true)
			logrus.WithField("component", "wsrpc").Debugf("Failed to connect to %s:\n%s", addr, b)
		}
		return nil, respHeaders, errors.Wrapf(err, "failed to connect to %s", addr)
	}
	return makeConn(ws, 1, "client->server"), resp.Header, nil
}

// Upgrade establishes connection by upgrading incoming HTTP request from the client.
func Upgrade(rw http.ResponseWriter, req *http.Request, respHeaders http.Header) (*Conn, error) {
	upgrader := &websocket.Upgrader{
		HandshakeTimeout: wsHandshakeTimeout,
		ReadBufferSize:   wsBufSize,
		WriteBufferSize:  wsBufSize,
	}
	ws, err := upgrader.Upgrade(rw, req, respHeaders)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to upgrade connection from %s", req.RemoteAddr)
	}
	return makeConn(ws, 2, "server->client"), nil
}

func makeConn(ws *websocket.Conn, readNextStreamID uint64, logConn string) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	if logConn == "" {
		logConn = fmt.Sprintf("%s->%s", ws.LocalAddr(), ws.RemoteAddr())
	}
	conn := &Conn{
		ws:               ws,
		l:                logrus.WithField("component", "wsrpc").WithField("conn", logConn),
		ctx:              ctx,
		cancel:           cancel,
		read:             make(chan *Message, wsReadCap),
		readNextStreamID: readNextStreamID,
		readStreams:      make(map[uint64]chan *Message),
	}
	conn.wg.Add(2)
	go conn.runPinger()
	go conn.runReader()
	conn.ws.SetPongHandler(conn.pongHandler)
	return conn
}

// Close properly closes WebSocket connection.
func (conn *Conn) Close() error {
	err := conn.stop(nil)
	conn.wg.Wait()
	return err
}

// stop properly closes WebSocket connection with given error (which can be nil).
// It also cancels connection context.
func (conn *Conn) stop(err error) error {
	conn.stopOnce.Do(func() {
		conn.stopErr = err

		switch e := errors.Cause(err).(type) {
		case nil:
			conn.l.Debug("Closing connection")
		case *websocket.CloseError:
			conn.l.Warnf("Closing connection: received %q", e)
		default:
			conn.l.Errorf("Closing connection: %+v", err)
		}

		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye")
		if err != nil {
			// TODO use different codes for different errors
			msg = websocket.FormatCloseMessage(websocket.CloseGoingAway, err.Error())
		}

		if e := conn.ws.WriteControl(websocket.CloseMessage, msg, time.Now().Add(wsWriteTimeout)); e != nil {
			err = e
		}
		if e := conn.ws.Close(); e != nil {
			err = e
		}

		if conn.stopErr == nil {
			conn.stopErr = err
		}

		conn.cancel()
	})

	return conn.stopErr
}

func (conn *Conn) pongHandler(data string) error {
	nsec, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "failed to parse %q", data)
	}
	t := time.Unix(0, nsec)
	conn.l.Infof("Latency %s", time.Since(t))
	return nil
}

// Invoke method on the other side of connection and get response.
func (conn *Conn) Invoke(path string, arg []byte) ([]byte, error) {
	ch := make(chan *Message)

	conn.readRW.Lock()
	streamID := conn.readNextStreamID
	conn.readNextStreamID += 2
	conn.readStreams[streamID] = ch
	conn.readRW.Unlock()

	if ch == nil {
		panic(fmt.Sprintf("channel for stream %d not found", streamID))
	}

	req := &Message{
		StreamID: streamID,
		Path:     path,
		Arg:      arg,
	}
	if err := conn.Write(req); err != nil {
		return nil, err
	}

	res := <-ch
	close(ch) // to make possible bugs more visible

	conn.readRW.Lock()
	delete(conn.readStreams, streamID)
	conn.readRW.Unlock()

	return res.Arg, nil
}

func (conn *Conn) Read() (*Message, error) {
	select {
	case <-conn.ctx.Done():
		return nil, conn.ctx.Err()
	case m, ok := <-conn.read:
		if !ok {
			return nil, errConnectionClosed
		}
		return m, nil
	}
}

func (conn *Conn) Write(m *Message) error {
	var err error
	defer func() {
		if err != nil {
			conn.stop(err)
		}
	}()

	conn.writeM.Lock()
	defer conn.writeM.Unlock()

	conn.l.Debugf("Write: %+v", m)
	err = errors.WithStack(writeMessage(conn.ctx, conn.ws, m))
	return err
}

// runPinger writes WebSocket ping messages periodically.
// When connection context is done, or on any other error, it stops connection and exits.
func (conn *Conn) runPinger() {
	var err error
	defer func() {
		conn.stop(err)
		conn.wg.Done()
	}()

	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	var t time.Time
	for {
		select {
		case <-conn.ctx.Done():
			err = errors.WithStack(conn.ctx.Err())
			return

		case t = <-ticker.C:
			data := []byte(strconv.FormatInt(t.UnixNano(), 10))
			if err = conn.ws.WriteControl(websocket.PingMessage, data, time.Now().Add(wsWriteTimeout)); err != nil {
				err = errors.WithStack(err)
				return
			}
		}
	}
}

// runReader reads WSRPC messages from WebSocket connection, sends responses to awaiting Invoke()-ers,
// sends requests to Read()-ers.
// When connection context is done, or on any other error, it stops connection and exits.
func (conn *Conn) runReader() {
	var err error
	defer func() {
		conn.stop(err)
		close(conn.read)
		conn.wg.Done()
	}()

	var m *Message
	for {
		m, err = readMessage(conn.ctx, conn.ws)
		if err != nil {
			return
		}
		conn.l.Debugf("runReader: %+v", m)

		conn.readRW.RLock()
		ch := conn.readStreams[m.StreamID] // is it response?
		conn.readRW.RUnlock()
		if ch == nil {
			// no, it is request
			ch = conn.read
		}

		select {
		case ch <- m:
			// nothing, continue loop
		case <-conn.ctx.Done():
			err = errors.WithStack(conn.ctx.Err())
			return
		}
	}
}
