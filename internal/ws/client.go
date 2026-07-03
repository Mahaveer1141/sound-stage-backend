package ws

import (
	"encoding/json"
	"sound-stage-backend/internal/config"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type Client struct {
	RoomID uint
	UserID uint
	hub    *Hub
	send   chan []byte
	conn   *websocket.Conn
	PC     *webrtc.PeerConnection
	cfg    *config.Config
}

func newClient(roomID, userID uint, hub *Hub, conn *websocket.Conn, cfg *config.Config) *Client {
	return &Client{
		RoomID: roomID,
		UserID: userID,
		hub:    hub,
		send:   make(chan []byte, 256),
		conn:   conn,
		cfg:    cfg,
	}
}

func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) readPump(handleEvent func(c *Client, evt Event)) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(c.cfg.WebSocket.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.cfg.WebSocket.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.cfg.WebSocket.PongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var evt Event
		if err := json.Unmarshal(msg, &evt); err != nil {
			return
		}
		handleEvent(c, evt)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(c.cfg.WebSocket.PingInterval)
	defer func() {
		c.conn.Close()
		ticker.Stop()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WebSocket.WriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WebSocket.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
