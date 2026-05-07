package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/pkg/jwt"
)

const UserIDKey = "user_id"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Error().Str("path", c.Request.URL.Path).Msg("api missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			c.Abort()
			return
		}

		if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
			authHeader = authHeader[7:]
		}

		userID, err := jwt.ParseToken(authHeader)
		if err != nil {
			log.Error().Err(err).Msg("api parse token failed")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set(UserIDKey, userID)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return ""
	}
	return userID.(string)
}
