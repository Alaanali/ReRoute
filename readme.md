# ReRoute

A lightweight TCP tunnel implementation in Go that allows you to expose local services through a public endpoint. Built as a weekend project for learning and experimentation.

## Architecture

ReRoute enables secure tunneling of HTTP traffic through a persistent TCP connection:

```
    Internet                    Local Network
        │                           │
        ▼                           ▼
┌───────────────┐               ┌───────────────┐
│ ReRoute Server│◄─────TCP──────►│ ReRoute Client│
│   (Public)    │               │   (Local)     │
└───────┬───────┘               └───────┬───────┘
        │                               │
        │ HTTP Requests                 │ HTTP Requests
        │ (UUID Correlated)             │ (to localhost)
        ▼                               ▼
┌───────────────┐               ┌───────────────┐
│  Web Browser  │               │ Local Service │
│               │               │ localhost:3000│
└───────────────┘               └───────────────┘
```

## Protocol Design

ReRoute uses a custom binary protocol with UUID-based message correlation:

```
+----------+-------------+------------------+--------------+--------------+
| Version  | MessageType | Message ID (UUID)| Body Length  |     Body     |
| 1 byte   |   1 byte    |    16 bytes      |   8 bytes    |   Variable   |
+----------+-------------+------------------+--------------+--------------+
```

**Message Types:**
- `REQUEST` - HTTP request forwarding (with UUID correlation)
- `RESPONSE` - HTTP response forwarding (matched by UUID)
- `HEARTBEAT` - Connection health check
- `CONNECTION_REQUEST` - Initial client connection
- `CONNECTION_ACCEPTED` - Server acknowledgment with client ID
- `DISCONNECT` - Graceful disconnection
- `ERROR` - Error responses (correlated to original request)

## Usage

### Start the Server

```bash
cd server
go run main.go
```

Server endpoints:
- TCP tunnel connections: `localhost:5500`
- HTTP request handling: `localhost:8000`

### Start the Client

```bash
cd client
go run . --tunnelHost=localhost --tunnelPort=5500 --localhostPort=3000
```

**Configuration Options:**
- `--tunnelHost`: Server hostname (default: localhost)
- `--tunnelPort`: Server TCP port (default: 5500)
- `--localhostPort`: Local service port to tunnel (default: 3000)

### Client Output

```bash
============================================================
🚀 Tunnel Active: http://abc123-def456.localhost:8000
📡 Forwarding to: localhost:3000
============================================================
Request Log:
[14:32:15] ✓ GET    200 /api/users                    (45ms)
[14:32:18] ✓ POST   201 /api/users/new                (123ms)
[14:32:22] ✗ GET    404 /api/nonexistent              (12ms)
[14:32:25] ✓ PUT    200 /api/users/123                (67ms)
```

## Implementation Details

### UUID-Based Request Correlation

Each message includes a unique identifier for proper request-response matching:

```go
type TunnelMessage struct {
    Type uint8
    Body []byte
    Id   uuid.UUID  // Enables concurrent request processing
}
```

### Concurrent Request Architecture

The server maintains per-request channels for isolated processing:

```go
type Client struct {
    requests map[uuid.UUID]chan protocol.TunnelMessage
    mu       sync.Mutex  // Protects concurrent map access
}

// Each HTTP request gets its own response channel
responseChan := make(chan protocol.TunnelMessage, 1)
messageId := uuid.New()
client.requests[messageId] = responseChan
client.SendMessage(encodedRequest, protocol.REQUEST, messageId)
```

### Binary Protocol Implementation

Efficient serialization with UUID support:

```go
func SerializeMessage(msg TunnelMessage) []byte {
    finalMessage := [][]byte{}
    finalMessage = append(finalMessage, []byte{byte(VERSION), byte(msg.Type)})
    finalMessage = append(finalMessage, msg.Id[:])  // 16-byte UUID
    
    messageLength := make([]byte, 8)
    binary.BigEndian.PutUint64(messageLength, uint64(len(msg.Body)))
    
    finalMessage = append(finalMessage, messageLength)
    finalMessage = append(finalMessage, msg.Body)
    
    return bytes.Join(finalMessage, nil)
}
```

## Testing


```bash
cd protocol
go test -v
```


## Dependencies

- `github.com/google/uuid` - UUID generation and parsing

## Project Structure

```
.
├── client/
│   ├── main.go          # Client with UUID correlation
│   └── utils.go         # Colored terminal output
├── server/
│   └── main.go          # Server with concurrent request handling
├── protocol/
│   ├── protocol.go      # Binary protocol with UUID support
│   └── protocol_test.go # Protocol serialization tests
└── colors/
    └── colors.go        # Terminal color utilities
```

## License

MIT License - Educational and experimental use encouraged.