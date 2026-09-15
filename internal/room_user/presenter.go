package roomuser

import (
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	user "sound-stage-backend/internal/user"
	"time"
)

func BuildRoomUserResponse(ru *RoomUser, viewerID uint) RoomUserResponse {
	showEmail := ru.UserID == viewerID
	return RoomUserResponse{
		ID:           ru.ID,
		User:         user.BuildUserResponse(&ru.User, showEmail),
		Role:         role.BuildRoleResponse(&ru.Role),
		LastJoinedAt: ru.LastJoinedAt.UTC().Format(listopts.CursorTimeLayout),
		LastLeftAt:   ru.LastLeftAt.Format(time.RFC3339),
		IsOnline:     ru.IsOnline,
		IsMuted:      ru.IsMuted,
		IsHandRaised: ru.IsHandRaised,
		CanManage:    ru.CanManage(),
		CanSpeak:     ru.CanSpeak(),
		IsAdmin:      ru.IsAdmin(),
		IsOwner:      ru.IsOwner(),
	}
}

func BuildRoomUserListResponse(roomUsers []RoomUser, viewerID uint) []RoomUserResponse {
	responses := make([]RoomUserResponse, len(roomUsers))
	for i := range roomUsers {
		responses[i] = BuildRoomUserResponse(&roomUsers[i], viewerID)
	}
	return responses
}

func BuildBlockedUserListResponse(roomUsers []RoomUser) []user.UserResponse {
	responses := make([]user.UserResponse, len(roomUsers))
	for i := range roomUsers {
		responses[i] = user.BuildUserResponse(&roomUsers[i].User, false)
	}
	return responses
}
