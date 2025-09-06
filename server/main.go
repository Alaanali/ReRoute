package main

import (
	"bufio"
	"log"
	"net"

	"github.com/Alaanali/ReRoute/protocol"
)

func handleRequest(conn net.Conn) {
	defer conn.Close()
	rd := bufio.NewReader(conn)
	for {

		msg, err := protocol.DeserializeMessage(rd)
		if err != nil {
			return
		}
		println("Clinet sent ", string(msg.Body))

		resp := protocol.TunnelMessage{Body: []byte("Hello World for server!"), Type: protocol.RESPONSE}
		respMsg := protocol.SerializeMessage(resp)
		conn.Write(respMsg)
	}
}
func main() {
	listner, err := net.Listen("tcp", "localhost:5500")

	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := listner.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go handleRequest(conn)
	}
}
