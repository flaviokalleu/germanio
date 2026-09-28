package servidor

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// WSMessage is a WebSocket broadcast message.
type WSMessage struct {
	Type    string `json:"type"`
	Model   string `json:"model"`
	ID      int64  `json:"id"`
	Session string `json:"session,omitempty"`
	Data    any    `json:"data"`
}

// WSHub manages WebSocket connections.
type WSHub struct {
	mu           sync.RWMutex
	clients      map[*WSConn]bool
	OnConnect    func(count int)
	OnDisconnect func(count int)
}

// WSConn is a single WebSocket connection.
type WSConn struct {
	hub  *WSHub
	w    http.ResponseWriter
	done chan struct{}
	send chan []byte
}

// NewWSHub creates a new WebSocket hub.
func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*WSConn]bool),
	}
}

// sameOrigin: no Origin (not a browser) or an Origin whose host is the one
// the request was sent to.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

// Broadcast sends a message to all connected clients.
func (h *WSHub) Broadcast(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	frame := wsFrame(data)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- frame:
		default:
			// Client buffer full, skip
		}
	}
}

// HandleWS upgrades an HTTP connection to WebSocket.
func (h *WSHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	// WebSocket handshake
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "Not a WebSocket request", http.StatusBadRequest)
		return
	}

	// A browser always sends Origin: a socket opened by another site would
	// ride on this person's session (cross-site WebSocket hijacking).
	if !sameOrigin(r) {
		http.Error(w, "origem não permitida", http.StatusForbidden)
		return
	}

	acceptKey := computeAcceptKey(key)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket not supported", http.StatusInternalServerError)
		return
	}

	conn, buf, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send upgrade response
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"
	buf.WriteString(response)
	buf.Flush()

	client := &WSConn{
		hub:  h,
		done: make(chan struct{}),
		send: make(chan []byte, 64),
	}

	h.mu.Lock()
	h.clients[client] = true
	count := len(h.clients)
	h.mu.Unlock()
	if h.OnConnect != nil {
		go h.OnConnect(count)
	}

	// Writer goroutine
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, client)
			count := len(h.clients)
			h.mu.Unlock()
			if h.OnDisconnect != nil {
				go h.OnDisconnect(count)
			}
			conn.Close()
		}()

		for {
			select {
			case msg, ok := <-client.send:
				if !ok {
					return
				}
				_, err := conn.Write(msg)
				if err != nil {
					return
				}
			case <-client.done:
				return
			}
		}
	}()

	// Reader goroutine (reads and discards, detects close)
	go func() {
		buf := make([]byte, 512)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				close(client.done)
				return
			}
		}
	}()
}

// computeAcceptKey generates the Sec-WebSocket-Accept header value.
func computeAcceptKey(key string) string {
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magic))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// wsFrame creates a WebSocket text frame.
func wsFrame(payload []byte) []byte {
	length := len(payload)
	var frame []byte

	// First byte: FIN + text opcode
	frame = append(frame, 0x81)

	// Length encoding
	if length <= 125 {
		frame = append(frame, byte(length))
	} else if length <= 65535 {
		frame = append(frame, 126)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(length))
		frame = append(frame, buf...)
	} else {
		frame = append(frame, 127)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(length))
		frame = append(frame, buf...)
	}

	frame = append(frame, payload...)
	return frame
}

// Count returns the number of connected clients.
func (h *WSHub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

var _ = fmt.Sprintf // keep fmt import
