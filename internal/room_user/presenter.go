package roomuser

import (
	"sound-stage-backend/internal/role"
	user "sound-stage-backend/internal/user"
	"time"
)

func BuildRoomUserResponse(ru *RoomUser, viewerID uint, viewerIsAdmin bool) RoomUserResponse {
	showEmail := viewerIsAdmin || ru.UserID == viewerID
	return RoomUserResponse{
		ID:           ru.ID,
		User:         user.BuildUserResponse(&ru.User, showEmail),
		Role:         role.BuildRoleResponse(&ru.Role),
		LastJoinedAt: ru.LastJoinedAt.Format(time.RFC3339),
		LastLeftAt:   ru.LastLeftAt.Format(time.RFC3339),
		IsOnline:     ru.IsOnline,
		CanManage:    ru.CanManage(),
		CanSpeak:     ru.CanSpeak(),
		IsAdmin:      ru.IsAdmin(),
		IsOwner:      ru.IsOwner(),
	}
}

func BuildRoomUserListResponse(roomUsers []RoomUser, viewerID uint, viewerIsAdmin bool) []RoomUserResponse {
	responses := make([]RoomUserResponse, len(roomUsers))
	for i := range roomUsers {
		responses[i] = BuildRoomUserResponse(&roomUsers[i], viewerID, viewerIsAdmin)
	}
	return responses
}
