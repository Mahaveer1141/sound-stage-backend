package fileattachment

import "sound-stage-backend/internal/model"

type Context string

const (
	ContextProfilePicture Context = "profile_picture"
	ContextRoomCover      Context = "room_cover"
	ContextRoomLogo       Context = "room_logo"
)

type FileAttachment struct {
	model.BaseModel
	OwnerType    string  `gorm:"not null;index:idx_file_attachment_owner"`
	OwnerID      uint    `gorm:"not null;index:idx_file_attachment_owner"`
	Context      Context `gorm:"not null;index:idx_file_attachment_owner"`
	PublicID     string  `gorm:"not null;uniqueIndex:idx_file_attachment_unique"`
	URL          string  `gorm:"not null"`
	ResourceType string  `gorm:"not null"`
	Bytes        int64   `gorm:"not null"`
	Format       string  `gorm:"not null"`
	Width        int
	Height       int
}
