package room

import (
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"

	"gorm.io/gorm"
)

type Service interface {
	FindByID(id uint) (*Room, error)
	Create(input *CreateRoomParams) (*Room, error)
	Update(id uint, input *UpdateRoomParams) (*Room, error)
	List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, int64, error)
	ListUsers(roomID uint, filter roomuser.RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]roomuser.RoomUser, int64, error)
}

type service struct {
	repo            Repo
	roomUserService roomuser.Service
	db              *gorm.DB
}

func NewService(repo Repo, roomUserService roomuser.Service, db *gorm.DB) Service {
	return &service{repo: repo, roomUserService: roomUserService, db: db}
}

func (s *service) FindByID(id uint) (*Room, error) {
	return s.repo.FindByID(id)
}

func (s *service) Create(input *CreateRoomParams) (*Room, error) {
	var room *Room
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		room, err = s.repo.Create(tx, input)
		if err != nil {
			return err
		}
		_, err = s.roomUserService.AddUserWithTx(tx, input.CreatorID, room.ID, role.RoleAdmin)
		return err
	})
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (s *service) Update(id uint, input *UpdateRoomParams) (*Room, error) {
	return s.repo.Update(id, input)
}

func (s *service) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, int64, error) {
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

func (s *service) ListUsers(roomID uint, filter roomuser.RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]roomuser.RoomUser, int64, error) {
	return s.roomUserService.ListByRoomID(roomID, filter, sort, p)
}
