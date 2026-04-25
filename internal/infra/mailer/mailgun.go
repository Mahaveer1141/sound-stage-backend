package mailer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"sound-stage-backend/internal/config"

	"github.com/mailgun/mailgun-go/v4"
)

func newMailGunProvider(cfg *config.Config, logger *slog.Logger) (provider, error) {
	if cfg.Mailer.MailgunAPIKey == "" {
		return nil, fmt.Errorf("Mailgun api key not configured")
	}

	mg := mailgun.NewMailgun(cfg.Mailer.MailgunDomain, cfg.Mailer.MailgunAPIKey)

	from := fmt.Sprintf("%s <%s>", cfg.Mailer.MailgunFromName, cfg.Mailer.MailgunFromEmail)

	logger.Info("Mailgun mailer initialized",
		slog.String("from_email", cfg.Mailer.MailgunFromEmail),
		slog.String("environment", cfg.Server.Environment),
	)

	return &mailgunProvider{
		client: mg,
		from:   from,
		config: cfg,
		logger: logger,
	}, nil
}

func (mg *mailgunProvider) sendEmail(ctx context.Context, email *Email) error {
	m := mg.client.NewMessage(
		mg.from,
		email.Subject,
		email.TextContent,
		email.To...)

	if email.HTMLContent != "" {
		m.SetHTML(email.HTMLContent)
	}

	_, id, err := mg.client.Send(ctx, m)

	if err != nil {
		return err
	}

	mg.logger.Info("Email sent successfully with ID",
		slog.String("id", id),
		slog.String("subject", email.Subject),
		slog.Any("to", email.To),
	)
	return nil
}

func (mg *mailgunProvider) sendEmailWithAttachments(ctx context.Context, email *EmailWithAttachments) error {
	m := mg.client.NewMessage(
		mg.from,
		email.Subject,
		email.TextContent,
		email.To...)

	if email.HTMLContent != "" {
		m.SetHtml(email.HTMLContent)
	}

	for _, attachment := range email.Attachments {
		m.AddReaderAttachment(attachment.Filename, io.NopCloser(bytes.NewReader(attachment.Content)))
	}

	_, id, err := mg.client.Send(ctx, m)

	if err != nil {
		return err
	}

	mg.logger.Info("Email with attachments sent successfully",
		slog.String("id", id),
		slog.String("subject", email.Subject),
		slog.Any("to", email.To),
		slog.Int("attachments", len(email.Attachments)),
	)

	return nil
}

func (mg *mailgunProvider) sendTemplateEmail(ctx context.Context, email *Email) error {
	m := mg.client.NewMessage(
		mg.from,
		email.Subject,
		"",
		email.To...)

	m.SetTemplate(email.TemplateName)

	for key, value := range email.TemplateData {
		m.AddTemplateVariable(key, value)
	}

	_, id, err := mg.client.Send(ctx, m)
	if err != nil {
		return err
	}

	mg.logger.Info("Email with attachments sent successfully",
		slog.String("id", id),
		slog.String("subject", email.Subject),
		slog.Any("to", email.To),
		slog.String("template_name", email.TemplateName),
	)

	return nil
}
