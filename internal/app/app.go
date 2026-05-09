package app

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"psnadd/internal/config"
	"psnadd/internal/handlers"
	"psnadd/internal/middleware"
	"psnadd/internal/utils"
)

func NewRouter(cfg config.Config, logger *slog.Logger, version string, handler *handlers.Handler) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		logger.Warn("failed to set trusted proxies", "error", err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(cfg.CORSAllowOrigin))
	r.Use(middleware.AccessLog(logger))

	r.GET("/ping", func(c *gin.Context) {
		utils.WriteData(c, http.StatusOK, gin.H{
			"status":  "ok",
			"name":    cfg.Name,
			"env":     cfg.Env,
			"version": version,
		})
	})

	handler.Register(r)

	r.NoRoute(func(c *gin.Context) {
		utils.WriteError(c, utils.NotFound("endpoint not found"))
	})

	return r
}
