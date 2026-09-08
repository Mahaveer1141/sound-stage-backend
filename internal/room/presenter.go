package room

import (
	"sound-stage-backend/internal/category"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/tag"
)

func BuildRoomResponse(room *Room, viewer *RoomViewer) RoomResponse {
	cats := make([]category.CategoryResponse, len(room.Categories))
	for i := range room.Categories {
		cats[i] = category.BuildCategoryResponse(&room.Categories[i])
	}

	tags := make([]tag.TagResponse, len(room.Tags))
	for i := range room.Tags {
		tags[i] = tag.BuildTagResponse(&room.Tags[i])
	}

	var privateCode *string
	var isRoomUser, isFavourited bool
	if viewer != nil {
		isRoomUser = viewer.RoomUser != nil
		isFavourited = viewer.IsFavourited
		if isRoomUser && viewer.RoomUser.IsAdmin() {
			privateCode = room.PrivateCode
		}
	}

	return RoomResponse{
		ID:            room.ID,
		Name:          room.Name,
		Description:   room.Description,
		Type:          room.Type,
		IsChatEnabled: room.IsChatEnabled,
		PrivateCode:   privateCode,
		CoverImage:    httpx.BuildFileAttachmentResponse(room.CoverImage),
		LogoImage:     httpx.BuildFileAttachmentResponse(room.LogoImage),
		Categories:    cats,
		Tags:          tags,
		IsRoomUser:    isRoomUser,
		IsFavourited:  isFavourited,
		TotalUsers:    room.TotalUsers,
		LiveUsers:     room.LiveUsers,
	}
}

func BuildRoomListResponse(rooms []Room, viewers map[uint]*RoomViewer) []RoomResponse {
	responses := make([]RoomResponse, len(rooms))
	for i := range rooms {
		responses[i] = BuildRoomResponse(&rooms[i], viewers[rooms[i].ID])
	}
	return responses
}
