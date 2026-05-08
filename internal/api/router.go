package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/sanbei101/im/internal/api/handler"
	"github.com/sanbei101/im/internal/api/middleware"
)

func SetupRouter(userHandler *handler.UserHandler, messageHandler *handler.MessageHandler, roomHandler *handler.RoomHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"*"},
		AllowHeaders: []string{"*"},
	}))
	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("/register", userHandler.Register)
			users.POST("/login", userHandler.Login)
			users.POST("/batch", userHandler.BatchGenerate)
		}

		messages := v1.Group("/messages")
		{
			messages.Use(middleware.AuthMiddleware())
			messages.GET("/history", messageHandler.GetHistory)
		}

		rooms := v1.Group("/rooms")
		{
			rooms.Use(middleware.AuthMiddleware())
			rooms.POST("/single", roomHandler.CreateOrGetSingleChatRoom)
			rooms.POST("/group", roomHandler.CreateGroupRoom)
			rooms.POST("/list", roomHandler.ListRooms)
		}
	}

	return r
}
