package middleware

import (
	"github.com/gin-gonic/gin"

	"psnadd/internal/utils"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			token, err := utils.NewToken()
			if err == nil {
				requestID = token[:16]
			}
		}
		if requestID != "" {
			c.Header("X-Request-ID", requestID)
			c.Set("request_id", requestID)
		}
		c.Next()
	}
}
