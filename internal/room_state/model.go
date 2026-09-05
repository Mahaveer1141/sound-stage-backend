package roomstate

type ParticipantState struct {
	UserID       uint `json:"userId"`
	IsMuted      bool `json:"isMuted"`
	IsHandRaised bool `json:"isHandRaised"`
}
