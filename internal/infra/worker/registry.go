package worker

import (
	"log/slog"
)

type TaskDeps struct {
	SendOTPEmail SendEmailFunc
}

type TaskRegistrar struct {
	pool   *Pool
	logger *slog.Logger
}

func NewTaskRegistrar(pool *Pool, logger *slog.Logger) *TaskRegistrar {
	return &TaskRegistrar{pool: pool, logger: logger}
}

func (r *TaskRegistrar) RegisterAll(deps TaskDeps) {
	r.pool.RegisterHandler(TaskTypeSendOTPEmail, HandleSendOTPEmailTask(deps.SendOTPEmail))
}
