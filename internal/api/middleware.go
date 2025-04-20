package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"notification-service/internal/logging"
)

// RequestLoggingMiddleware logs incoming HTTP requests with latency, status code, client IP, and user-agent.
func RequestLoggingMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate metrics
		latency := time.Since(start)
		status := c.Writer.Status()

		// Use route pattern if available, fallback to URL path
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Additional context
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Structured log message
		logger.Infof("%s %s %s %d %v %s", clientIP, c.Request.Method, path, status, latency, userAgent)
	}
}
