package wsrpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// Message represents WSRPC WebSocket message.
//
// Framing
//
// Each WSRPC message (RPC requests, respones, etc.) maps to a single WebSocket binary message.
//
//   * uint8 : version - fixed to 1
//   * uint64: stream ID
//   * uint8 : path (package + service + method name) length
//   * string: path itself
//   * bytes : request or response body (until the end of the WebSocket message)
type Message struct {
	StreamID uint64
	Path     string
	Arg      []byte
}

func (m Message) String() string {
	return fmt.Sprintf("{%d %s %d bytes}", m.StreamID, m.Path, len(m.Arg))
}

type v1MessageHeader struct {
	StreamID uint64
	PathLen  uint8
}

// readMessage reads one next message from WebSocket connection, and returns it, or wrapped error.
func readMessage(ctx context.Context, ws *websocket.Conn) (*Message, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "already done")
	}

	if d, ok := ctx.Deadline(); ok {
		if err := ws.SetReadDeadline(d); err != nil {
			return nil, errors.Wrap(err, "failed to set read deadline")
		}
	}

	t, b, err := ws.ReadMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read WebSocket message")
	}
	if t != websocket.BinaryMessage {
		return nil, errors.Wrapf(err, "expected binary WebSocket message, got type %d", t)
	}

	r := bytes.NewReader(b)
	version, err := r.ReadByte()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read version byte")
	}
	if version != 1 {
		return nil, errors.Errorf("expected version 1, got %d", version)
	}

	var h v1MessageHeader
	if err = binary.Read(r, binary.BigEndian, &h); err != nil {
		return nil, errors.Wrap(err, "failed to read v1 message header")
	}
	path := make([]byte, h.PathLen)
	if _, err = io.ReadFull(r, path); err != nil {
		return nil, errors.Wrap(err, "failed to read v1 message path")
	}
	var arg []byte
	if arg, err = ioutil.ReadAll(r); err != nil {
		return nil, errors.Wrap(err, "failed to read v1 message arg")
	}

	return &Message{
		StreamID: h.StreamID,
		Path:     string(path),
		Arg:      arg,
	}, nil
}

// writeMessage writes one message to WebSocket connection, and returns nil, or wrapped error.
func writeMessage(ctx context.Context, ws *websocket.Conn, m *Message) error {
	var w bytes.Buffer
	w.WriteByte(1) // version

	if len(m.Path) > 255 {
		return errors.Errorf("path %q is too long", m.Path)
	}
	h := v1MessageHeader{
		StreamID: m.StreamID,
		PathLen:  uint8(len(m.Path)),
	}
	if err := binary.Write(&w, binary.BigEndian, &h); err != nil {
		return errors.Wrap(err, "failed to write v1 message header")
	}
	w.WriteString(m.Path)
	w.Write(m.Arg)

	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "already done")
	}

	if d, ok := ctx.Deadline(); ok {
		if err := ws.SetWriteDeadline(d); err != nil {
			return errors.Wrap(err, "failed to set write deadline")
		}
	}

	if err := ws.WriteMessage(websocket.BinaryMessage, w.Bytes()); err != nil {
		return errors.Wrap(err, "failed to write WebSocket message")
	}
	return nil
}

// check interfaces
var (
	_ fmt.Stringer = Message{}
	_ fmt.Stringer = &Message{}
)
