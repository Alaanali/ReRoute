package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Alaanali/ReRoute/protocol"
)

const (
	PAUSE = uint8(iota)
	RESUME
)

type Client struct {
	protocol.Tunnel
	heartbeatchan chan uint8
}

func (c *Client) handleTCPConnection() {
	rd := bufio.NewReader(c.Conn)

	for {
		resp, err := protocol.DeserializeMessage(rd)
		if err != nil {
			log.Fatalln(err)
		}

		c.heartbeatchan <- PAUSE

		switch resp.Type {
		case protocol.REQUEST:
			r := [][]byte{}
			r = append(r, []byte("received"))
			r = append(r, resp.Body)
			c.SendMessage(bytes.Join(r, []byte(" ")), protocol.RESPONSE)

		case protocol.CONNECTION_ACCEPTED:
			c.Id = string(resp.Body)
			fmt.Println("Your subdomain is ", c.Id)
		}

		c.heartbeatchan <- RESUME

	}
}
func (client *Client) heartbeatTicker() {

	ticker := time.NewTicker(time.Second * 5)
	paused := false

	for {
		select {
		case cmd := <-client.heartbeatchan:
			if cmd == PAUSE {
				paused = true
			} else {
				paused = false
			}

		case <-ticker.C:
			if !paused {
				client.SendMessage(nil, protocol.HEARTBEAT)
			}
		}
	}
}

func main() {
	conn, err := net.DialTimeout("tcp", "localhost:5500", time.Second*30)
	if err != nil {
		log.Fatalln(err)
	}

	heartbeatchan := make(chan uint8, 1)
	client := Client{protocol.Tunnel{Conn: conn, Id: ""}, heartbeatchan}

	go client.handleTCPConnection()
	go client.heartbeatTicker()

	client.SendMessage([]byte{}, protocol.CONNECTION_REQUEST)

	for {
		time.Sleep(time.Second * 50)
	}

}
