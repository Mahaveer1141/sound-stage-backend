package room

import (
	"crypto/rand"
	"math/big"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
	"sound-stage-backend/internal/tag"

	"gorm.io/gorm"
)

type roomUserService interface {
	Create(tx *gorm.DB, userID uint, roomID uint, roleName role.RoleName) (*roomuser.RoomUser, error)
	Rejoin(ru *roomuser.RoomUser) error
	RemoveUser(userID uint, roomID uint) error
	ListByRoomID(roomID uint, filter roomuser.RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]roomuser.RoomUser, int64, error)
	FindBy(userID uint, roomID uint) (*roomuser.RoomUser, error)
	UpdateRole(roomID uint, userID uint, role role.RoleName, actorID uint) error
	HasRoles(userID uint, roomID uint, permissions []role.RoleName) (bool, error)
}

type repository interface {
	Create(tx *gorm.DB, input *CreateRoomParams) (*Room, error)
	Update(id uint, input *UpdateRoomParams) (*Room, error)
	UpdatePrivateCode(id uint, code string) error
	FindByID(id uint) (*Room, error)
	List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, error)
	Count(filter RoomFilter) (int64, error)
	LoadTagsForRooms(roomIds []uint) (map[uint][]tag.Tag, error)
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
	tagToRooms, err := s.repo.LoadTagsForRooms([]uint{id})
	if err != nil {
		return nil, err
	}
	room, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	room.Tags = tagToRooms[id]
	return room, nil
}

func (s *Service) Create(input *CreateRoomParams) (*Room, error) {
	var room *Room
	if input.Type == RoomTypePrivate {
		code, err := generatePrivateCode(8)
		if err != nil {
			return nil, err
		}
		input.privateCode = &code
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		room, err = s.repo.Create(tx, input)
		if err != nil {
			return err
		}
		_, err = s.roomUserService.Create(tx, input.CreatorID, room.ID, role.RoleOwner)
		return err
	})
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (s *Service) Update(id, userID uint, input *UpdateRoomParams) (*Room, error) {
	ok, err := s.roomUserService.HasRoles(userID, id, []role.RoleName{role.RoleOwner, role.RoleAdmin})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.ErrForbidden
	}

	room, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if input.Type == RoomTypePublic {
		input.privateCode = nil
	} else {
		var code string
		if room.PrivateCode != nil {
			code = *room.PrivateCode
		} else {
			code, err = generatePrivateCode(8)
			if err != nil {
				return nil, err
			}
		}
		input.privateCode = &code
	}
	return s.repo.Update(id, input)
}

func (s *Service) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, int64, error) {
	rooms, err := s.repo.List(filter, sort, p)
	if err != nil {
		return nil, 0, err
	}

	roomIds := make([]uint, len(rooms))
	for i, room := range rooms {
		roomIds[i] = room.ID
	}
	tagToRooms, err := s.repo.LoadTagsForRooms(roomIds)
	if err != nil {
		return nil, 0, err
	}
	for i, room := range rooms {
		rooms[i].Tags = tagToRooms[room.ID]
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

func (s *Service) UpdatePrivateCode(roomID uint) error {
	code, err := generatePrivateCode(8)
	if err != nil {
		return err
	}
	return s.repo.UpdatePrivateCode(roomID, code)
}

func (s *Service) AddRoomUser(roomID, userID uint, privateCode string) (*roomuser.RoomUser, error) {
	ru, err := s.roomUserService.FindBy(userID, roomID)
	if err != nil {
		return nil, err
	}
	if ru != nil {
		return ru, s.roomUserService.Rejoin(ru)
	}

	room, err := s.repo.FindByID(roomID)
	if err != nil {
		return nil, err
	}
	if room.Type == RoomTypePrivate && privateCode != *room.PrivateCode {
		return nil, httpx.ErrForbidden
	}

	return s.roomUserService.Create(nil, userID, roomID, role.RoleListener)
}

func generatePrivateCode(length int) (string, error) {
	const digits = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	code := make([]byte, length)

	for i := range length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[n.Int64()]
	}
	return string(code), nil
}
