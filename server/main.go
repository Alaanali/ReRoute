package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/Alaanali/ReRoute/protocol"
	"github.com/google/uuid"
)

type Client struct {
	protocol.Tunnel
	requests map[uuid.UUID]chan protocol.TunnelMessage
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
}

type Server struct {
	mu      sync.Mutex
	clients map[string]*Client
}

func (s *Server) handleClientDisconnect(client *Client) {
	fmt.Println("Client ", client.Id, "sent disconnect")
	client.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clients[client.Id]; exists {
		client.Conn.Close()
		delete(s.clients, client.Id)
	}
}

func (client *Client) removeRequest(reqId uuid.UUID) {
	client.mu.Lock()
	delete(client.requests, reqId)
	client.mu.Unlock()
}

func (s *Server) handleTCPRequests(client *Client) {
	rd := bufio.NewReader(client.Conn)
	for {

		select {
		case <-client.ctx.Done():
			return
		default:
			client.Conn.SetReadDeadline(time.Now().Add(time.Second * 30))
			msg, err := protocol.DeserializeMessage(rd)
			if err != nil {
				return
			}

			switch msg.Type {

			case protocol.RESPONSE, protocol.ERROR:
				client.mu.Lock()
				ch, ok := client.requests[msg.Id]
				client.mu.Unlock()
				if ok {
					ch <- *msg
				}

			case protocol.HEARTBEAT:
				client.SendMessage([]byte{}, protocol.HEARTBEAT_OK, uuid.New())

			case protocol.DISCONNECT:
				s.handleClientDisconnect(client)
				return
			}
		}

	}
}

func (s *Server) handleTCPConnection(conn net.Conn) {
	uniqueID := uuid.New().String()
	requests := make(map[uuid.UUID]chan protocol.TunnelMessage)
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		Tunnel:   protocol.Tunnel{Id: uniqueID, Conn: conn},
		requests: requests,
		ctx:      ctx,
		cancel:   cancel,
	}

	s.mu.Lock()
	s.clients[uniqueID] = client
	s.mu.Unlock()

	go s.handleTCPRequests(client)

	client.SendMessage([]byte(client.Id), protocol.CONNECTION_ACCEPTED, uuid.New())

}

func (s *Server) handleHttpRequest(w http.ResponseWriter, r *http.Request) {
	subDomain := strings.Split(r.Host, ".")[0]

	client, ok := s.clients[subDomain]
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	encodedRequest, err := protocol.EncodeRequest(r)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	responseChan := make(chan protocol.TunnelMessage, 1)
	messageId := uuid.New()
	defer client.removeRequest(messageId)

	client.mu.Lock()
	client.requests[messageId] = responseChan
	client.mu.Unlock()
	client.SendMessage(encodedRequest, protocol.REQUEST, messageId)

	select {

	case res := <-responseChan:

		if res.Type == protocol.ERROR {
			http.Error(w, string(res.Body), http.StatusInternalServerError)
			return
		}

		decodedResponse, err := protocol.DecodeResponse(res.Body, r)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		defer decodedResponse.Body.Close()

		for key, values := range decodedResponse.Header {
			for _, v := range values {
				w.Header().Add(key, v)
			}
		}

		w.WriteHeader(decodedResponse.StatusCode)
		_, err = io.Copy(w, decodedResponse.Body)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

	case <-client.ctx.Done():
		http.Error(w, "Client disconnected", http.StatusServiceUnavailable)
		return

	case <-r.Context().Done():
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
		return
	case <-time.After(30 * time.Second):
		http.Error(w, "Gateway timeout", http.StatusGatewayTimeout)
		return
	}

}

func main() {
	sigInt := make(chan os.Signal, 1)
	signal.Notify(sigInt, os.Interrupt)
	s := Server{clients: map[string]*Client{}}

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

	<-sigInt

	// currently nothing done after that
}
