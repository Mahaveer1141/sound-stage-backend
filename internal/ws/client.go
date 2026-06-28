package ws

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type Client struct {
	RoomID uint
	UserID uint
	hub    *Hub
	send   chan []byte
	conn   *websocket.Conn
}

func newClient(roomID, userID uint, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		RoomID: roomID,
		UserID: userID,
		hub:    hub,
		send:   make(chan []byte, 256),
		conn:   conn,
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
	defer func() {
		c.conn.Close()
	}()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
