package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	apitoken "sound-stage-backend/internal/api_token"
	"sound-stage-backend/internal/auth"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/health"
	"sound-stage-backend/internal/infra/database"
	"sound-stage-backend/internal/infra/mailer"
	"sound-stage-backend/internal/infra/redis"
	"sound-stage-backend/internal/infra/worker"
	otprequest "sound-stage-backend/internal/otp_request"
	"sound-stage-backend/internal/room"
	roomuser "sound-stage-backend/internal/room_user"
	"sound-stage-backend/internal/router"
	"sound-stage-backend/internal/user"
	"sound-stage-backend/internal/ws"
	"syscall"
	"time"
)

type Server struct {
	cfg    *config.Config
	logger *slog.Logger
}

func NewServer(cfg *config.Config, logger *slog.Logger) *Server {
	return &Server{cfg: cfg, logger: logger}
}

func (s *Server) Run() error {
	db, err := database.Connect(s.cfg, s.logger)
	if err != nil {
		return fmt.Errorf("Database failure: %w", err)
	}

	rdb, err := redis.Connect(s.cfg, s.logger)
	if err != nil {
		return fmt.Errorf("Redis Error: %w", err)
	}

	pool := worker.NewPool(s.cfg, s.logger)

	hub := ws.NewHub()
	go hub.Run()

	mailService, err := mailer.NewService(s.cfg, s.logger, pool)
	if err != nil {
		return fmt.Errorf("Mailer Error: %w", err)
	}

	userRepo := user.NewRepo(db)
	otpRequestRepo := otprequest.NewRepo(db)
	apiTokenRepo := apitoken.NewRepo(db)
	roomRepo := room.NewRepo(db)
	roomUserRepo := roomuser.NewRepo(db)

	apiTokenService := apitoken.NewService(s.cfg, apiTokenRepo)
	userService := user.NewService(userRepo)
	otpRequestService := otprequest.NewService(otpRequestRepo)
	authService := auth.NewService(userService, otpRequestService, apiTokenService, mailService)
	roomService := room.NewService(roomRepo)
	roomUserService := roomuser.NewService(roomUserRepo)

	registrar := worker.NewTaskRegistrar(pool, s.logger)
	registrar.RegisterAll(worker.TaskDeps{
		SendOTPEmail: mailService.SendOTP,
	})

	if err := pool.Start(); err != nil {
		return fmt.Errorf("Worker Error: %w", err)
	}

	wsHandler := ws.NewHandler(hub, s.cfg)
	roomWsHandler := room.NewWSHandler(hub, roomUserService, s.cfg)

	roomWsHandler.Register(wsHandler)

	healthHandler := health.NewHandler(db)
	authHandler := auth.NewHandler(authService)
	userHandler := user.NewHandler(userService)
	roomHandler := room.NewHandler(roomService)

	handlers := &router.Handlers{
		Health: healthHandler,
		Auth:   authHandler,
		User:   userHandler,
		Room:   roomHandler,
		WS:     wsHandler,
	}

	r := router.Setup(s.cfg, handlers, apiTokenService, s.logger)
	srv := &http.Server{
		Addr:         ":" + s.cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
		IdleTimeout:  s.cfg.Server.IdleTimeout,
	}

	go func() {
		s.logger.Info("Starting server", slog.String("port", s.cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	<-quit

	s.logger.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	if err := database.Close(db); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	if err := redis.Close(rdb, s.logger); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	if err := pool.Close(); err != nil {
		return fmt.Errorf("failed to close worker pool: %w", err)
	}

	s.logger.Info("Server exited")

	return nil
}
