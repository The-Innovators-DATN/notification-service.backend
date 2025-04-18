package api

import (
	"github.com/gin-gonic/gin"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/logging"
)

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
		api.DELETE("/contact-points/:id", h.DeleteContactPoint)
		api.PUT("/contact-points/:id", h.UpdateContactPoint)

		// Policies
		api.POST("/policies", h.CreatePolicy)
		api.GET("/policies/:id", h.GetPolicy)
		api.GET("/policies/user/:user_id", h.GetPoliciesByUserID)
		api.DELETE("/policies/:id", h.DeletePolicy)
		api.PUT("/policies/:id", h.UpdatePolicy)

		// Notifications
		api.GET("/notifications/user/:user_id", h.GetNotificationsByUserID)
		//api.GET("/notifications", h.GetAllNotifications)
	}
	return r
}
