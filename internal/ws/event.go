package ws

import "encoding/json"

type EventName string

const (
	EventJoinRoom      EventName = "join_room"
	EventLeaveRoom     EventName = "leave_room"
	EventUserKickedOut EventName = "user_kicked_out"
	EventRoomDeleted   EventName = "room_deleted"

	EventError EventName = "error"

	EventWebRTCOffer     EventName = "webrtc_offer"
	EventWebRTCCandidate EventName = "webrtc_candidate"
	EventWebRTCAddTrack  EventName = "webrtc_add_track"
	EventWebRTCAnswer    EventName = "webrtc_answer"

	EventUserRoleUpdated    EventName = "user_role_updated"
	EventChatEnabledUpdated EventName = "chat_enabled_updated"

	EventSetMuted      EventName = "set_muted"
	EventSetHandRaised EventName = "set_hand_raised"

	EventChatMessage EventName = "chat_message"
)

type Event struct {
	Name    EventName       `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

func Encode(eventName EventName, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	event := Event{
		Name:    eventName,
		Payload: data,
	}
	return json.Marshal(event)
}
