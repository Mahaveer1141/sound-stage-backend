package chatmessage

import (
	"sound-stage-backend/internal/pkg/listopts"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(input *CreateChatMessageParams) (*ChatMessage, error) {
	msg := ChatMessage{
		RoomID:   input.RoomID,
		UserID:   input.UserID,
		Content:  input.Content,
		IsPinned: input.IsPinned,
	}

	if err := r.db.Create(&msg).Error; err != nil {
		return nil, err
	}

	if err := r.db.Preload("User").First(&msg, msg.ID).Error; err != nil {
		return nil, err
	}

	return &msg, nil
}

func (r *Repo) FindByID(id uint) (*ChatMessage, error) {
	var msg ChatMessage

	if err := r.db.Preload("User").First(&msg, id).Error; err != nil {
		return nil, err
	}

	return &msg, nil
}

func (r *Repo) SetPinned(id uint, pinned bool) error {
	return r.db.Model(&ChatMessage{}).
		Where("id = ?", id).
		Update("is_pinned", pinned).Error
}

func (r *Repo) List(filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, error) {
	var messages []ChatMessage

	err := r.db.Preload("User").
		Scopes(Filters(filter), Sort(listopts.Sort{}), p.Scope()).
		Find(&messages).Error

	return messages, err
}

func (r *Repo) Count(filter ChatMessageFilter) (int64, error) {
	var count int64

	err := r.db.Model(&ChatMessage{}).
		Scopes(Filters(filter)).
		Count(&count).Error

	return count, err
}
