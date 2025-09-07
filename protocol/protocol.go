package protocol

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
)

const (
	REQUEST = uint8(iota)
	RESPONSE
	HEARTBEAT
	HEARTBEAT_OK
	CONNECTION_REQUEST
	CONNECTION_ACCEPTED
)

const VERSION = 1

const MESSAGE_DATA_DELIMITER = '\n'

/*
   Version   | MessageType   |  Body Lenght      | delimter | body
   1 byte    |  1 byte       |  1 or more bytes  | 1 byte   | variable length
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

	finalMessage = append(finalMessage, []byte(strconv.Itoa(len(msg.Body))))
	finalMessage = append(finalMessage, []byte{byte(MESSAGE_DATA_DELIMITER)})
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
	case REQUEST, RESPONSE, HEARTBEAT, HEARTBEAT_OK, CONNECTION_ACCEPTED, CONNECTION_REQUEST:
		// valid
	default:
		return nil, fmt.Errorf("invalid message type: %d", messageType)
	}

	messageLength, err := r.ReadString(MESSAGE_DATA_DELIMITER)

	if err != nil {
		return nil, err
	}

	n, err := strconv.Atoi(messageLength[:len(messageLength)-1])
	if err != nil {
		return nil, err
	}

	body := make([]byte, n)
	_, err = io.ReadFull(r, body)
	if err != nil {
		return nil, err
	}

	msg := TunnelMessage{Body: body, Type: messageType}
	return &msg, nil

}

func (t *Tunnel) SendMessage(body []byte, msgType uint8) {
	// TODO handle and return errors
	msg := TunnelMessage{Body: body, Type: msgType}
	req := SerializeMessage(msg)
	t.Conn.Write(req)

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
