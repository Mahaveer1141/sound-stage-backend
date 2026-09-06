package fileattachment

import (
	"bytes"
	"context"
	"mime/multipart"
	"testing"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"sound-stage-backend/internal/config"
)

type mockUploader struct{ mock.Mock }

func (m *mockUploader) Upload(ctx context.Context, file interface{}, params uploader.UploadParams) (*uploader.UploadResult, error) {
	args := m.Called(ctx, file, params)
	res, _ := args.Get(0).(*uploader.UploadResult)
	return res, args.Error(1)
}
func (m *mockUploader) Destroy(ctx context.Context, params uploader.DestroyParams) (*uploader.DestroyResult, error) {
	args := m.Called(ctx, params)
	res, _ := args.Get(0).(*uploader.DestroyResult)
	return res, args.Error(1)
}

type mockAttachmentRepo struct{ mock.Mock }

func (m *mockAttachmentRepo) Create(att *FileAttachment) error {
	return m.Called(att).Error(0)
}
func (m *mockAttachmentRepo) FindByID(id uint) (*FileAttachment, error) {
	args := m.Called(id)
	att, _ := args.Get(0).(*FileAttachment)
	return att, args.Error(1)
}
func (m *mockAttachmentRepo) Save(att *FileAttachment) error {
	return m.Called(att).Error(0)
}
func (m *mockAttachmentRepo) Delete(att *FileAttachment) error {
	return m.Called(att).Error(0)
}

type harness struct {
	uploader *mockUploader
	repo     *mockAttachmentRepo
	svc      *Service
}

func newHarness() *harness {
	up := new(mockUploader)
	repo := new(mockAttachmentRepo)
	svc := NewService(config.CloudinaryConfig{CloudName: "demo", APIKey: "key", APISecret: "secret"}, repo)
	svc.uploader = up
	return &harness{uploader: up, repo: repo, svc: svc}
}

func uploadResult(publicID string) *uploader.UploadResult {
	return &uploader.UploadResult{
		PublicID:     publicID,
		SecureURL:    "https://res.cloudinary.com/demo/image/upload/" + publicID + ".png",
		ResourceType: "image",
		Format:       "png",
		Bytes:        1024,
		Width:        100,
		Height:       100,
	}
}

func multipartFileHeader(filename string) *multipart.FileHeader {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		panic(err)
	}
	if _, err := fw.Write([]byte("fake-image")); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}

	r := multipart.NewReader(&b, w.Boundary())
	form, err := r.ReadForm(1024)
	if err != nil {
		panic(err)
	}
	files := form.File["file"]
	if len(files) == 0 {
		panic("no file")
	}
	return files[0]
}

func TestService_UploadFile(t *testing.T) {
	t.Run("success: uploads to cloudinary and saves the attachment", func(t *testing.T) {
		h := newHarness()
		in := UploadFileParams{OwnerType: "user", OwnerID: 7, Context: ContextProfilePicture, File: multipartFileHeader("test.png")}

		h.uploader.On("Upload", mock.Anything, mock.Anything, mock.MatchedBy(func(p uploader.UploadParams) bool {
			return p.ResourceType == "auto"
		})).Return(uploadResult("avatar-abc"), nil)
		h.repo.On("Create", mock.MatchedBy(func(att *FileAttachment) bool {
			return att.OwnerType == "user" && att.OwnerID == 7 && att.Context == ContextProfilePicture &&
				att.PublicID == "avatar-abc" && att.Format == "png" && att.Bytes == 1024
		})).Return(nil)

		got, err := h.svc.UploadFile(context.Background(), in)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "avatar-abc", got.PublicID)
		assert.Contains(t, got.URL, "avatar-abc")
		h.uploader.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: upload error does not save attachment", func(t *testing.T) {
		h := newHarness()
		h.uploader.On("Upload", mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)

		got, err := h.svc.UploadFile(context.Background(), UploadFileParams{File: multipartFileHeader("test.png")})

		require.Error(t, err)
		assert.Nil(t, got)
		h.repo.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newHarness()
		h.uploader.On("Upload", mock.Anything, mock.Anything, mock.Anything).Return(uploadResult("p1"), nil)
		h.repo.On("Create", mock.Anything).Return(assert.AnError)

		got, err := h.svc.UploadFile(context.Background(), UploadFileParams{File: multipartFileHeader("test.png")})

		require.Error(t, err)
		assert.Nil(t, got)
		h.uploader.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})
}

func TestService_ReplaceFile(t *testing.T) {
	t.Run("success: overwrites the same public id and updates the row", func(t *testing.T) {
		h := newHarness()
		existing := &FileAttachment{PublicID: "users/7/avatar-abc", URL: "old"}
		existing.ID = 5
		h.repo.On("FindByID", uint(5)).Return(existing, nil)
		h.uploader.On("Upload", mock.Anything, mock.Anything, mock.MatchedBy(func(p uploader.UploadParams) bool {
			return p.PublicID == "users/7/avatar-abc" && p.Overwrite != nil && *p.Overwrite && p.Invalidate != nil && *p.Invalidate
		})).Return(uploadResult("users/7/avatar-abc"), nil)
		h.repo.On("Save", existing).Return(nil)

		got, err := h.svc.ReplaceFile(context.Background(), 5, multipartFileHeader("test.png"))

		require.NoError(t, err)
		assert.Same(t, existing, got)
		assert.Contains(t, got.URL, "avatar-abc")
		h.uploader.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: missing attachment returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindByID", uint(5)).Return(nil, gorm.ErrRecordNotFound)

		got, err := h.svc.ReplaceFile(context.Background(), 5, multipartFileHeader("test.png"))

		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, got)
		h.uploader.AssertNotCalled(t, "Upload", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: upload error does not save", func(t *testing.T) {
		h := newHarness()
		existing := &FileAttachment{PublicID: "p1"}
		existing.ID = 5
		h.repo.On("FindByID", uint(5)).Return(existing, nil)
		h.uploader.On("Upload", mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)

		got, err := h.svc.ReplaceFile(context.Background(), 5, multipartFileHeader("test.png"))

		require.Error(t, err)
		assert.Nil(t, got)
		h.repo.AssertNotCalled(t, "Save", mock.Anything)
	})
}

func TestService_DeleteFile(t *testing.T) {
	t.Run("success: destroys the asset and deletes the row", func(t *testing.T) {
		h := newHarness()
		existing := &FileAttachment{PublicID: "users/7/avatar-abc", ResourceType: "image"}
		existing.ID = 5
		h.repo.On("FindByID", uint(5)).Return(existing, nil)
		h.uploader.On("Destroy", mock.Anything, mock.MatchedBy(func(p uploader.DestroyParams) bool {
			return p.PublicID == "users/7/avatar-abc" && p.ResourceType == "image" && p.Invalidate != nil && *p.Invalidate
		})).Return(&uploader.DestroyResult{Result: "ok"}, nil)
		h.repo.On("Delete", existing).Return(nil)

		err := h.svc.DeleteFile(context.Background(), 5)

		require.NoError(t, err)
		h.uploader.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: missing attachment returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindByID", uint(5)).Return(nil, gorm.ErrRecordNotFound)

		err := h.svc.DeleteFile(context.Background(), 5)

		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		h.uploader.AssertNotCalled(t, "Destroy", mock.Anything, mock.Anything)
	})

	t.Run("failure: destroy error keeps the row", func(t *testing.T) {
		h := newHarness()
		existing := &FileAttachment{PublicID: "p1"}
		existing.ID = 5
		h.repo.On("FindByID", uint(5)).Return(existing, nil)
		h.uploader.On("Destroy", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		err := h.svc.DeleteFile(context.Background(), 5)

		require.Error(t, err)
		h.repo.AssertNotCalled(t, "Delete", mock.Anything)
	})
}
