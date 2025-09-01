package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Alaanali/ReRoute/protocol"
)

func main() {
	conn, err := net.DialTimeout("tcp", "localhost:5500", time.Second*30)
	if err != nil {
		log.Fatalln(err)
	}

	msg := protocol.SerializeMessage([]byte("hello  from clinet"))
	conn.Write(msg)
	rd := bufio.NewReader(conn)

	msg, err = protocol.DeserializeMessage(rd)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("server sent ", string(msg))
	conn.Close()
}
