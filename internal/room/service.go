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
	FindBy(userID uint, roomID uint) (*roomuser.RoomUser, error)
	MapByUserAndRoomIDs(userID uint, roomIDs []uint) (map[uint]*roomuser.RoomUser, error)
}

type roomUserFavouriteService interface {
	FavouritedRoomIDs(userID uint, roomIDs []uint) (map[uint]bool, error)
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

type authorizer interface {
	CanView(roomID, userID uint) error
	CanUpdate(roomID, userID uint) error
	CanAddRoomUser(roomID, userID uint) error
}

type Service struct {
	repo             repository
	roomUserService  roomUserService
	favouriteService roomUserFavouriteService
	authz            authorizer
	db               *gorm.DB
}

func NewService(r repository, roomUserSvc roomUserService,
	favouriteSvc roomUserFavouriteService, db *gorm.DB, authz authorizer) *Service {
	return &Service{
		repo:             r,
		roomUserService:  roomUserSvc,
		favouriteService: favouriteSvc,
		authz:            authz,
		db:               db,
	}
}

func (s *Service) FindByID(id, userID uint) (*Room, error) {
	room, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.authz.CanView(id, userID); err != nil {
		return nil, err
	}

	tagToRooms, err := s.repo.LoadTagsForRooms([]uint{id})
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
	if err := s.authz.CanUpdate(id, userID); err != nil {
		return nil, err
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

func (s *Service) ViewerContext(roomID, userID uint) (*RoomViewer, error) {
	viewers, err := s.ViewerContexts([]uint{roomID}, userID)
	if err != nil {
		return nil, err
	}
	return viewers[roomID], nil
}

func (s *Service) ViewerContexts(roomIDs []uint, userID uint) (map[uint]*RoomViewer, error) {
	memberships, err := s.roomUserService.MapByUserAndRoomIDs(userID, roomIDs)
	if err != nil {
		return nil, err
	}
	favourited, err := s.favouriteService.FavouritedRoomIDs(userID, roomIDs)
	if err != nil {
		return nil, err
	}
	viewers := make(map[uint]*RoomViewer, len(roomIDs))
	for _, id := range roomIDs {
		viewers[id] = &RoomViewer{RoomUser: memberships[id], IsFavourited: favourited[id]}
	}
	return viewers, nil
}

func (s *Service) UpdatePrivateCode(roomID, userID uint) error {
	if err := s.authz.CanUpdate(roomID, userID); err != nil {
		return err
	}
	code, err := generatePrivateCode(8)
	if err != nil {
		return err
	}
	return s.repo.UpdatePrivateCode(roomID, code)
}

func (s *Service) AddRoomUser(roomID, userID uint, privateCode string) (*roomuser.RoomUser, error) {
	if err := s.authz.CanAddRoomUser(roomID, userID); err != nil {
		return nil, err
	}

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
