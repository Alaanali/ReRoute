package protocol

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	REQUEST = uint8(iota)
	RESPONSE
)

const VERSION = 1

const MESSAGE_DATA_DELIMITER = '\n'

/*
   Version   | MessageType   |  Body Lenght      | delimter | body
   1 byte    |  1 byte       |  1 or more bytes  | 1 byte   | variable length
*/

func SerializeMessage(body []byte) []byte {
	finalMessage := [][]byte{}
	finalMessage = append(finalMessage, []byte{byte(VERSION), byte(REQUEST)})

	finalMessage = append(finalMessage, []byte(strconv.Itoa(len(body))))
	finalMessage = append(finalMessage, []byte{byte(MESSAGE_DATA_DELIMITER)})
	finalMessage = append(finalMessage, body)

	return bytes.Join(finalMessage, nil)
}

func DeserializeMessage(r *bufio.Reader) ([]byte, error) {
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

	if uint8(messageType) != REQUEST && uint8(messageType) != RESPONSE {
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

	return body, nil

}
