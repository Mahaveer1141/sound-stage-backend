package apitoken

import (
	"errors"
	"fmt"
	"sound-stage-backend/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service interface {
	CreateToken(userID uint, tokenType TokenType) (*APIToken, error)
	Deactivate(userID uint) error
	ValidateToken(token string, tokenType TokenType) (uint, error)
}

type service struct {
	cfg  *config.Config
	repo Repo
}

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

func NewService(cfg *config.Config, repo Repo) *service {
	return &service{cfg: cfg, repo: repo}
}

func (s *service) CreateToken(userID uint, tokenType TokenType) (*APIToken, error) {
	if tokenType == "" {
		tokenType = "access"
	}

	token, err := s.generateJWT(userID, string(tokenType))
	if err != nil {
		return nil, err
	}
	inputs := CreateAPITokenInput{
		UserID: userID,
		Token:  token,
		Type:   tokenType,
	}
	return s.repo.CreateToken(inputs)
}

func (s *service) Deactivate(userID uint) error {
	return s.repo.Deactivate(userID)
}

func (s *service) ValidateToken(token string, tokenType TokenType) (uint, error) {
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
		return 0, errors.New("invalid or expired token")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid token claims")
	}

	claimUserID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid token claims")
	}

	at, err := s.repo.FindByToken(token)
	if err != nil {
		return 0, errors.New("token not found or revoked")
	}

	if at.Type != string(tokenType) || at.UserID != uint(claimUserID) {
		return 0, errors.New("token data mismatch")
	}

	return at.UserID, nil
}

func (s *service) generateJWT(userID uint, typ string) (string, error) {
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
