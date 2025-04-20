package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"notification-service/internal/db"
	"notification-service/internal/logging"
	"notification-service/internal/models"
)

type Handler struct {
	db     *db.DB
	logger *logging.Logger
}

func NewHandler(db *db.DB, logger *logging.Logger) *Handler {
	return &Handler{db: db, logger: logger}
}

// Contact Point
func (h *Handler) CreateContactPoint(c *gin.Context) {
	var cp models.ContactPoint
	if err := c.ShouldBindJSON(&cp); err != nil {
		h.logger.Errorf("Invalid request body for contact point: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if _, err := h.db.CreateContactPoint(c.Request.Context(), cp); err != nil {
		h.logger.Errorf("Failed to create contact point: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contact point"})
		return
	}

	h.logger.Infof("Created contact point: %s", cp.ID)
	c.JSON(http.StatusCreated, cp)
}

func (h *Handler) GetContactPoint(c *gin.Context) {
	id := c.Param("id")
	cp, err := h.db.GetContactPointById(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get contact point %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact point not found"})
		return
	}

	h.logger.Infof("Retrieved contact point: %s", id)
	c.JSON(http.StatusOK, cp)
}

func (h *Handler) GetContactPointsByUserID(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Errorf("Invalid user_id %s: %v", userIDStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	cps, err := h.db.GetContactPointsByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorf("Failed to get contact points for user_id %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contact points"})
		return
	}

	h.logger.Infof("Retrieved %d contact points for user_id %d", len(cps), userID)
	c.JSON(http.StatusOK, cps)
}

func (h *Handler) DeleteContactPoint(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteContactPoint(c.Request.Context(), id); err != nil {
		h.logger.Errorf("Failed to delete contact point %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contact point"})
		return
	}

	h.logger.Infof("Deleted contact point: %s", id)
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) UpdateContactPoint(c *gin.Context) {
	id := c.Param("id")
	var cp models.ContactPoint
	if err := c.ShouldBindJSON(&cp); err != nil {
		h.logger.Errorf("Invalid request body for contact point: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.db.UpdateContactPoint(c.Request.Context(), cp); err != nil {
		h.logger.Errorf("Failed to update contact point %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contact point"})
		return
	}

	h.logger.Infof("Updated contact point: %s", id)
	c.JSON(http.StatusOK, cp)
}

// Notification Policy
func (h *Handler) CreatePolicy(c *gin.Context) {
	var policy models.Policy
	if err := c.ShouldBindJSON(&policy); err != nil {
		h.logger.Errorf("Invalid request body for policy: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.db.CreatePolicy(c.Request.Context(), policy); err != nil {
		h.logger.Errorf("Failed to create policy: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create policy"})
		return
	}

	h.logger.Infof("Created policy: %s", policy.ID)
	c.JSON(http.StatusCreated, policy)
}

func (h *Handler) GetPolicy(c *gin.Context) {
	id := c.Param("id")
	policy, err := h.db.GetPolicyById(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get policy %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	h.logger.Infof("Retrieved policy: %s", id)
	c.JSON(http.StatusOK, policy)
}

func (h *Handler) GetPoliciesByUserID(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Errorf("Invalid user_id %s: %v", userIDStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	policies, err := h.db.GetPoliciesByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorf("Failed to get policies for user_id %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get policies"})
		return
	}

	h.logger.Infof("Retrieved %d policies for user_id %d", len(policies), userID)
	c.JSON(http.StatusOK, policies)
}

func (h *Handler) DeletePolicy(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeletePolicy(c.Request.Context(), id); err != nil {
		h.logger.Errorf("Failed to delete policy %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete policy"})
		return
	}

	h.logger.Infof("Deleted policy: %s", id)
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) UpdatePolicy(c *gin.Context) {
	id := c.Param("id")
	var policy models.Policy
	if err := c.ShouldBindJSON(&policy); err != nil {
		h.logger.Errorf("Invalid request body for policy: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.db.UpdatePolicy(c.Request.Context(), policy); err != nil {
		h.logger.Errorf("Failed to update policy %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update policy"})
		return
	}

	h.logger.Infof("Updated policy: %s", id)
	c.JSON(http.StatusOK, policy)
}

// Notifications
func (h *Handler) GetNotificationsByUserID(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		h.logger.Errorf("Invalid user_id %s: %v", userIDStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	notifications, err := h.db.GetNotificationsByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorf("Failed to get sent notifications for user_id %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}

	h.logger.Infof("Retrieved %d sent notifications for user_id %d", len(notifications), userID)
	c.JSON(http.StatusOK, notifications)
}

func (h *Handler) GetAllNotifications(c *gin.Context) {
	notifications, err := h.db.GetAllNotifications(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Failed to get all notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get all notifications"})
		return
	}

	h.logger.Infof("Retrieved %d notifications", len(notifications))
	c.JSON(http.StatusOK, notifications)
}
