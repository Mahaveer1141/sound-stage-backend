package room

import (
	"sound-stage-backend/internal/category"
	roomuser "sound-stage-backend/internal/room_user"
	"sound-stage-backend/internal/tag"
)

func BuildRoomResponse(room *Room, viewer *roomuser.RoomUser) RoomResponse {
	cats := make([]category.CategoryResponse, len(room.Categories))
	for i := range room.Categories {
		cats[i] = category.BuildCategoryResponse(&room.Categories[i])
	}

	tags := make([]tag.TagResponse, len(room.Tags))
	for i := range room.Tags {
		tags[i] = tag.BuildTagResponse(&room.Tags[i])
	}

	var privateCode *string
	if viewer != nil && viewer.IsAdmin() {
		privateCode = room.PrivateCode
	}

	return RoomResponse{
		ID:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		Type:        room.Type,
		PrivateCode: privateCode,
		Categories:  cats,
		Tags:        tags,
	}
}

func BuildRoomListResponse(rooms []Room, viewer *roomuser.RoomUser) []RoomResponse {
	responses := make([]RoomResponse, len(rooms))
	for i := range rooms {
		responses[i] = BuildRoomResponse(&rooms[i], viewer)
	}
	return responses
}
