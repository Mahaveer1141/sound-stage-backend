package ws

import "log/slog"

type roomMessage struct {
	roomID uint
	data   []byte
}

type clientMessage struct {
	client *Client
	data   []byte
}

type clientsQuery struct {
	roomID uint
	reply  chan []*Client
}

type ErrorPayload struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

type Hub struct {
	rooms      map[uint]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan roomMessage
	direct     chan clientMessage
	clients    chan clientsQuery
	logger     *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		rooms:      make(map[uint]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan roomMessage),
		direct:     make(chan clientMessage),
		clients:    make(chan clientsQuery),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			if h.rooms[c.RoomID] == nil {
				h.rooms[c.RoomID] = make(map[*Client]struct{})
			}
			h.rooms[c.RoomID][c] = struct{}{}
		case c := <-h.unregister:
			if _, exists := h.rooms[c.RoomID][c]; exists {
				close(c.send)
				delete(h.rooms[c.RoomID], c)
				if len(h.rooms[c.RoomID]) == 0 {
					delete(h.rooms, c.RoomID)
				}
			}
		case m := <-h.broadcast:
			for c := range h.rooms[m.roomID] {
				c.Send(m.data)
			}
		case cm := <-h.direct:
			if _, exists := h.rooms[cm.client.RoomID][cm.client]; exists {
				cm.client.Send(cm.data)
			}
		case q := <-h.clients:
			clients := make([]*Client, 0, len(h.rooms[q.roomID]))
			for c := range h.rooms[q.roomID] {
				clients = append(clients, c)
			}
			q.reply <- clients
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID uint, eventName EventName, payload any) {
	data, err := Encode(eventName, payload)
	if err != nil {
		h.logger.Error("Error encoding json for websocket", slog.Any("error", err))
		return
	}
	h.broadcast <- roomMessage{roomID: roomID, data: data}
}

func (h *Hub) ClientsInRoom(roomID uint) []*Client {
	reply := make(chan []*Client, 1)
	h.clients <- clientsQuery{roomID: roomID, reply: reply}
	return <-reply
}

func (h *Hub) SendToClient(c *Client, eventName EventName, payload any) {
	data, err := Encode(eventName, payload)
	if err != nil {
		h.logger.Error("Error encoding json for websocket", slog.Any("error", err))
		return
	}
	h.direct <- clientMessage{client: c, data: data}
}

func (h *Hub) ErrorToClient(c *Client, message string, code int) {
	ep := ErrorPayload{Message: message, Code: code}
	data, err := Encode(EventError, ep)
	if err != nil {
		h.logger.Error("Error encoding json for websocket", slog.Any("error", err))
		return
	}
	h.direct <- clientMessage{client: c, data: data}
}
