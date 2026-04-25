package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

type SendEmailFunc func(ctx context.Context, to []string, vars map[string]any) error

func HandleSendOTPEmailTask(sendEmail SendEmailFunc) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p SendOTPEmailPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
		}

		return sendEmail(ctx, []string{p.To}, map[string]any{"otp": p.OTP})
	}
}
