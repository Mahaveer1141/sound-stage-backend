package router

import (
	"log/slog"
	apitoken "sound-stage-backend/internal/api_token"
	"sound-stage-backend/internal/auth"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/health"
	"sound-stage-backend/internal/middleware"
	"sound-stage-backend/internal/room"
	"sound-stage-backend/internal/user"
	"sound-stage-backend/internal/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	WS     ws.Handler
	Health health.Handler
	Auth   auth.Handler
	User   user.Handler
	Room   room.Handler
}

func Setup(cfg *config.Config, handlers *Handlers, apiTokenService apitoken.Service, logger *slog.Logger) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)

	router := gin.New()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))

	router.GET("/health", handlers.Health.Health)

	auth := router.Group("/auth")
	{
		auth.POST("/request_otp", handlers.Auth.RequestOTP)
		auth.POST("/verify_otp", handlers.Auth.VerifyOTP)
		auth.POST("/sign_up", handlers.Auth.SignUp)
		auth.POST("/refresh", handlers.Auth.RefreshToken)
		auth.POST("/logout", middleware.AuthMiddleware(apiTokenService), handlers.Auth.Logout)
	}

	users := router.Group("/users", middleware.AuthMiddleware(apiTokenService))
	{
		users.GET("/current", handlers.User.CurrentUser)
		users.PUT("/profile", handlers.User.UpdateProfile)
	}

	rooms := router.Group("/rooms", middleware.AuthMiddleware(apiTokenService))
	{
		rooms.GET("", handlers.Room.List)
		rooms.GET("/:id", handlers.Room.FindByID)
		rooms.POST("", handlers.Room.Create)
		rooms.PUT("/:id", handlers.Room.Update)
	}

	router.GET("/ws/rooms/:roomId", middleware.AuthMiddleware(apiTokenService), handlers.WS.ServeWS)

	return router
}
