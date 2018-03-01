package wsrpc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type V1MessageHeader struct {
	StreamID uint64
	PathLen  uint8
}

type V1Message struct {
	V1MessageHeader
	Path string
	Arg  []byte
}

func readMessage(l *log.Logger, conn *websocket.Conn) (*V1Message, error) {
	t, b, err := conn.ReadMessage()
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

	var h V1MessageHeader
	if err = binary.Read(r, binary.BigEndian, &h); err != nil {
		return nil, errors.Wrap(err, "failed to read v1 message header")
	}
	path := make([]byte, h.PathLen)
	if _, err = io.ReadFull(r, path); err != nil {
		return nil, errors.Wrap(err, "failed to read v1 path")
	}
	var arg []byte
	if arg, err = ioutil.ReadAll(r); err != nil {
		err = errors.Wrap(err, "failed to read v1 arg")
	}

	m := &V1Message{
		V1MessageHeader: h,
		Path:            string(path),
		Arg:             arg,
	}
	l.Printf("conn %p read message: %+v", conn, m)
	return m, nil
}

func writeMessage(l *log.Logger, conn *websocket.Conn, m *V1Message) error {
	l.Printf("conn %p writing message: %+v", conn, m)

	var w bytes.Buffer
	w.WriteByte(1) // version

	if len(m.Path) > 255 {
		panic(fmt.Errorf("%q is too long", m.Path))
	}
	if m.PathLen != uint8(len(m.Path)) {
		panic(fmt.Errorf("Path %q has length %d, expected %d", m.Path, len(m.Path), m.PathLen))
	}
	if err := binary.Write(&w, binary.BigEndian, m.V1MessageHeader); err != nil {
		return errors.Wrap(err, "failed to write v1 message header")
	}
	w.WriteString(m.Path)
	w.Write(m.Arg)

	if err := conn.WriteMessage(websocket.BinaryMessage, w.Bytes()); err != nil {
		return errors.Wrap(err, "failed to write WebSocket message")
	}
	return nil
}
