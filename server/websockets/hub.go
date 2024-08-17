// Package websockets is used to broadcast messages to connected clients
package websockets

import (
	"encoding/json"
	"time"

	"github.com/axllent/mailpit/internal/logger"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	Clients map[*Client]bool

	// Inbound messages from the clients.
	Broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

// WebsocketNotification struct for responses
type WebsocketNotification struct {
	Type string
	Data interface{}
}

// NewHub returns a new hub configuration
func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
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
			}
		case client := <-h.unregister:
			if _, ok := h.Clients[client]; ok {
				logger.Log().Debugf("[websocket] client %s disconnected", client.conn.RemoteAddr().String())
				delete(h.Clients, client)
				close(client.send)
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

// Broadcast will spawn a broadcast message to all connected clients
func Broadcast(t string, msg interface{}) {
	if MessageHub == nil || len(MessageHub.Clients) == 0 {
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

	// add a very small delay to prevent broadcasts from being interpreted
	// as a multi-line messages (eg: storage.DeleteMessages() which can send a very quick series)
	time.Sleep(time.Millisecond)

	go func() { MessageHub.Broadcast <- b }()
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

	Broadcast("error", msg)
}
