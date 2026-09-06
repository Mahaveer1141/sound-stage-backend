package user

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"testing"

	fileattachment "sound-stage-backend/internal/file_attachment"
	"sound-stage-backend/internal/pkg/httpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) Create(input *CreateUserParams) (*User, error) {
	args := m.Called(input)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}
func (m *mockRepository) FindByID(id uint) (*User, error) {
	args := m.Called(id)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}
func (m *mockRepository) FindByEmail(email string) (*User, error) {
	args := m.Called(email)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}
func (m *mockRepository) UpdateLastLoginAt(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *mockRepository) Save(user *User) error {
	args := m.Called(user)
	return args.Error(0)
}

type mockFileAttachmentService struct{ mock.Mock }

func (m *mockFileAttachmentService) UploadOrReplaceFile(ctx context.Context, existing *fileattachment.FileAttachment, in fileattachment.UploadFileParams) (*fileattachment.FileAttachment, error) {
	args := m.Called(ctx, existing, in)
	att, _ := args.Get(0).(*fileattachment.FileAttachment)
	return att, args.Error(1)
}

func TestService_FindByID(t *testing.T) {
	t.Run("success: returns the exact record from repo", func(t *testing.T) {
		repo := new(mockRepository)
		want := &User{Email: "a@example.com"}
		repo.On("FindByID", uint(1)).Return(want, nil)

		svc := NewService(repo, nil)
		got, err := svc.FindByID(1)

		require.NoError(t, err)
		assert.Same(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged", func(t *testing.T) {
		repo := new(mockRepository)
		repoErr := errors.New("not found")
		repo.On("FindByID", uint(99)).Return(nil, repoErr)

		svc := NewService(repo, nil)
		got, err := svc.FindByID(99)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func TestService_FindByEmail(t *testing.T) {
	t.Run("success: returns the exact record from repo", func(t *testing.T) {
		repo := new(mockRepository)
		want := &User{Email: "a@example.com"}
		repo.On("FindByEmail", "a@example.com").Return(want, nil)

		svc := NewService(repo, nil)
		got, err := svc.FindByEmail("a@example.com")

		require.NoError(t, err)
		assert.Same(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged", func(t *testing.T) {
		repo := new(mockRepository)
		repoErr := errors.New("not found")
		repo.On("FindByEmail", "missing@example.com").Return(nil, repoErr)

		svc := NewService(repo, nil)
		got, err := svc.FindByEmail("missing@example.com")

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	t.Run("success: returns the created record from repo", func(t *testing.T) {
		repo := new(mockRepository)
		input := &CreateUserParams{Email: "new@example.com", FirstName: "Ada"}
		want := &User{Email: "new@example.com", FirstName: "Ada"}
		repo.On("Create", input).Return(want, nil)

		svc := NewService(repo, nil)
		got, err := svc.Create(input)

		require.NoError(t, err)
		assert.Same(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged (e.g. duplicate email)", func(t *testing.T) {
		repo := new(mockRepository)
		input := &CreateUserParams{Email: "dup@example.com", FirstName: "Ada"}
		repoErr := errors.New("duplicate email")
		repo.On("Create", input).Return(nil, repoErr)

		svc := NewService(repo, nil)
		got, err := svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func TestService_UpdateLastLoginAt(t *testing.T) {
	t.Run("success: delegates to repo", func(t *testing.T) {
		repo := new(mockRepository)
		repo.On("UpdateLastLoginAt", uint(1)).Return(nil)

		svc := NewService(repo, nil)
		err := svc.UpdateLastLoginAt(1)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged", func(t *testing.T) {
		repo := new(mockRepository)
		repoErr := errors.New("update failed")
		repo.On("UpdateLastLoginAt", uint(1)).Return(repoErr)

		svc := NewService(repo, nil)
		err := svc.UpdateLastLoginAt(1)

		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func multipartFileHeader(filename string) (*multipart.FileHeader, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("profile_picture", filename)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write([]byte("fake-image")); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	r := multipart.NewReader(&b, w.Boundary())
	form, err := r.ReadForm(1024)
	if err != nil {
		return nil, err
	}
	files := form.File["profile_picture"]
	if len(files) == 0 {
		return nil, errors.New("no file")
	}
	return files[0], nil
}

func TestService_UpdateProfile(t *testing.T) {
	t.Run("success: updates names when no profile picture", func(t *testing.T) {
		repo := new(mockRepository)
		input := &UpdateUserParams{FirstName: "Grace"}
		existing := &User{Email: "old@example.com", FirstName: "Old"}
		existing.ID = 1
		repo.On("FindByID", uint(1)).Return(existing, nil)
		repo.On("Save", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			saved := args.Get(0).(*User)
			assert.Equal(t, "Grace", saved.FirstName)
		})

		svc := NewService(repo, nil)
		got, err := svc.UpdateProfile(1, input)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "Grace", got.FirstName)
		repo.AssertExpectations(t)
	})

	t.Run("success: uploads new profile picture", func(t *testing.T) {
		repo := new(mockRepository)
		fileSvc := new(mockFileAttachmentService)
		existing := &User{Email: "old@example.com", FirstName: "Old"}
		existing.ID = 1
		repo.On("FindByID", uint(1)).Return(existing, nil)
		repo.On("Save", mock.Anything).Return(nil)
		fileSvc.On("UploadOrReplaceFile", mock.Anything, mock.Anything, mock.MatchedBy(func(in fileattachment.UploadFileParams) bool {
			return in.OwnerType == "users" && in.OwnerID == 1 &&
				in.Context == fileattachment.ContextProfilePicture && in.File != nil
		})).Return(&fileattachment.FileAttachment{URL: "https://example.com/pic.png"}, nil)

		fh, err := multipartFileHeader("test.png")
		require.NoError(t, err)
		input := &UpdateUserParams{FirstName: "Grace", ProfilePicture: fh}

		svc := NewService(repo, fileSvc)
		got, err := svc.UpdateProfile(1, input)

		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotNil(t, got.ProfilePicture)
		assert.Equal(t, "https://example.com/pic.png", got.ProfilePicture.URL)
		repo.AssertExpectations(t)
		fileSvc.AssertExpectations(t)
	})

	t.Run("success: replaces existing profile picture", func(t *testing.T) {
		repo := new(mockRepository)
		fileSvc := new(mockFileAttachmentService)
		existingPic := &fileattachment.FileAttachment{}
		existingPic.ID = 5
		existing := &User{Email: "old@example.com", FirstName: "Old", ProfilePicture: existingPic}
		existing.ID = 1
		repo.On("FindByID", uint(1)).Return(existing, nil)
		repo.On("Save", mock.Anything).Return(nil)
		fileSvc.On("UploadOrReplaceFile", mock.Anything, existingPic, mock.MatchedBy(func(in fileattachment.UploadFileParams) bool {
			return in.OwnerType == "users" && in.OwnerID == 1 &&
				in.Context == fileattachment.ContextProfilePicture && in.File != nil
		})).Return(&fileattachment.FileAttachment{URL: "https://example.com/new.png"}, nil)

		fh, err := multipartFileHeader("test.png")
		require.NoError(t, err)
		input := &UpdateUserParams{FirstName: "Grace", ProfilePicture: fh}

		svc := NewService(repo, fileSvc)
		got, err := svc.UpdateProfile(1, input)

		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotNil(t, got.ProfilePicture)
		assert.Equal(t, "https://example.com/new.png", got.ProfilePicture.URL)
		repo.AssertExpectations(t)
		fileSvc.AssertExpectations(t)
	})

	t.Run("failure: user not found", func(t *testing.T) {
		repo := new(mockRepository)
		input := &UpdateUserParams{FirstName: "Grace"}
		repo.On("FindByID", uint(99)).Return(nil, nil)

		svc := NewService(repo, nil)
		got, err := svc.UpdateProfile(99, input)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrRecordNotFound)
		repo.AssertExpectations(t)
	})

	t.Run("failure: file upload error is propagated", func(t *testing.T) {
		repo := new(mockRepository)
		fileSvc := new(mockFileAttachmentService)
		existing := &User{Email: "old@example.com", FirstName: "Old"}
		existing.ID = 1
		repo.On("FindByID", uint(1)).Return(existing, nil)
		fileSvc.On("UploadOrReplaceFile", mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)

		fh, err := multipartFileHeader("test.png")
		require.NoError(t, err)
		input := &UpdateUserParams{FirstName: "Grace", ProfilePicture: fh}

		svc := NewService(repo, fileSvc)
		got, err := svc.UpdateProfile(1, input)

		require.Nil(t, got)
		require.Error(t, err)
		repo.AssertExpectations(t)
		fileSvc.AssertExpectations(t)
	})
}
