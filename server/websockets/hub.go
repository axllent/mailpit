// Package websockets is used to broadcast messages to connected clients
package websockets

import (
	"encoding/json"
	"sync/atomic"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/segment"
	"github.com/gorilla/websocket"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	Clients map[*Client]bool

	// Inbound messages from the clients.
	Broadcast chan Envelope

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// clientCount is an atomic count of connected clients, safe for concurrent reads.
	clientCount atomic.Int64
}

// WebsocketNotification struct for responses
type WebsocketNotification struct {
	Type string
	Data any
}

// Envelope is a broadcast message together with the segment it is intended for
type Envelope struct {
	// Message is the JSON encoded WebsocketNotification
	Message []byte
	// Segment the message is intended for, segment.All for every connected client
	Segment string
}

// NewHub returns a new hub configuration
func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan Envelope),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

// Run runs the listener
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if _, ok := h.Clients[client]; !ok {
				logger.Log().Debugf("[websocket] client %s connected", client.conn.RemoteAddr().String())
				h.Clients[client] = true
				h.clientCount.Add(1)
			}
		case client := <-h.unregister:
			if _, ok := h.Clients[client]; ok {
				logger.Log().Debugf("[websocket] client %s disconnected", client.conn.RemoteAddr().String())
				delete(h.Clients, client)
				close(client.send)
				h.clientCount.Add(-1)
			}
		case message := <-h.Broadcast:
			prepared, err := websocket.NewPreparedMessage(websocket.TextMessage, message.Message)
			if err != nil {
				logger.Log().Errorf("[websocket] error preparing message: %s", err.Error())
				continue
			}
			for client := range h.Clients {
				if !client.receives(message.Segment) {
					continue
				}
				select {
				case client.send <- prepared:
				default:
					close(client.send)
					delete(h.Clients, client)
					h.clientCount.Add(-1)
				}
			}
		}
	}
}

// Broadcast will spawn a broadcast message to all connected clients.
// Use BroadcastSegment for anything that contains or refers to message data.
func Broadcast(t string, msg any) {
	BroadcastSegment(segment.All, t, msg)
}

// BroadcastSegment will spawn a broadcast message to the clients connected to a segment.
// When segments are disabled the message is sent to all connected clients.
func BroadcastSegment(seg, t string, msg any) {
	if MessageHub == nil || MessageHub.clientCount.Load() == 0 {
		return
	}

	w := WebsocketNotification{}
	w.Type = t
	w.Data = msg
	b, err := json.Marshal(w)

	if err != nil {
		logger.Log().Errorf("[websocket] broadcast received invalid data: %s", err.Error())
		return
	}

	go func() { MessageHub.Broadcast <- Envelope{Message: b, Segment: seg} }()
}

// BroadCastClientError is a wrapper to broadcast client errors to the web UI
func BroadCastClientError(severity, errorType, ip, message string) {
	msg := struct {
		Level   string
		Type    string
		IP      string
		Message string
	}{
		severity,
		errorType,
		ip,
		message,
	}

	// client errors are not linked to a segment, and may contain client IP addresses
	BroadcastSegment(segment.Default, "error", msg)
}
