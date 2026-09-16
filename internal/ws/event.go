package ws

import "encoding/json"

type EventName string

const (
	EventJoinStream EventName = "join_stream"

	EventJoinRoom        EventName = "join_room"
	EventUserRoleUpdated EventName = "user_role_updated"

	EventSetMuted      EventName = "set_muted"
	EventSetHandRaised EventName = "set_hand_raised"

	EventLeaveRoom     EventName = "leave_room"
	EventUserKickedOut EventName = "user_kicked_out"

	EventRoomDeleted        EventName = "room_deleted"
	EventChatEnabledUpdated EventName = "chat_enabled_updated"

	EventChatMessage EventName = "chat_message"

	EventWebRTCOffer     EventName = "webrtc_offer"
	EventWebRTCCandidate EventName = "webrtc_candidate"
	EventWebRTCAddTrack  EventName = "webrtc_add_track"
	EventWebRTCAnswer    EventName = "webrtc_answer"

	EventError EventName = "error"
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
