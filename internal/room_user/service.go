package roomuser

import (
	"context"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomstate "sound-stage-backend/internal/room_state"

	"gorm.io/gorm"
)

type roleFinder interface {
	FindByName(name role.RoleName) (*role.Role, error)
}

type publishRevoker interface {
	RevokePublishing(roomID uint, userID uint)
}

type roomStateService interface {
	GetParticipantStates(ctx context.Context, roomID uint, userIDs []uint) (map[uint]*roomstate.ParticipantState, error)
	GetRaisedHands(ctx context.Context, roomID uint, p listopts.Pagination) ([]uint, error)
	CountRaisedHands(ctx context.Context, roomID uint) (int64, error)
	SetMuted(ctx context.Context, roomID, userID uint, isMuted bool) error
	SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool) error
	Leave(ctx context.Context, roomID, userID uint) error
}

type repo interface {
	Create(tx *gorm.DB, userID uint, roomID uint, roleID uint) (*RoomUser, error)
	FindBy(userID uint, roomID uint) (*RoomUser, error)
	FindAnyBy(userID uint, roomID uint) (*RoomUser, error)
	UpdateActivity(ru *RoomUser, activity Activity) error
	HasRoles(userID uint, roomID uint, permissions []role.RoleName) (bool, error)
	ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, c listopts.Cursor) ([]RoomUser, bool, string, error)
	ListByUserIDs(roomID uint, userIDs []uint) ([]RoomUser, error)
	CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error)
	CountByRoomIDs(roomIDs []uint, filter RoomUserFilter) (map[uint]int64, error)
	UpdateRole(roomID uint, userID uint, roleID uint) error
	Delete(roomID uint, userID uint) error
	MapByUserAndRoomIDs(userID uint, roomIDs []uint) (map[uint]*RoomUser, error)
	IsBlocked(roomID, userID uint) (bool, error)
	Block(roomID, userID, blockedByID, listenerRoleID uint) error
	Unblock(roomID, userID uint) error
	ListBlockedByRoomID(roomID uint, filter RoomUserFilter, p listopts.Pagination) ([]RoomUser, error)
	CountBlockedByRoomID(roomID uint, filter RoomUserFilter) (int64, error)
}

type authorizer interface {
	CanBlock(roomID, actorID, targetID uint) (*RoomUser, error)
	CanUnblock(roomID, actorID, targetID uint) error
	CanListBlocked(roomID, actorID uint) error
	CanListUsers(roomID, userID uint) error
	CanListRaisedHands(roomID, userID uint) error
	CanUpdateRole(roomID, actorID uint, roleName role.RoleName) error
	CanDeleteUser(roomID, userID, actorID uint) (*RoomUser, error)
	CanSetMuted(roomID, actorID, targetID uint, isMuted bool) error
	CanSetHandRaised(roomID, userID uint) error
	CanModerate(roomID, actorID, targetID uint) (*RoomUser, *RoomUser, error)
}

type Service struct {
	repo        repo
	roleService roleFinder
	revoker     publishRevoker
	roomState   roomStateService
	authz       authorizer
}

func NewService(r repo, roleService roleFinder, revoker publishRevoker, roomState roomStateService, authz authorizer) *Service {
	return &Service{repo: r, roleService: roleService, revoker: revoker, roomState: roomState, authz: authz}
}

func (s *Service) Rejoin(ru *RoomUser) error {
	return s.repo.UpdateActivity(ru, ActivityJoin)
}

func (s *Service) Create(tx *gorm.DB, userID uint, roomID uint, roleName role.RoleName) (*RoomUser, error) {
	if roleName == "" {
		roleName = role.RoleListener
	}
	role, err := s.roleService.FindByName(roleName)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(tx, userID, roomID, role.ID)
}

func (s *Service) FindBy(userID uint, roomID uint) (*RoomUser, error) {
	return s.repo.FindBy(userID, roomID)
}

func (s *Service) FindAnyBy(userID uint, roomID uint) (*RoomUser, error) {
	return s.repo.FindAnyBy(userID, roomID)
}

func (s *Service) IsBlocked(roomID, userID uint) (bool, error) {
	return s.repo.IsBlocked(roomID, userID)
}

func (s *Service) Block(roomID, userID, actorID uint) (*RoomUser, error) {
	ru, err := s.authz.CanBlock(roomID, actorID, userID)
	if err != nil {
		return nil, err
	}

	listenerRole, err := s.roleService.FindByName(role.RoleListener)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Block(roomID, userID, actorID, listenerRole.ID); err != nil {
		return nil, err
	}

	s.revoker.RevokePublishing(roomID, userID)

	return ru, nil
}

func (s *Service) Unblock(roomID, userID, actorID uint) error {
	if err := s.authz.CanUnblock(roomID, actorID, userID); err != nil {
		return err
	}

	return s.repo.Unblock(roomID, userID)
}

func (s *Service) ListBlockedByRoomID(roomID, actorID uint, filter RoomUserFilter, p listopts.Pagination) ([]RoomUser, int64, error) {
	if err := s.authz.CanListBlocked(roomID, actorID); err != nil {
		return nil, 0, err
	}

	users, err := s.repo.ListBlockedByRoomID(roomID, filter, p)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.CountBlockedByRoomID(roomID, filter)
	if err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

func (s *Service) MapByUserAndRoomIDs(userID uint, roomIDs []uint) (map[uint]*RoomUser, error) {
	return s.repo.MapByUserAndRoomIDs(userID, roomIDs)
}

func (s *Service) CountByRoomIDs(roomIDs []uint, filter RoomUserFilter) (map[uint]int64, error) {
	return s.repo.CountByRoomIDs(roomIDs, filter)
}

func (s *Service) CountsByRoomID(roomID uint) (RoomUserCounts, error) {
	online := true
	total, err := s.repo.CountByRoomID(roomID, RoomUserFilter{})
	if err != nil {
		return RoomUserCounts{}, err
	}
	listenerCount, err := s.repo.CountByRoomID(roomID, RoomUserFilter{
		Roles:    []string{string(role.RoleListener)},
		IsOnline: &online,
	})
	if err != nil {
		return RoomUserCounts{}, err
	}
	speakerCount, err := s.repo.CountByRoomID(roomID, RoomUserFilter{
		Roles: []string{
			string(role.RoleOwner), string(role.RoleAdmin),
			string(role.RoleModerator), string(role.RoleSpeaker),
		},
		IsOnline: &online,
	})
	if err != nil {
		return RoomUserCounts{}, err
	}
	return RoomUserCounts{
		TotalUsersCount: total,
		Online:          OnlineCounts{ListenerCount: listenerCount, SpeakerCount: speakerCount},
	}, nil
}

func (s *Service) RemoveUser(ctx context.Context, userID uint, roomID uint) (*RoomUser, error) {
	ru, err := s.repo.FindBy(userID, roomID)
	if err != nil {
		return nil, err
	}
	if ru == nil {
		return nil, httpx.ErrRecordNotFound
	}
	if err := s.repo.UpdateActivity(ru, ActivityLeave); err != nil {
		return nil, err
	}
	return ru, s.roomState.Leave(ctx, roomID, userID)
}

func (s *Service) ListByRoomID(ctx context.Context, roomID, userID uint, filter RoomUserFilter, c listopts.Cursor) ([]RoomUser, int64, bool, string, error) {
	if err := s.authz.CanListUsers(roomID, userID); err != nil {
		return nil, 0, false, "", err
	}

	sort := listopts.Sort{Field: "last_joined_at", Order: "asc"}
	users, hasMore, nextCursor, err := s.repo.ListByRoomID(roomID, filter, sort, c)
	if err != nil {
		return nil, 0, false, "", err
	}
	count, err := s.repo.CountByRoomID(roomID, filter)
	if err != nil {
		return nil, 0, false, "", err
	}
	if err := s.attachStates(ctx, roomID, users); err != nil {
		return nil, 0, false, "", err
	}
	return users, count, hasMore, nextCursor, nil
}

func (s *Service) FindByWithState(ctx context.Context, userID, roomID uint) (*RoomUser, error) {
	ru, err := s.repo.FindBy(userID, roomID)
	if err != nil || ru == nil {
		return ru, err
	}
	states, err := s.roomState.GetParticipantStates(ctx, roomID, []uint{userID})
	if err != nil {
		return nil, err
	}
	if st := states[userID]; st != nil {
		ru.IsMuted = st.IsMuted
		ru.IsHandRaised = st.IsHandRaised
	}
	return ru, nil
}

func (s *Service) ListRaisedHands(ctx context.Context, roomID, userID uint, p listopts.Pagination) ([]RoomUser, int64, error) {
	if err := s.authz.CanListRaisedHands(roomID, userID); err != nil {
		return nil, 0, err
	}

	userIDs, err := s.roomState.GetRaisedHands(ctx, roomID, p)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.roomState.CountRaisedHands(ctx, roomID)
	if err != nil {
		return nil, 0, err
	}

	users, err := s.repo.ListByUserIDs(roomID, userIDs)
	if err != nil {
		return nil, 0, err
	}
	if err := s.attachStates(ctx, roomID, users); err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

func (s *Service) attachStates(ctx context.Context, roomID uint, users []RoomUser) error {
	if len(users) == 0 {
		return nil
	}
	userIDs := make([]uint, len(users))
	for i := range users {
		userIDs[i] = users[i].UserID
	}
	states, err := s.roomState.GetParticipantStates(ctx, roomID, userIDs)
	if err != nil {
		return err
	}
	for i := range users {
		if st := states[users[i].UserID]; st != nil {
			users[i].IsMuted = st.IsMuted
			users[i].IsHandRaised = st.IsHandRaised
		}
	}
	return nil
}

func (s *Service) HasRoles(userID uint, roomID uint, permissions []role.RoleName) (bool, error) {
	return s.repo.HasRoles(userID, roomID, permissions)
}

func (s *Service) UpdateRole(ctx context.Context, roomID uint, userID uint, roleName role.RoleName, actorID uint) (*RoomUser, error) {
	if err := s.authz.CanUpdateRole(roomID, actorID, roleName); err != nil {
		return nil, err
	}
	r, err := s.roleService.FindByName(roleName)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateRole(roomID, userID, r.ID); err != nil {
		return nil, err
	}

	if roleName == role.RoleListener {
		s.revoker.RevokePublishing(roomID, userID)
	}

	ru, err := s.FindByWithState(ctx, userID, roomID)
	if err != nil {
		return nil, err
	}
	if ru == nil {
		return nil, httpx.ErrRecordNotFound
	}
	return ru, nil
}

func (s *Service) DeleteUser(ctx context.Context, roomID, userID, actorID uint) (*RoomUser, error) {
	ru, err := s.authz.CanDeleteUser(roomID, userID, actorID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Delete(roomID, userID); err != nil {
		return nil, err
	}
	return ru, s.roomState.Leave(ctx, roomID, userID)
}

func (s *Service) SetMuted(ctx context.Context, roomID, userID, actorID uint, isMuted bool) error {
	if err := s.authz.CanSetMuted(roomID, actorID, userID, isMuted); err != nil {
		return err
	}

	return s.roomState.SetMuted(ctx, roomID, userID, isMuted)
}

func (s *Service) SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool) error {
	if err := s.authz.CanSetHandRaised(roomID, userID); err != nil {
		return err
	}
	return s.roomState.SetHandRaised(ctx, roomID, userID, isHandRaised)
}
