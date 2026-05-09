package middleware

import "github.com/gin-gonic/gin"

func CORS(allowOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
