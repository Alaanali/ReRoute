package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Alaanali/ReRoute/protocol"
)

const (
	PAUSE = uint8(iota)
	RESUME
)

type Configuration struct {
	tunnelHost    string
	tunnelPort    string
	localhostPort string
}
type Client struct {
	protocol.Tunnel
	Configuration
	heartbeatchan chan uint8
}

func (c *Client) handleIncomingRequestOverTunnel(body []byte) {
	decodedRequest, err := protocol.DecodeRequest(body)

	if err != nil {
		c.SendMessage([]byte(protocol.ERROR_MESSAGE), protocol.ERROR)
		return
	}
	defer decodedRequest.Body.Close()

	localhost := fmt.Sprintf("http://localhost:%v%v", c.localhostPort, decodedRequest.RequestURI)

	localhostURL, err := url.Parse(localhost)

	if err != nil {
		c.SendMessage([]byte(protocol.ERROR_MESSAGE), protocol.ERROR)
		return
	}

	decodedRequest.URL = localhostURL
	decodedRequest.RequestURI = ""

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := http.DefaultClient.Do(decodedRequest.WithContext(ctx))
	if err != nil {
		c.SendMessage([]byte(protocol.ERROR_MESSAGE), protocol.ERROR)
		return
	}

	defer resp.Body.Close()
	duration := time.Since(start)
	c.printRequest(resp, decodedRequest, duration)

	encodedResponse, err := protocol.EncodeResponse(resp)
	if err != nil {
		c.SendMessage([]byte(protocol.ERROR_MESSAGE), protocol.ERROR)
		return
	}
	c.SendMessage(encodedResponse, protocol.RESPONSE)
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
			go c.handleIncomingRequestOverTunnel(resp.Body)

		case protocol.CONNECTION_ACCEPTED:
			c.Id = string(resp.Body)
			c.printTunnelInfo()
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

	tunnelPort := flag.String("tunnelPort", "5500", "port number of tunnel server")
	tunnelHost := flag.String("tunnelHost", "localhost", "host of tunnel server")
	localhostPort := flag.String("localhostPort", "3000", "port number of localhost service")

	flag.Parse()
	conf := Configuration{*tunnelHost, *tunnelPort, *localhostPort}

	// Clear the terminal screen
	fmt.Print("\033[H\033[2J\033[3J")

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(conf.tunnelHost, conf.tunnelPort), time.Second*30)
	if err != nil {
		log.Fatalln(err)
	}

	heartbeatchan := make(chan uint8, 1)
	client := Client{protocol.Tunnel{Conn: conn}, conf, heartbeatchan}

	go client.handleTCPConnection()
	go client.heartbeatTicker()

	client.SendMessage([]byte{}, protocol.CONNECTION_REQUEST)

	for {
		time.Sleep(time.Second * 50)
	}

}
