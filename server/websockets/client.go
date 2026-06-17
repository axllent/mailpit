// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websockets

import (
	"net/http"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var (
	// MessageHub global
	MessageHub *Hub
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	EnableCompression: !config.DisableHTTPCompression,
	CheckOrigin: func(_ *http.Request) bool {
		// origin is checked via server.go's CORS settings
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan *websocket.PreparedMessage
}

// ReadPump is used here solely to monitor the connection, not to actually receive messages.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log().Errorf("[websocket] error: %v", err.Error())
			}
			break
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WritePreparedMessage(message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	if auth.UICredentials != nil {
		user, pass, ok := r.BasicAuth()

		if !ok {
			basicAuthResponse(w)
			return
		}

		if !auth.UICredentials.Match(user, pass) {
			basicAuthResponse(w)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log().Errorf("[websocket] %s", err.Error())
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan *websocket.PreparedMessage, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.
	go client.readPump()
	go client.writePump()
}

// BasicAuthResponse returns an basic auth response to the browser
func basicAuthResponse(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Login"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized.\n"))
}
