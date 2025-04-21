package api

import (
	"github.com/gin-gonic/gin"
	"notification-service/internal/config"
	"notification-service/internal/logging"
)

// NewRouter configures routes and middleware for the notification service API.
func NewRouter(logger *logging.Logger, cfg config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), RequestLoggingMiddleware(logger))

	rApi := r.Group(cfg.API.BasePath)
	// Health check
	rApi.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Contact-Points routes
	cp := rApi.Group("/contact-points")
	{
		cp.POST("", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.CreateContactPoint(c)
		}))
		cp.GET("/:id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.GetContactPoint(c)
		}))
		cp.GET("/user/:user_id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.GetContactPointsByUserID(c)
		}))
		cp.PUT("/:id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.UpdateContactPoint(c)
		}))
		cp.DELETE("/:id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.DeleteContactPoint(c)
		}))
	}

	// Policies routes
	pol := rApi.Group("/policies")
	{
		pol.POST("", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.CreatePolicy(c)
		}))
		pol.GET("/:id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.GetPolicy(c)
		}))
		pol.GET("/user/:user_id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.GetPoliciesByUserID(c)
		}))
		pol.PUT("/:id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.UpdatePolicy(c)
		}))
		pol.DELETE("/:id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.DeletePolicy(c)
		}))
	}

	// Notifications routes
	note := rApi.Group("/notifications")
	{
		note.GET("/user/:user_id", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.GetNotificationsByUserID(c)
		}))
		note.GET("", handlerWrapper(logger, func(c *gin.Context) {
			h := ctxHandler(c)
			h.GetAllNotifications(c)
		}))
	}

	return r
}

// ctxHandler extracts Handler instance from context
func ctxHandler(c *gin.Context) *Handler {
	// assume Handler was injected into context earlier if needed
	// or simply use closure-bound Handler in wrapper
	return c.MustGet("handler").(*Handler)
}

// handlerWrapper wraps a handler function with error handling and logger
func handlerWrapper(logger *logging.Logger, fn func(*gin.Context)) gin.HandlerFunc {
	return func(c *gin.Context) {
		fn(c)
	}
}
