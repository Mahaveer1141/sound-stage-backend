package chatmessage

import (
	"time"

	"sound-stage-backend/internal/user"
)

func BuildChatMessageResponse(m *ChatMessage) ChatMessageResponse {
	var ur *user.UserResponse
	if m.User.ID != 0 {
		r := user.BuildUserResponse(&m.User, false)
		ur = &r
	}

	return ChatMessageResponse{
		ID:        m.ID,
		Content:   m.Content,
		IsPinned:  m.IsPinned,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
		User:      ur,
	}
}

func BuildChatMessageListResponse(messages []ChatMessage) []ChatMessageResponse {
	res := make([]ChatMessageResponse, 0, len(messages))
	for i := range messages {
		res = append(res, BuildChatMessageResponse(&messages[i]))
	}
	return res
}
