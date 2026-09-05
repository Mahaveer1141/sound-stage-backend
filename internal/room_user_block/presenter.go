package roomuserblock

import (
	"time"

	"sound-stage-backend/internal/user"
)

func BuildRoomUserBlockResponse(block *RoomUserBlock) RoomUserBlockResponse {
	return RoomUserBlockResponse{
		ID:        block.ID,
		User:      user.BuildUserResponse(&block.User, false),
		BlockedBy: user.BuildUserResponse(&block.BlockedBy, false),
		CreatedAt: block.CreatedAt.Format(time.RFC3339),
	}
}

func BuildRoomUserBlockListResponse(blocks []RoomUserBlock) []RoomUserBlockResponse {
	responses := make([]RoomUserBlockResponse, len(blocks))
	for i := range blocks {
		responses[i] = BuildRoomUserBlockResponse(&blocks[i])
	}
	return responses
}
