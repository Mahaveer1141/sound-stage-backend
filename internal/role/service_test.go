package role

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) FindByName(name RoleName) (*Role, error) {
	args := m.Called(name)
	return args.Get(0).(*Role), args.Error(1)
}

func TestService_FindByName(t *testing.T) {
	expectedRole := &Role{Name: "admin"}
	expectedErr := errors.New("role not found")

	tests := []struct {
		name      string
		roleName  RoleName
		mockSetup func(name RoleName) (*Role, error)
		want      *Role
		wantErr   bool
	}{
		{
			name:     "Success - Role Found",
			roleName: "Admin",
			mockSetup: func(name RoleName) (*Role, error) {
				return expectedRole, nil
			},
			want:    expectedRole,
			wantErr: false,
		},
		{
			name:     "Failure - Repository Error",
			roleName: "Unknown",
			mockSetup: func(name RoleName) (*Role, error) {
				return nil, expectedErr
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mockRepository)
			mockRepo.On("FindByName", tt.roleName).Return(tt.mockSetup(tt.roleName))

			svc := NewService(mockRepo)

			got, err := svc.FindByName(tt.roleName)

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.FindByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Service.FindByName() = %v, want %v", got, tt.want)
			}
		})
	}
}
