# ReRoute

A lightweight TCP tunnel implementation in Go that allows you to expose local services through a public endpoint. Built as a weekend project for learning and experimentation.

## Architecture

ReRoute consists of three main components:

```
┌─────────────────┐    TCP Connection    ┌─────────────────┐
│                 │◄───────────────────►│                 │
│  ReRoute Server │                     │  ReRoute Client │
│   (Public)      │                     │   (Local)       │
│                 │                     │                 │
└─────────┬───────┘                     └─────────┬───────┘
          │                                       │
          │ HTTP Requests                         │
          │                                       │
          ▼                                       ▼
┌─────────────────┐                     ┌─────────────────┐
│   Web Browser   │                     │ Local Service   │
│                 │                     │ (localhost:3000)│
└─────────────────┘                     └─────────────────┘
```

### Protocol Design

ReRoute uses a custom binary protocol for efficient communication:

```
+----------+-------------+--------------+--------------+
| Version  | MessageType | Body Length  |     Body     |
| 1 byte   |   1 byte    |   8 bytes    |   Variable   |
+----------+-------------+--------------+--------------+
```

**Message Types:**
- `REQUEST` - HTTP request forwarding
- `RESPONSE` - HTTP response forwarding  
- `HEARTBEAT` - Connection health check
- `CONNECTION_REQUEST` - Initial client connection
- `CONNECTION_ACCEPTED` - Server acknowledgment
- `DISCONNECT` - Graceful disconnection
- `ERROR` - Error handling

## Features

- **Custom Binary Protocol** - Efficient message serialization with version support
- **HTTP Tunneling** - Forward HTTP requests between client and server
- **Connection Management** - Heartbeat mechanism with graceful disconnection
- **Concurrent Processing** - Handle multiple requests simultaneously
- **Timeout Handling** - Request timeouts and connection deadlines
- **Colored Terminal Output** - Real-time request logging with timestamps
- **Context-Based Cancellation** - Proper resource cleanup and goroutine management

## Usage

### Start the Server

```bash
cd server
go run main.go
```

The server will listen on:
- TCP connections: `localhost:5500`
- HTTP requests: `localhost:8000`

### Start the Client

```bash
cd client
go run . --tunnelHost=localhost --tunnelPort=5500 --localhostPort=3000
```

**Command Line Options:**
- `--tunnelHost`: Server hostname (default: localhost)
- `--tunnelPort`: Server TCP port (default: 5500)  
- `--localhostPort`: Local service port to tunnel (default: 3000)

### Access Your Service

Once connected, you'll see output like:
```
============================================================
🚀 Tunnel Active: http://abc123-def456.localhost:8000
📡 Forwarding to: localhost:3000
============================================================
Request Log:
[14:32:15] ✓ GET    200 /api/users                    (45ms)
[14:32:18] ✓ POST   201 /api/users                    (123ms)
[14:32:22] ✗ GET    404 /api/nonexistent             (12ms)
```

Visit the tunnel URL to access your local service through the public endpoint.

## Implementation Highlights

### Concurrent Request Handling

The server uses goroutines and channels to handle multiple clients and requests simultaneously:

```go
go s.handleTCPRequest(&client)
go s.handleInboundRequests(&client)
```

### Context-Based Resource Management

Proper cleanup and cancellation using Go's context package:

```go
ctx, cancel := context.WithCancel(context.Background())
client := Client{..., ctx, cancel}

// Graceful shutdown
defer client.cancel()
```

### Binary Protocol Implementation

Efficient message serialization with big-endian encoding:

```go
func SerializeMessage(msg TunnelMessage) []byte {
    finalMessage := [][]byte{}
    finalMessage = append(finalMessage, []byte{byte(VERSION), byte(msg.Type)})
    
    messageLength := make([]byte, 8)
    binary.BigEndian.PutUint64(messageLength, uint64(len(msg.Body)))
    
    finalMessage = append(finalMessage, messageLength)
    finalMessage = append(finalMessage, msg.Body)
    
    return bytes.Join(finalMessage, nil)
}
```

## Technical Details

### Error Handling and Timeouts

- **Connection timeouts**: 30-second TCP read deadlines
- **Request timeouts**: 10-second HTTP request processing
- **Graceful disconnection**: Proper channel cleanup and context cancellation

### Heartbeat Mechanism

Maintains connection health with pausable heartbeat system:

```go
case <-ticker.C:
    if !paused {
        client.SendMessage(nil, protocol.HEARTBEAT)
    }
```

### HTTP Request/Response Serialization

Uses Go's `httputil` package for reliable HTTP message handling:

```go
func EncodeRequest(req *http.Request) ([]byte, error) {
    return httputil.DumpRequest(req, true)
}
```

## Dependencies

- `github.com/google/uuid` - Unique client identification

## Project Structure

```
.
├── client/
│   ├── main.go          # Client implementation
│   └── utils.go         # Terminal output utilities
├── server/
│   └── main.go          # Server implementation  
├── protocol/
│   └── protocol.go      # Binary protocol definition
└── colors/
    └── colors.go        # Terminal color utilities
```

## License

MIT License - feel free to use this code for learning and experimentation.