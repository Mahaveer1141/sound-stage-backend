package router

import (
	"log/slog"
	"sound-stage-backend/internal/auth"
	"sound-stage-backend/internal/category"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/health"
	"sound-stage-backend/internal/middleware"
	"sound-stage-backend/internal/room"
	roomuser "sound-stage-backend/internal/room_user"
	roomuserblock "sound-stage-backend/internal/room_user_block"
	roomuserfavourite "sound-stage-backend/internal/room_user_favourite"
	"sound-stage-backend/internal/tag"
	"sound-stage-backend/internal/user"
	"sound-stage-backend/internal/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	WS                ws.Handler
	Health            *health.Handler
	Auth              *auth.Handler
	Category          *category.Handler
	User              *user.Handler
	Room              *room.Handler
	RoomUser          *roomuser.Handler
	Tag               *tag.Handler
	RoomUserFavourite *roomuserfavourite.Handler
	RoomUserBlock     *roomuserblock.Handler
}

func Setup(cfg *config.Config, handlers *Handlers, tokenValidator middleware.TokenValidator, logger *slog.Logger) *gin.Engine {
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
	router.Use(middleware.RateLimiter(logger))

	go middleware.StartBucketCleanup()

	router.GET("/health", handlers.Health.Health)
	router.GET("/categories", handlers.Category.List)

	auth := router.Group("/auth")
	{
		auth.POST("/request_otp", handlers.Auth.RequestOTP)
		auth.POST("/verify_otp", handlers.Auth.VerifyOTP)
		auth.POST("/sign_up", handlers.Auth.SignUp)
		auth.POST("/refresh", handlers.Auth.RefreshToken)
		auth.POST("/logout", middleware.AuthMiddleware(tokenValidator), handlers.Auth.Logout)
	}

	users := router.Group("/users", middleware.AuthMiddleware(tokenValidator))
	{
		users.GET("/current", handlers.User.CurrentUser)
		users.PUT("/profile", handlers.User.UpdateProfile)
		users.POST("/current/favorites", handlers.RoomUserFavourite.Add)
		users.DELETE("/current/favorites/:roomId", handlers.RoomUserFavourite.Remove)
	}

	tags := router.Group("/tags", middleware.AuthMiddleware(tokenValidator))
	{
		tags.GET("", handlers.Tag.List)
		tags.POST("", handlers.Tag.Create)
	}

	rooms := router.Group("/rooms", middleware.AuthMiddleware(tokenValidator))
	{
		rooms.GET("", handlers.Room.List)
		rooms.GET("/:id", handlers.Room.FindByID)
		rooms.POST("", handlers.Room.Create)
		rooms.PUT("/:id", handlers.Room.Update)
		rooms.GET("/:id/users", handlers.RoomUser.ListUsers)
		rooms.POST("/:id/users", handlers.RoomUser.AddRoomUser)
		rooms.GET("/:id/users/current", handlers.RoomUser.CurrentRoomUser)
		rooms.PUT("/:id/users/:userId/role", handlers.RoomUser.UpdateUserRole)
		rooms.PUT("/:id/private-code", handlers.Room.UpdatePrivateCode)
		rooms.POST("/:id/blocks", handlers.RoomUserBlock.Add)
		rooms.DELETE("/:id/blocks/:userId", handlers.RoomUserBlock.Remove)
	}

	router.GET("/ws/rooms/:roomId", middleware.AuthMiddleware(tokenValidator), handlers.WS.ServeWS)

	return router
}
