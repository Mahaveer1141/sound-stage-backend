package ws

import "encoding/json"

type EventName string

var (
	EventJoinRoom  EventName = "join_room"
	EventLeaveRoom EventName = "leave_room"

	EventWebRTCOffer     EventName = "webrtc_offer"
	EventWebRTCCandidate EventName = "webrtc_candidate"
	EventWebRTCAddTrack  EventName = "webrtc_add_track"
	EventWebRTCAnswer    EventName = "webrtc_answer"
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
