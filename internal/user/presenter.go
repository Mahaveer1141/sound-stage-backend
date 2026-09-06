package user

import "sound-stage-backend/internal/pkg/httpx"

func BuildUserResponse(u *User, includeEmail bool) UserResponse {
	email := ""
	if includeEmail {
		email = u.Email
	}

	return UserResponse{
		ID:             u.ID,
		Email:          email,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		FullName:       u.FullName(),
		ProfilePicture: httpx.BuildFileAttachmentResponse(u.ProfilePicture),
	}
}
