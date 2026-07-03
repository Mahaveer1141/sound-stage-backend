package ws

type roomMessage struct {
	roomID uint
	data   []byte
}

type clientMessage struct {
	client *Client
	data   []byte
}

type roomMethod struct {
	roomID uint
	fn     func(*Client)
}

type Hub struct {
	rooms      map[uint]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan roomMessage
	direct     chan clientMessage
	roomMethod chan roomMethod
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[uint]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan roomMessage),
		direct:     make(chan clientMessage),
		roomMethod: make(chan roomMethod),
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
			cm.client.Send(cm.data)
		case rm := <-h.roomMethod:
			for c := range h.rooms[rm.roomID] {
				rm.fn(c)
			}
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID uint, eventName EventName, payload any) {
	data, err := Encode(eventName, payload)
	if err != nil {
		return
	}
	h.broadcast <- roomMessage{roomID: roomID, data: data}
}

func (h *Hub) ForEachClientInRoom(roomID uint, fn func(*Client)) {
	h.roomMethod <- roomMethod{roomID: roomID, fn: fn}
}

func (h *Hub) SendToClient(c *Client, eventName EventName, payload any) {
	data, err := Encode(eventName, payload)
	if err != nil {
		return
	}
	h.direct <- clientMessage{client: c, data: data}
}
