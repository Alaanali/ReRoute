package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Alaanali/ReRoute/protocol"
	"github.com/google/uuid"
)

type Client struct {
	protocol.Tunnel
	tunnelOutbound chan OutBoundResponse
	httpRequests   chan HttpRequest
}

type Server struct {
	mu      sync.Mutex
	clients map[string]Client
}

type OutBoundResponse struct {
	body []byte
}

type HttpRequest struct {
	body     []byte
	response chan OutBoundResponse
}

func (s *Server) handleTCPRequest(client *Client) {
	rd := bufio.NewReader(client.Conn)
	for {

		msg, err := protocol.DeserializeMessage(rd)
		if err != nil {
			return
		}

		switch msg.Type {

		case protocol.RESPONSE:
			client.tunnelOutbound <- OutBoundResponse{body: msg.Body}

		case protocol.HEARTBEAT:
			msg := fmt.Appendf([]byte("Hello World from Server client"), "%s", client.Id)
			client.SendMessage(msg, protocol.HEARTBEAT_OK)
		}

	}
}

func (s *Server) handleInboundRequests(client *Client) {
	for {
		in, ok := <-client.httpRequests
		if !ok {
			return
		}

		// TODO handle error on sending
		client.SendMessage(in.body, protocol.REQUEST)

		res, ok := <-client.tunnelOutbound
		if !ok {
			return
		}

		in.response <- OutBoundResponse{res.body}
	}

}

func (s *Server) handleTCPConnection(conn net.Conn) {
	uniqeID := uuid.New().String()
	tunnelOutbound := make(chan OutBoundResponse)
	httpRequests := make(chan HttpRequest)

	client := Client{protocol.Tunnel{Id: uniqeID, Conn: conn}, tunnelOutbound, httpRequests}

	s.mu.Lock()
	s.clients[uniqeID] = client
	s.mu.Unlock()

	go s.handleTCPRequest(&client)
	go s.handleInboundRequests(&client)

	client.SendMessage([]byte(client.Id), protocol.CONNECTION_ACCEPTED)

}

func (s *Server) handleHttpRequest(w http.ResponseWriter, r *http.Request) {
	// This function shoudl handle the encode and decode of the response
	subDomain := strings.Split(r.Host, ".")[0]
	fmt.Println("subDomain is ", subDomain)
	client, ok := s.clients[subDomain]
	if !ok {
		// 	// TODO return error to the caller
		return
	}
	w.Write([]byte(subDomain))
	response := make(chan OutBoundResponse)
	client.httpRequests <- HttpRequest{[]byte("test message"), response}

	res := <-response
	w.Write(res.body)

}
func main() {

	s := Server{clients: map[string]Client{}}

	go func() {
		listner, err := net.Listen("tcp", "localhost:5500")
		if err != nil {
			log.Fatalln(err)
		}
		for {
			conn, err := listner.Accept()
			if err != nil {
				log.Fatalln(err)
			}
			go s.handleTCPConnection(conn)
		}
	}()

	go func() {
		http.HandleFunc("/", s.handleHttpRequest)
		http.ListenAndServe(":8000", nil)
	}()

	for {
		time.Sleep(time.Second * 50)
	}

}
