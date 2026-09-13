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
	"sound-stage-backend/internal/category"
	chatmessage "sound-stage-backend/internal/chat_message"
	"sound-stage-backend/internal/config"
	fileattachment "sound-stage-backend/internal/file_attachment"
	"sound-stage-backend/internal/health"
	"sound-stage-backend/internal/infra/database"
	"sound-stage-backend/internal/infra/mailer"
	"sound-stage-backend/internal/infra/redis"
	"sound-stage-backend/internal/infra/worker"
	mediarouter "sound-stage-backend/internal/media_router"
	otprequest "sound-stage-backend/internal/otp_request"
	"sound-stage-backend/internal/role"
	"sound-stage-backend/internal/room"
	roomstate "sound-stage-backend/internal/room_state"
	roomuser "sound-stage-backend/internal/room_user"
	roomuserfavourite "sound-stage-backend/internal/room_user_favourite"
	"sound-stage-backend/internal/router"
	"sound-stage-backend/internal/tag"
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

	hub := ws.NewHub(s.logger)

	roomStateRepo := roomstate.NewRedisRepo(rdb)
	roomStatePublisher := roomstate.NewPublisher(rdb, roomstate.DefaultChannel, s.logger)
	roomStateService := roomstate.NewService(roomStateRepo, roomStatePublisher, s.logger)
	roomStateSubscriber := roomstate.NewSubscriber(rdb, roomstate.DefaultChannel, hub, s.logger)
	go roomStateSubscriber.Run(context.Background())

	mediaRouter := mediarouter.NewMediaRouter(hub, s.logger)

	mailService, err := mailer.NewService(s.cfg, s.logger, pool)
	if err != nil {
		return fmt.Errorf("Mailer Error: %w", err)
	}

	fileAttachmentRepo := fileattachment.NewRepo(db)
	fileAttachmentService := fileattachment.NewService(s.cfg.Cloudinary, fileAttachmentRepo)

	userRepo := user.NewRepo(db)
	otpRequestRepo := otprequest.NewRepo(db)
	apiTokenRepo := apitoken.NewRepo(db)
	roleRepo := role.NewRepo(db)
	roomRepo := room.NewRepo(db)
	roomUserRepo := roomuser.NewRepo(db)
	roomUserFavouriteRepo := roomuserfavourite.NewRepo(db)
	categoryRepo := category.NewRepo(db)
	tagRepo := tag.NewRepo(db)
	chatMessageRepo := chatmessage.NewRepo(db)

	apiTokenService := apitoken.NewService(s.cfg, apiTokenRepo)
	userService := user.NewService(userRepo, fileAttachmentService)
	otpRequestService := otprequest.NewService(otpRequestRepo)
	authService := auth.NewService(userService, otpRequestService, apiTokenService, mailService)
	roleService := role.NewService(roleRepo)
	roomUserAuthz := roomuser.NewAuthz(roomUserRepo)
	roomUserService := roomuser.NewService(roomUserRepo, roleService, mediaRouter, roomStateService, roomUserAuthz)
	roomUserFavouriteAuthz := roomuserfavourite.NewAuthz(roomUserService)
	roomUserFavouriteService := roomuserfavourite.NewService(roomUserFavouriteRepo, roomUserFavouriteAuthz)
	roomAuthz := room.NewAuthz(roomUserService)
	roomService := room.NewService(roomRepo, roomUserService, roomUserFavouriteService, db, roomAuthz, fileAttachmentService)
	categoryService := category.NewService(categoryRepo)
	tagService := tag.NewService(tagRepo)
	chatMessageAuthz := chatmessage.NewAuthz(roomUserService, roomRepo)
	chatMessageService := chatmessage.NewService(chatMessageRepo, chatMessageAuthz)

	wsHandler := ws.NewHandler(hub, s.cfg, roomUserAuthz.CanJoinRoom)
	roomWsHandler := roomuser.NewWSHandler(hub, roomUserService, roomUserAuthz, mediaRouter, s.cfg, s.logger)
	roomWsHandler.Register(wsHandler)

	registrar := worker.NewTaskRegistrar(pool, s.logger)
	registrar.RegisterAll(worker.TaskDeps{
		SendOTPEmail: mailService.SendOTP,
	})

	if err := pool.Start(); err != nil {
		return fmt.Errorf("Worker Error: %w", err)
	}

	go hub.Run()

	healthHandler := health.NewHandler(db)
	authHandler := auth.NewHandler(authService)
	userHandler := user.NewHandler(userService)
	roomHandler := room.NewHandler(roomService, hub)
	roomUserHandler := roomuser.NewHandler(roomUserService, roomService, hub)
	categoryHandler := category.NewHandler(categoryService)
	tagHandler := tag.NewHandler(tagService)
	roomUserFavouriteHandler := roomuserfavourite.NewHandler(roomUserFavouriteService)
	chatMessageHandler := chatmessage.NewHandler(chatMessageService, hub)

	handlers := &router.Handlers{
		Health:            healthHandler,
		Auth:              authHandler,
		Category:          categoryHandler,
		User:              userHandler,
		Room:              roomHandler,
		RoomUser:          roomUserHandler,
		Tag:               tagHandler,
		RoomUserFavourite: roomUserFavouriteHandler,
		ChatMessage:       chatMessageHandler,
		WS:                wsHandler,
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

	if err := roomStateSubscriber.Shutdown(ctx); err != nil {
		s.logger.Error("failed to close room state subscriber", slog.Any("error", err))
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
