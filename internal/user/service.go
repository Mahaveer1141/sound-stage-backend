package user

import (
	"context"

	fileattachment "sound-stage-backend/internal/file_attachment"
	"sound-stage-backend/internal/pkg/httpx"
)

type repository interface {
	Create(input *CreateUserParams) (*User, error)
	FindByID(id uint) (*User, error)
	FindByEmail(email string) (*User, error)
	UpdateLastLoginAt(id uint) error
	Save(user *User) error
}

type fileAttachmentService interface {
	UploadOrReplaceFile(ctx context.Context, existing *fileattachment.FileAttachment,
		in fileattachment.UploadFileParams) (*fileattachment.FileAttachment, error)
}

type Service struct {
	repo repository
	file fileAttachmentService
}

func NewService(r repository, file fileAttachmentService) *Service {
	return &Service{repo: r, file: file}
}

func (s *Service) FindByID(userId uint) (*User, error) {
	return s.repo.FindByID(userId)
}

func (s *Service) FindByEmail(email string) (*User, error) {
	return s.repo.FindByEmail(email)
}

func (s *Service) Create(input *CreateUserParams) (*User, error) {
	user, err := s.repo.Create(input)
	if err != nil {
		return nil, err
	}

	if input.ProfilePicture != nil {
		att, err := s.file.UploadOrReplaceFile(context.Background(), nil, fileattachment.UploadFileParams{
			OwnerType: user.TableName(),
			OwnerID:   user.ID,
			Context:   fileattachment.ContextProfilePicture,
			File:      input.ProfilePicture,
		})
		if err != nil {
			return nil, err
		}
		user.ProfilePicture = att
	}

	return user, nil
}

func (s *Service) UpdateLastLoginAt(id uint) error {
	return s.repo.UpdateLastLoginAt(id)
}

func (s *Service) UpdateProfile(id uint, input *UpdateUserParams) (*User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, httpx.ErrRecordNotFound
	}

	user.FirstName = input.FirstName
	user.LastName = &input.LastName

	if input.ProfilePicture != nil {
		att, err := s.file.UploadOrReplaceFile(context.Background(), user.ProfilePicture, fileattachment.UploadFileParams{
			OwnerType: user.TableName(),
			OwnerID:   user.ID,
			Context:   fileattachment.ContextProfilePicture,
			File:      input.ProfilePicture,
		})
		if err != nil {
			return nil, err
		}
		user.ProfilePicture = att
	}

	if err := s.repo.Save(user); err != nil {
		return nil, err
	}
	return user, nil
}
