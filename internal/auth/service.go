package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	apitoken "sound-stage-backend/internal/api_token"
	"sound-stage-backend/internal/infra/mailer"
	otprequest "sound-stage-backend/internal/otp_request"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/user"
)

type Service interface {
	RequestOTP(email string) (*otprequest.OTPRequest, error)
	VerifyOTP(params VerifyOTPParams) (*apitoken.TokenResult, error)
	SignUp(input *SignUpParams) (*apitoken.TokenResult, error)
	RefreshToken(refreshToken string) (*apitoken.TokenResult, error)
	Logout(userID uint) error
}

type service struct {
	userService       user.Service
	otpRequestService otprequest.Service
	apiTokenService   apitoken.Service
	mailer            mailer.Service
}

func NewService(userService user.Service, otpRequestService otprequest.Service, apiTokenService apitoken.Service, mailer mailer.Service) *service {
	return &service{userService: userService, otpRequestService: otpRequestService, apiTokenService: apiTokenService, mailer: mailer}
}

func (s *service) RequestOTP(email string) (*otprequest.OTPRequest, error) {
	otp, err := generateOTP(6)
	if err != nil {
		return nil, err
	}

	user, err := s.userService.FindByEmail(email)
	if err != nil {
		return nil, err
	}

	var inputEmail *string
	var userID *uint
	if user == nil {
		inputEmail = &email
	} else {
		userID = &user.ID
	}

	otpRequest, err := s.otpRequestService.Create(otprequest.CreateOTPRequestInput{
		Email:  inputEmail,
		UserID: userID,
		OTP:    otp,
	})
	if err != nil {
		return nil, err
	}

	s.mailer.SendOTPEmail(context.Background(), email, map[string]any{"otp": otp})

	return otpRequest, nil
}

func (s *service) VerifyOTP(params VerifyOTPParams) (*apitoken.TokenResult, error) {
	otpRequest, err := s.otpRequestService.FindByEmail(params.Email)
	if err != nil {
		return nil, err
	}

	if !otpRequest.VerifyOTP(params.OTP) {
		return nil, httpx.ErrInvalidOTP
	}

	err = s.otpRequestService.Deactivate(otpRequest.ID)
	if err != nil {
		return nil, err
	}

	if otpRequest.UserID == nil {
		return nil, nil
	}

	accessToken, err := s.apiTokenService.CreateToken(*otpRequest.UserID, apitoken.AccessToken)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.apiTokenService.CreateToken(*otpRequest.UserID, apitoken.RefreshToken)
	if err != nil {
		return nil, err
	}

	err = s.userService.UpdateLastLoginAt(*otpRequest.UserID)
	if err != nil {
		return nil, err
	}

	return &apitoken.TokenResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) SignUp(input *SignUpParams) (*apitoken.TokenResult, error) {
	user, err := s.userService.Create(&user.CreateUserParams{
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, err := s.apiTokenService.CreateToken(user.ID, apitoken.AccessToken)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.apiTokenService.CreateToken(user.ID, apitoken.RefreshToken)
	if err != nil {
		return nil, err
	}

	return &apitoken.TokenResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) RefreshToken(refreshToken string) (*apitoken.TokenResult, error) {
	userID, err := s.apiTokenService.ValidateToken(refreshToken, apitoken.RefreshToken)
	if err != nil {
		return nil, httpx.ErrInvalidRefreshToken
	}

	accessToken, err := s.apiTokenService.CreateToken(userID, apitoken.AccessToken)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.apiTokenService.CreateToken(userID, apitoken.RefreshToken)
	if err != nil {
		return nil, err
	}

	return &apitoken.TokenResult{AccessToken: accessToken, RefreshToken: newRefreshToken}, nil
}

func (s *service) Logout(userID uint) error {
	return s.apiTokenService.Deactivate(userID)
}

func generateOTP(length int) (string, error) {
	const digits = "0123456789"

	otp := make([]byte, length)
	for i := range length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[n.Int64()]
	}

	return string(otp), nil
}
