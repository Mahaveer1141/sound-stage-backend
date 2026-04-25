package worker

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const (
	TaskTypeSendOTPEmail = "email:send_otp"
)

type SendOTPEmailPayload struct {
	To  string `json:"to"`
	OTP string `json:"otp"`
}

func NewSendOTPEmailTask(to string, otp string) (*asynq.Task, error) {
	payload, err := json.Marshal(SendOTPEmailPayload{To: to, OTP: otp})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTypeSendOTPEmail, payload), nil
}
