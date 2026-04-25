package worker

import (
	"context"
	"log/slog"
	"sound-stage-backend/internal/config"

	"github.com/hibiken/asynq"
)

type Pool struct {
	client *asynq.Client
	server *asynq.Server
	logger *slog.Logger
	mux    *asynq.ServeMux
}

func NewPool(cfg *config.Config, logger *slog.Logger) *Pool {
	redisOpts := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	client := asynq.NewClient(redisOpts)

	server := asynq.NewServer(redisOpts, asynq.Config{
		Concurrency: cfg.Worker.Concurrency,
		Queues: map[string]int{
			"default": cfg.Worker.DefaultQueue,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			logger.Error("Task processing failed",
				slog.String("task_type", task.Type()),
				slog.String("error", err.Error()),
			)
		}),
	})

	mux := asynq.NewServeMux()

	logger.Info("Worker pool created",
		slog.Int("concurrency", cfg.Worker.Concurrency),
		slog.Int("default_queue", cfg.Worker.DefaultQueue),
	)

	return &Pool{
		client: client,
		server: server,
		logger: logger,
		mux:    mux,
	}
}

func (p *Pool) Start() error {
	err := p.server.Start(p.mux)
	if err != nil {
		return err
	}
	p.logger.Info("Worker pool started")
	return nil
}

func (p *Pool) Enqueue(task *asynq.Task, opts ...asynq.Option) error {
	info, err := p.client.Enqueue(task, opts...)
	if err != nil {
		return err
	}
	p.logger.Debug("Task enqueued",
		slog.String("queue", info.Queue),
		slog.String("Task ID", info.ID),
		slog.String("Task Type", task.Type()),
	)
	return nil
}

func (p *Pool) RegisterHandler(pattern string, handler asynq.HandlerFunc) {
	p.mux.Handle(pattern, handler)
	p.logger.Debug("Handler registered", slog.String("task_type", pattern))
}

func (p *Pool) Close() error {
	p.server.Shutdown()
	if err := p.client.Close(); err != nil {
		return err
	}

	p.logger.Info("Worker pool shut down")

	return nil
}
