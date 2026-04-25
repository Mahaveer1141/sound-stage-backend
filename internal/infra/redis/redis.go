package redis

import (
	"context"
	"fmt"
	"log/slog"
	"sound-stage-backend/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

func Connect(cfg *config.Config, logger *slog.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Info("Connected to the redis successfully")

	return rdb, nil
}

func Close(rdb *redis.Client, logger *slog.Logger) error {
	err := rdb.Close()
	if err != nil {
		return err
	}

	logger.Info("Closed the redis connection successfully")

	return nil
}
