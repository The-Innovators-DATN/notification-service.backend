package api

import (
	"github.com/gin-gonic/gin"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/logging"
)

// NewRouter khởi tạo router cho API
func NewRouter(db *db.DB, logger *logging.Logger, cfg config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(RequestLoggingMiddleware(logger))

	h := NewHandler(db, logger)
	api := r.Group("/api/v0")
	{
		// Contact Points
		api.POST("/contact-points", h.CreateContactPoint)
		api.GET("/contact-points/:id", h.GetContactPoint)
		api.GET("/contact-points/user/:user_id", h.GetContactPointsByUserID)

		// Policies
		api.POST("/policies", h.CreatePolicy)
		api.GET("/policies/:id", h.GetPolicy)
		api.GET("/policies/user/:user_id", h.GetPoliciesByUserID)

		// Notifications
		api.GET("/notifications/user/:user_id", h.GetSentNotificationsByUserID)
		api.GET("/notifications", h.GetAllNotifications)
	}
	return r
}
