package room

import (
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"

	"gorm.io/gorm"
)

type roomUserService interface {
	AddUser(userID uint, roomID uint, roleName role.RoleName) (*roomuser.RoomUser, error)
	AddUserWithTx(tx *gorm.DB, userID uint, roomID uint, roleName role.RoleName) (*roomuser.RoomUser, error)
	RemoveUser(userID uint, roomID uint) error
	ListByRoomID(roomID uint, filter roomuser.RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]roomuser.RoomUser, int64, error)
	FindBy(userID uint, roomID uint) (*roomuser.RoomUser, error)
	UpdateRole(roomID uint, userID uint, role role.RoleName, actorID uint) error
}

type repository interface {
	Create(tx *gorm.DB, input *CreateRoomParams) (*Room, error)
	Update(id uint, input *UpdateRoomParams) (*Room, error)
	FindByID(id uint) (*Room, error)
	List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, error)
	Count(filter RoomFilter) (int64, error)
}

type Service struct {
	repo            repository
	roomUserService roomUserService
	db              *gorm.DB
}

func NewService(r repository, roomUserSvc roomUserService, db *gorm.DB) *Service {
	return &Service{repo: r, roomUserService: roomUserSvc, db: db}
}

func (s *Service) FindByID(id uint) (*Room, error) {
	return s.repo.FindByID(id)
}

func (s *Service) Create(input *CreateRoomParams) (*Room, error) {
	var room *Room
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		room, err = s.repo.Create(tx, input)
		if err != nil {
			return err
		}
		_, err = s.roomUserService.AddUserWithTx(tx, input.CreatorID, room.ID, role.RoleOwner)
		return err
	})
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (s *Service) Update(id uint, input *UpdateRoomParams) (*Room, error) {
	return s.repo.Update(id, input)
}

func (s *Service) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, int64, error) {
	rooms, err := s.repo.List(filter, sort, p)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.Count(filter)
	if err != nil {
		return nil, 0, err
	}
	return rooms, count, nil
}

func (s *Service) ListUsers(roomID uint, filter roomuser.RoomUserFilter,
	sort listopts.Sort, p listopts.Pagination) ([]roomuser.RoomUser, int64, error) {
	return s.roomUserService.ListByRoomID(roomID, filter, sort, p)
}

func (s *Service) UpdateUserRole(roomID uint, userID uint, newRole role.RoleName, actorID uint) error {
	return s.roomUserService.UpdateRole(roomID, userID, newRole, actorID)
}

func (s *Service) CurrentRoomUser(roomID, userID uint) (*roomuser.RoomUser, error) {
	return s.roomUserService.FindBy(userID, roomID)
}
