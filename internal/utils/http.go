package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func WriteData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func WriteError(c *gin.Context, err error) {
	appErr := AsAppError(err)
	c.JSON(appErr.Status, gin.H{
		"error": gin.H{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	})
}

func WantsJSON(c *gin.Context) bool {
	return c.GetHeader("Accept") == "application/json" || c.ContentType() == "application/json"
}

func RedirectBack(c *gin.Context, fallback string) {
	target := c.GetHeader("Referer")
	if target == "" {
		target = fallback
	}
	c.Redirect(http.StatusSeeOther, target)
}
