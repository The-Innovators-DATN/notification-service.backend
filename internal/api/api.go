package api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/logging"
	"notification-service/internal/models"
	"notification-service/pkg/telegram"
	"strings"
	"time"
)

type Handler struct {
	db     *db.DB
	logger *logging.Logger
	config config.Config
}

func NewRouter(db *db.DB, logger *logging.Logger, cfg config.Config) *gin.Engine {
	r := gin.Default()
	h := &Handler{db: db, logger: logger, config: cfg}

	basePath := cfg.API.BasePath

	r.Group(basePath)
	{
		r.POST("/contact-points", h.CreateContactPoint)
		r.POST("/policies", h.CreatePolicy)
		r.GET("/notifications", h.GetNotifications)
		r.GET("/notifications/:id", h.GetNotificationByID)
		r.POST("/notifications/retry/:id", h.RetryNotification)
		r.POST("/telegram/register", h.RegisterTelegram)
		r.POST("/sms/register", h.RegisterSMS)
		r.POST("/email/register", h.RegisterEmail)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

func (h *Handler) CreateContactPoint(c *gin.Context) {
	var cp models.ContactPoint
	if err := c.ShouldBindJSON(&cp); err != nil {
		h.logger.Error("", "Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cp.ID = uuid.New().String()
	cp.CreatedAt = time.Now()
	cp.Status = "active"

	if err := h.db.CreateContactPoint(c.Request.Context(), cp); err != nil {
		h.logger.Error("", "Create contact point failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("", "Created contact point: %s", cp.ID)
	c.JSON(http.StatusCreated, cp)
}

func (h *Handler) CreatePolicy(c *gin.Context) {
	var p models.Policy
	if err := c.ShouldBindJSON(&p); err != nil {
		h.logger.Error("", "Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.ID = uuid.New().String()
	p.CreatedAt = time.Now()
	p.Status = "active"

	if err := h.db.CreatePolicy(c.Request.Context(), p); err != nil {
		h.logger.Error("", "Create policy failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("", "Created policy: %s", p.ID)
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) GetNotifications(c *gin.Context) {
	notifications, err := h.db.GetAllNotifications(c.Request.Context())
	if err != nil {
		h.logger.Error("", "Get notifications failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("", "Retrieved %d notifications", len(notifications))
	c.JSON(http.StatusOK, notifications)
}

func (h *Handler) GetNotificationByID(c *gin.Context) {
	id := c.Param("id")
	notification, err := h.db.GetNotificationByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("", "Get notification %s failed: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	h.logger.Info("", "Retrieved notification %s", id)
	c.JSON(http.StatusOK, notification)
}

func (h *Handler) RetryNotification(c *gin.Context) {
	id := c.Param("id")
	notification, err := h.db.GetNotificationByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("", "Get notification %s failed: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	if notification.Status != "failed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Notification is not in failed state"})
		return
	}

	// Queue for retry
	h.logger.Info("", "Queuing retry for notification %s", id)
	// Simulate Kafka message for retry (simplified for demo)
	c.JSON(http.StatusOK, gin.H{"message": "Retry queued"})
}

func (h *Handler) RegisterTelegram(c *gin.Context) {
	type TelegramRequest struct {
		UserID         int64  `json:"user_id" binding:"required"`
		ChatID         int64  `json:"chat_id" binding:"required"`
		ContactPointID string `json:"contact_point_id" binding:"required"`
	}

	var req TelegramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("", "Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Test chat_id
	err := telegram.Send(h.config.Telegram.BotToken, []int64{req.ChatID}, "Welcome to Notification Service!")
	if err != nil {
		h.logger.Error("", "Invalid chat_id: %v", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot send message to this chat_id. Please start the bot first."})
		return
	}

	// Update contact point
	cp, err := h.db.GetContactPoint(c.Request.Context(), req.ContactPointID)
	if err != nil {
		h.logger.Error("", "Get contact point failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact point not found"})
		return
	}

	chatIDsRaw, ok := cp.Configuration["chat_ids"].([]interface{})
	if !ok {
		chatIDsRaw = []interface{}{}
	}
	for _, v := range chatIDsRaw {
		if int64(v.(float64)) == req.ChatID {
			c.JSON(http.StatusOK, gin.H{"message": "Chat ID already registered"})
			return
		}
	}
	chatIDsRaw = append(chatIDsRaw, req.ChatID)
	cp.Configuration["chat_ids"] = chatIDsRaw

	if err := h.db.CreateContactPoint(c.Request.Context(), cp); err != nil {
		h.logger.Error("", "Update contact point failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("", "Added chat_id %d to contact point %s", req.ChatID, cp.ID)
	c.JSON(http.StatusOK, cp)
}

func (h *Handler) RegisterSMS(c *gin.Context) {
	type SMSRequest struct {
		UserID         int64  `json:"user_id" binding:"required"`
		ToNumber       string `json:"to_number" binding:"required"`
		ContactPointID string `json:"contact_point_id" binding:"required"`
	}

	var req SMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("", "Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate phone number format (basic check)
	if !strings.HasPrefix(req.ToNumber, "+") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number must start with +"})
		return
	}

	// Update contact point
	cp, err := h.db.GetContactPoint(c.Request.Context(), req.ContactPointID)
	if err != nil {
		h.logger.Error("", "Get contact point failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact point not found"})
		return
	}

	toNumbersRaw, ok := cp.Configuration["to_numbers"].([]interface{})
	if !ok {
		toNumbersRaw = []interface{}{}
	}
	for _, v := range toNumbersRaw {
		if v.(string) == req.ToNumber {
			c.JSON(http.StatusOK, gin.H{"message": "Phone number already registered"})
			return
		}
	}
	toNumbersRaw = append(toNumbersRaw, req.ToNumber)
	cp.Configuration["to_numbers"] = toNumbersRaw

	if err := h.db.CreateContactPoint(c.Request.Context(), cp); err != nil {
		h.logger.Error("", "Update contact point failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("", "Added phone number %s to contact point %s", req.ToNumber, cp.ID)
	c.JSON(http.StatusOK, cp)
}

func (h *Handler) RegisterEmail(c *gin.Context) {
	type EmailRequest struct {
		UserID         int64  `json:"user_id" binding:"required"`
		Recipient      string `json:"recipient" binding:"required,email"`
		ContactPointID string `json:"contact_point_id" binding:"required"`
	}

	var req EmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("", "Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update contact point
	cp, err := h.db.GetContactPoint(c.Request.Context(), req.ContactPointID)
	if err != nil {
		h.logger.Error("", "Get contact point failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact point not found"})
		return
	}

	recipientsRaw, ok := cp.Configuration["recipients"].([]interface{})
	if !ok {
		recipientsRaw = []interface{}{}
	}
	for _, v := range recipientsRaw {
		if v.(string) == req.Recipient {
			c.JSON(http.StatusOK, gin.H{"message": "Email already registered"})
			return
		}
	}
	recipientsRaw = append(recipientsRaw, req.Recipient)
	cp.Configuration["recipients"] = recipientsRaw

	if err := h.db.CreateContactPoint(c.Request.Context(), cp); err != nil {
		h.logger.Error("", "Update contact point failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("", "Added email %s to contact point %s", req.Recipient, cp.ID)
	c.JSON(http.StatusOK, cp)
}
