package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
)

const (
	REQUEST = uint8(iota)
	RESPONSE
	HEARTBEAT
	HEARTBEAT_OK
	CONNECTION_REQUEST
	CONNECTION_ACCEPTED
	DISCONNECT
	ERROR
)

const VERSION = 1
const ERROR_MESSAGE = "Something went wrong"

/*
   Version   | MessageType   |  Body Length     | body
   1 byte    |  1 byte       |  8 bytes         | variable length
*/

type TunnelMessage struct {
	Type uint8
	Body []byte
}

type Tunnel struct {
	Id   string
	Conn net.Conn
}

func SerializeMessage(msg TunnelMessage) []byte {
	finalMessage := [][]byte{}
	finalMessage = append(finalMessage, []byte{byte(VERSION), byte(msg.Type)})

	messageLength := make([]byte, 8)
	binary.BigEndian.PutUint64(messageLength, uint64(len(msg.Body)))

	finalMessage = append(finalMessage, messageLength)
	finalMessage = append(finalMessage, msg.Body)

	return bytes.Join(finalMessage, nil)
}

func DeserializeMessage(r *bufio.Reader) (*TunnelMessage, error) {
	version, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	if uint8(version) != VERSION {
		return nil, fmt.Errorf("unsupported version %d", version)
	}

	messageType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch uint8(messageType) {
	case REQUEST, RESPONSE, HEARTBEAT, HEARTBEAT_OK, CONNECTION_ACCEPTED, CONNECTION_REQUEST, ERROR, DISCONNECT:
		// valid
	default:
		return nil, fmt.Errorf("invalid message type: %d", messageType)
	}

	var messageLength uint64
	err = binary.Read(r, binary.BigEndian, &messageLength)
	if err != nil {
		return nil, err
	}

	body := make([]byte, messageLength)
	_, err = io.ReadFull(r, body)
	if err != nil {
		return nil, err
	}

	msg := TunnelMessage{Body: body, Type: messageType}
	return &msg, nil

}

func (t *Tunnel) SendMessage(body []byte, msgType uint8) error {
	msg := TunnelMessage{Body: body, Type: msgType}
	req := SerializeMessage(msg)
	_, err := t.Conn.Write(req)
	return err

}

func EncodeRequest(req *http.Request) ([]byte, error) {
	return httputil.DumpRequest(req, true)
}

func EncodeResponse(res *http.Response) ([]byte, error) {
	return httputil.DumpResponse(res, true)
}

func DecodeRequest(data []byte) (*http.Request, error) {
	reader := bufio.NewReader(bytes.NewReader(data))
	return http.ReadRequest(reader)
}

func DecodeResponse(data []byte, req *http.Request) (*http.Response, error) {
	reader := bufio.NewReader(bytes.NewReader(data))
	return http.ReadResponse(reader, req)
}
