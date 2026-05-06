package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/sanbei101/im/internal/api/handler"
)

func SetupRouter(userHandler *handler.UserHandler, messageHandler *handler.MessageHandler, roomHandler *handler.RoomHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(cors.Default())
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
			messages.GET("/history", messageHandler.GetHistory)
		}

		rooms := v1.Group("/rooms")
		{
			rooms.POST("", roomHandler.CreateOrGetSingleChatRoom)
		}
	}

	return r
}
