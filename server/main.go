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
	msg, err := protocol.DeserializeMessage(rd)
	if err != nil {
		log.Fatalln(err)
	}
	println("Clinet sent ", string(msg))
	msg = protocol.SerializeMessage([]byte("Hello World for server!"))
	conn.Write(msg)
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
