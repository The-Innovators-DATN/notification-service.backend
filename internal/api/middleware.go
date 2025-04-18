package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"notification-service/internal/logging"
)

func RequestLoggingMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		logger.Infof("Request: %s %s, Status: %d, Latency: %v", method, path, status, latency)
	}
}
