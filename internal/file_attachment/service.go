package fileattachment

import (
	"context"
	"fmt"
	"mime/multipart"

	"sound-stage-backend/internal/config"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

const defaultFolder = "sound-stage"

type uploaderAPI interface {
	Upload(ctx context.Context, file interface{}, uploadParams uploader.UploadParams) (*uploader.UploadResult, error)
	Destroy(ctx context.Context, params uploader.DestroyParams) (*uploader.DestroyResult, error)
}

type Repository interface {
	Create(att *FileAttachment) error
	FindByID(id uint) (*FileAttachment, error)
	Save(att *FileAttachment) error
	Delete(att *FileAttachment) error
}

type UploadFileParams struct {
	OwnerType string
	OwnerID   uint
	Context   Context
	Folder    *string
	File      *multipart.FileHeader
}

type Service struct {
	uploader uploaderAPI
	repo     Repository
}

func NewService(cfg config.CloudinaryConfig, repo Repository) *Service {
	s := &Service{
		repo: repo,
	}
	if cld, err := cloudinary.NewFromParams(cfg.CloudName, cfg.APIKey, cfg.APISecret); err == nil {
		s.uploader = &cld.Upload
	}
	return s
}

func (s *Service) UploadOrReplaceFile(ctx context.Context, existing *FileAttachment, in UploadFileParams) (*FileAttachment, error) {
	if in.File == nil {
		return existing, nil
	}
	if existing != nil {
		return s.ReplaceFile(ctx, existing.ID, in.File)
	}
	return s.UploadFile(ctx, in)
}

func (s *Service) UploadFile(ctx context.Context, in UploadFileParams) (*FileAttachment, error) {
	if s.uploader == nil {
		return nil, fmt.Errorf("file attachment: cloudinary is not configured")
	}
	if in.File == nil {
		return nil, fmt.Errorf("file attachment: file is required")
	}

	file, err := in.File.Open()
	if err != nil {
		return nil, fmt.Errorf("file attachment: open file: %w", err)
	}
	defer file.Close()

	folder := defaultFolder
	if in.Folder != nil {
		folder = *in.Folder
	}

	res, err := s.uploader.Upload(ctx, file, uploader.UploadParams{
		Folder:       folder,
		ResourceType: "auto",
	})
	if err != nil {
		return nil, fmt.Errorf("file attachment: upload: %w", err)
	}

	att := attachmentFromUpload(in, res)
	if err := s.repo.Create(att); err != nil {
		return nil, fmt.Errorf("file attachment: save attachment: %w", err)
	}
	return att, nil
}

func (s *Service) ReplaceFile(ctx context.Context, attachmentID uint, file *multipart.FileHeader) (*FileAttachment, error) {
	if s.uploader == nil {
		return nil, fmt.Errorf("file attachment: cloudinary is not configured")
	}
	if file == nil {
		return nil, fmt.Errorf("file attachment: file is required")
	}

	att, err := s.repo.FindByID(attachmentID)
	if err != nil {
		return nil, err
	}

	r, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("file attachment: open file: %w", err)
	}
	defer r.Close()

	res, err := s.uploader.Upload(ctx, r, uploader.UploadParams{
		PublicID:     att.PublicID,
		ResourceType: "auto",
		Overwrite:    boolPtr(true),
		Invalidate:   boolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("file attachment: replace: %w", err)
	}

	applyUploadResult(att, res)
	if err := s.repo.Save(att); err != nil {
		return nil, fmt.Errorf("file attachment: save attachment: %w", err)
	}
	return att, nil
}

func (s *Service) DeleteFile(ctx context.Context, attachmentID uint) error {
	if s.uploader == nil {
		return fmt.Errorf("file attachment: cloudinary is not configured")
	}

	att, err := s.repo.FindByID(attachmentID)
	if err != nil {
		return err
	}

	if _, err := s.uploader.Destroy(ctx, uploader.DestroyParams{
		PublicID:     att.PublicID,
		ResourceType: att.ResourceType,
		Invalidate:   boolPtr(true),
	}); err != nil {
		return fmt.Errorf("file attachment: destroy: %w", err)
	}

	return s.repo.Delete(att)
}

func attachmentFromUpload(in UploadFileParams, res *uploader.UploadResult) *FileAttachment {
	att := &FileAttachment{
		OwnerType:    in.OwnerType,
		OwnerID:      in.OwnerID,
		Context:      in.Context,
		PublicID:     res.PublicID,
		ResourceType: res.ResourceType,
	}
	applyUploadResult(att, res)
	return att
}

func applyUploadResult(att *FileAttachment, res *uploader.UploadResult) {
	att.PublicID = res.PublicID
	att.URL = res.SecureURL
	if att.URL == "" {
		att.URL = res.URL
	}
	att.ResourceType = res.ResourceType
	att.Format = res.Format
	att.Bytes = int64(res.Bytes)
	att.Width = res.Width
	att.Height = res.Height
}

func boolPtr(b bool) *bool { return &b }
