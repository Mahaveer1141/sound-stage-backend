package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/infra/worker"

	"github.com/mailgun/mailgun-go/v4"
)

type provider interface {
	sendEmail(ctx context.Context, email *Email) error
	sendEmailWithAttachments(ctx context.Context, email *EmailWithAttachments) error
	sendTemplateEmail(ctx context.Context, email *Email) error
}

type mailgunProvider struct {
	client *mailgun.MailgunImpl
	from   string
	config *config.Config
	logger *slog.Logger
}

type Service interface {
	SendOTPEmail(ctx context.Context, to string, vars map[string]any)

	SendOTP(ctx context.Context, to []string, vars map[string]any) error
}

type Email struct {
	To           []string
	Subject      string
	HTMLContent  string
	TextContent  string
	TemplateName string
	TemplateData map[string]any
}

type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

type EmailWithAttachments struct {
	Email
	Attachments []Attachment
}

type service struct {
	provider provider
	logger   *slog.Logger
	pool     *worker.Pool
}

func NewService(cfg *config.Config, logger *slog.Logger, pool *worker.Pool) (Service, error) {
	provider, err := newMailGunProvider(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &service{
		provider: provider,
		logger:   logger,
		pool:     pool,
	}, nil
}

func (s *service) SendOTPEmail(ctx context.Context, to string, vars map[string]any) {
	otp, ok := vars["otp"].(string)
	if !ok {
		s.logger.Error("OTP not found in vars")
		return
	}
	task, err := worker.NewSendOTPEmailTask(to, otp)
	if err != nil {
		s.logger.Error("Failed to create OTP email task", slog.String("error", err.Error()))
		return
	}

	if err := s.pool.Enqueue(task); err != nil {
		s.logger.Error("Failed to enqueue OTP email task", slog.String("error", err.Error()))
		return
	}
}

func (s *service) SendOTP(ctx context.Context, to []string, vars map[string]any) error {
	otp, ok := vars["otp"].(string)
	if !ok {
		return fmt.Errorf("OTP not found in vars")
	}

	email := &Email{
		To:          to,
		Subject:     "Your OTP for Sound Stage: " + otp,
		HTMLContent: renderOTPEmailHTML(otp),
	}
	return s.provider.sendEmail(ctx, email)
}
