package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Alaanali/ReRoute/protocol"
)

func sendheartbeat(conn net.Conn) {
	msg := protocol.TunnelMessage{Type: protocol.HEARTBEAT}
	resp := protocol.SerializeMessage(msg)
	conn.Write(resp)
	rd := bufio.NewReader(conn)
	serverResp, err := protocol.DeserializeMessage(rd)

	if err != nil {
		log.Fatalln("gege", err)
	}

	fmt.Println(string(serverResp.Body))
}
func heartbeatTicker(conn net.Conn, heartbeatchan <-chan string) {

	ticker := time.NewTicker(time.Second * 5)
	paused := false

	for {
		select {
		case cmd := <-heartbeatchan:
			if cmd == "pause" {
				paused = true
			} else {
				paused = false
			}

		case <-ticker.C:
			if !paused {
				sendheartbeat(conn)
			}
		}
	}
}

func main() {
	conn, err := net.DialTimeout("tcp", "localhost:5500", time.Second*30)
	if err != nil {
		log.Fatalln(err)
	}

	msg := protocol.TunnelMessage{Body: []byte("hello  from clinet"), Type: protocol.REQUEST}
	req := protocol.SerializeMessage(msg)

	conn.Write(req)
	rd := bufio.NewReader(conn)

	resp, err := protocol.DeserializeMessage(rd)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("server sent ", string(resp.Body))

	heartbeat := make(chan string, 1)
	go heartbeatTicker(conn, heartbeat)

	for {
		time.Sleep(time.Second * 50)
	}

}
