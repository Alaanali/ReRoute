package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
	tunnelOutbound chan protocol.TunnelMessage
	httpRequests   chan HttpRequest
	ctx            context.Context
	cancel         context.CancelFunc
}

type Server struct {
	mu      sync.Mutex
	clients map[string]Client
}

type HttpRequest struct {
	body     []byte
	response chan protocol.TunnelMessage
}

func (s *Server) handleClientDisconnect(client *Client) {
	fmt.Println("Client ", client.Id, "sent disconnect")
	client.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clients[client.Id]; exists {
		client.Conn.Close()
		close(client.httpRequests)
		close(client.tunnelOutbound)
		delete(s.clients, client.Id)
	}
}

func (s *Server) handleTCPRequest(client *Client) {
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
				client.tunnelOutbound <- *msg

			case protocol.HEARTBEAT:
				client.SendMessage([]byte{}, protocol.HEARTBEAT_OK)

			case protocol.DISCONNECT:
				s.handleClientDisconnect(client)
				return
			}
		}

	}
}

func (s *Server) handleInboundRequests(client *Client) {
	for {
		select {

		case <-client.ctx.Done():
			return

		case in, ok := <-client.httpRequests:

			if !ok {
				return
			}

			// TODO handle error on sending
			client.SendMessage(in.body, protocol.REQUEST)

			select {

			case res, ok := <-client.tunnelOutbound:
				if !ok {
					return
				}

				in.response <- res

			case <-client.ctx.Done():
				return

			case <-time.After(time.Second * 10): // Request timeout
				in.response <- protocol.TunnelMessage{
					Type: protocol.ERROR,
					Body: []byte("Request timeout"),
				}
			}

		}
	}

}

func (s *Server) handleTCPConnection(conn net.Conn) {
	uniqueID := uuid.New().String()
	tunnelOutbound := make(chan protocol.TunnelMessage)
	httpRequests := make(chan HttpRequest)

	ctx, cancel := context.WithCancel(context.Background())
	client := Client{protocol.Tunnel{Id: uniqueID, Conn: conn}, tunnelOutbound, httpRequests, ctx, cancel}

	s.mu.Lock()
	s.clients[uniqueID] = client
	s.mu.Unlock()

	go s.handleTCPRequest(&client)
	go s.handleInboundRequests(&client)

	client.SendMessage([]byte(client.Id), protocol.CONNECTION_ACCEPTED)

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

	response := make(chan protocol.TunnelMessage)

	select {
	case client.httpRequests <- HttpRequest{encodedRequest, response}:
		// Success
	case <-client.ctx.Done():
		http.Error(w, "Client disconnected", http.StatusServiceUnavailable)
		return
	case <-r.Context().Done():
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
		return
	}

	select {

	case res := <-response:

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
	}

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
