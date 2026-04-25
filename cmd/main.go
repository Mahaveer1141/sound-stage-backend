package main

import (
	"log/slog"
	"os"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using environment variables")
	}

	cfg := config.Load()

	level := slog.LevelInfo
	if cfg.Logger.Level == "debug" {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	s := server.NewServer(cfg, logger)
	if err := s.Run(); err != nil {
		logger.Error("Server failed to start", slog.String("error", err.Error()))
		panic(err)
	}
}
