package roomuserfavourite

import (
	"sound-stage-backend/internal/pkg/httpx"
)

type blockedChecker interface {
	IsBlocked(roomID, userID uint) (bool, error)
}

type Authz struct {
	checker blockedChecker
}

func NewAuthz(checker blockedChecker) *Authz {
	return &Authz{checker: checker}
}

func (a *Authz) CanAdd(userID, roomID uint) error {
	blocked, err := a.checker.IsBlocked(roomID, userID)
	if err != nil {
		return err
	}
	if blocked {
		return httpx.ErrUserBlocked
	}
	return nil
}
