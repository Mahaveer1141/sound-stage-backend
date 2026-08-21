package apitoken

import (
	"fmt"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/pkg/httpx"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type repository interface {
	FindByToken(token string) (*APIToken, error)
	CreateToken(inputs CreateAPITokenInput) (*APIToken, error)
	Deactivate(userID uint) error
}

type Service struct {
	cfg  *config.Config
	repo repository
}

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

func NewService(cfg *config.Config, r repository) *Service {
	return &Service{cfg: cfg, repo: r}
}

func (s *Service) CreateToken(userID uint, tokenType TokenType) (*APIToken, error) {
	if tokenType == "" {
		tokenType = "access"
	}

	token, err := s.generateJWT(userID, string(tokenType))
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(s.cfg.JWT.AccessTokenExpiry)
	if tokenType == RefreshToken {
		expiresAt = time.Now().Add(s.cfg.JWT.RefreshTokenExpiry)
	}

	inputs := CreateAPITokenInput{
		UserID:    userID,
		Token:     token,
		Type:      tokenType,
		ExpiresAt: expiresAt,
	}
	return s.repo.CreateToken(inputs)
}

func (s *Service) Deactivate(userID uint) error {
	return s.repo.Deactivate(userID)
}

func (s *Service) ValidateToken(token string, tokenType TokenType) (uint, error) {
	if tokenType == "" {
		tokenType = AccessToken
	}

	key := s.cfg.JWT.AccessTokenSecret
	if tokenType == RefreshToken {
		key = s.cfg.JWT.RefreshTokenSecret
	}

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return []byte(key), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil || !parsed.Valid {
		return 0, httpx.ErrInvalidOrExpiredToken
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return 0, httpx.ErrInvalidTokenClaims
	}

	claimUserID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, httpx.ErrInvalidTokenClaims
	}

	at, err := s.repo.FindByToken(token)
	if err != nil {
		return 0, httpx.ErrTokenNotFoundOrRevoked
	}

	if at.Type != string(tokenType) || at.UserID != uint(claimUserID) {
		return 0, httpx.ErrTokenDataMismatch
	}

	return at.UserID, nil
}

func (s *Service) generateJWT(userID uint, typ string) (string, error) {
	if typ != "access" && typ != "refresh" {
		return "", fmt.Errorf("invalid token type: %s", typ)
	}

	exp := s.cfg.JWT.AccessTokenExpiry
	if typ == "refresh" {
		exp = s.cfg.JWT.RefreshTokenExpiry
	}
	key := s.cfg.JWT.AccessTokenSecret
	if typ == "refresh" {
		key = s.cfg.JWT.RefreshTokenSecret
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    typ,
		"exp":     time.Now().Add(exp).Unix(),
		"iat":     time.Now().Unix(),
		"iss":     s.cfg.JWT.Issuer,
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(key))
}
