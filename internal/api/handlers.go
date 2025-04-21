package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"notification-service/internal/db"
	"notification-service/internal/logging"
	"notification-service/internal/models"
)

type StandardResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PaginatedResponse struct {
	Total int         `json:"total"`
	Items interface{} `json:"items"`
}

// Handler groups dependencies for HTTP handlers
type Handler struct {
	db     *db.DB
	logger *logging.Logger
}

// NewHandler constructs a new API handler
func NewHandler(db *db.DB, logger *logging.Logger) *Handler {
	return &Handler{db: db, logger: logger}
}

// CreateContactPoint creates and returns a new contact point
func (h *Handler) CreateContactPoint(c *gin.Context) {
	var input models.ContactPoint
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Errorf("invalid create contact point payload: %v", err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid request body", nil})
		return
	}

	created, err := h.db.CreateContactPoint(c.Request.Context(), input)
	if err != nil {
		h.logger.Errorf("failed to create contact point: %v", err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not create contact point", nil})
		return
	}

	h.logger.Infof("created contact point %s", uuid.UUID(created.ID).String())
	c.JSON(http.StatusCreated, StandardResponse{true, "contact point created", created})
}

// GetContactPoint retrieves a single active contact point by UUID
func (h *Handler) GetContactPoint(c *gin.Context) {
	id := c.Param("id")
	cp, err := h.db.GetContactPointByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("contact point %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, StandardResponse{false, "contact point not found", nil})
		return
	}

	h.logger.Infof("retrieved contact point %s", id)
	c.JSON(http.StatusOK, StandardResponse{true, "contact point retrieved", cp})
}

// GetContactPointsByUserID lists active contact points for a user
func (h *Handler) GetContactPointsByUserID(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.logger.Errorf("invalid user_id %s: %v", c.Param("user_id"), err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid user_id", nil})
		return
	}

	list, err := h.db.GetContactPointsByUserID(c.Request.Context(), uid)
	if err != nil {
		h.logger.Errorf("could not list contact points for user %d: %v", uid, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "failed to fetch contact points", nil})
		return
	}

	h.logger.Infof("listed %d contact points for user %d", len(list), uid)
	c.JSON(http.StatusOK, StandardResponse{true, "contact points list", list})
}

// DeleteContactPoint marks a contact point as deleted
func (h *Handler) DeleteContactPoint(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteContactPoint(c.Request.Context(), id); err != nil {
		h.logger.Errorf("failed to delete contact point %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not delete contact point", nil})
		return
	}

	h.logger.Infof("deleted contact point %s", id)
	c.Status(http.StatusNoContent)
}

// UpdateContactPoint updates fields of an active contact point
func (h *Handler) UpdateContactPoint(c *gin.Context) {
	id := c.Param("id")
	var input models.ContactPoint
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Errorf("invalid update payload for contact point %s: %v", id, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid request body", nil})
		return
	}

	// ensure ID matches path
	if parsed, err := uuid.Parse(id); err == nil {
		copy(input.ID[:], parsed[:])
	} else {
		h.logger.Errorf("invalid contact point ID %s: %v", id, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid contact point ID", nil})
		return
	}

	if err := h.db.UpdateContactPoint(c.Request.Context(), input); err != nil {
		h.logger.Errorf("failed to update contact point %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not update contact point", nil})
		return
	}

	updated, err := h.db.GetContactPointByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("failed to fetch updated contact point %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "update succeeded but retrieval failed", nil})
		return
	}

	h.logger.Infof("updated contact point %s", id)
	c.JSON(http.StatusOK, StandardResponse{true, "contact point updated", updated})
}

// CreatePolicy creates a new policy and returns it
func (h *Handler) CreatePolicy(c *gin.Context) {
	var input models.Policy
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Errorf("invalid create policy payload: %v", err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid request body", nil})
		return
	}

	// let DB handle ID generation
	if err := h.db.CreatePolicy(c.Request.Context(), input); err != nil {
		h.logger.Errorf("failed to create policy: %v", err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not create policy", nil})
		return
	}

	// fetch created resource
	policy, err := h.db.GetPolicyByID(c.Request.Context(), uuid.UUID(input.ID).String())
	if err != nil {
		h.logger.Errorf("policy created but fetch failed: %v", err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "policy created but retrieval failed", nil})
		return
	}

	h.logger.Infof("created policy %s", uuid.UUID(policy.ID).String())
	c.JSON(http.StatusCreated, StandardResponse{true, "policy created", policy})
}

// GetPolicy retrieves an active policy
func (h *Handler) GetPolicy(c *gin.Context) {
	id := c.Param("id")
	policy, err := h.db.GetPolicyByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("policy %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, StandardResponse{false, "policy not found", nil})
		return
	}

	h.logger.Infof("retrieved policy %s", id)
	c.JSON(http.StatusOK, StandardResponse{true, "policy retrieved", policy})
}

// GetPoliciesByUserID lists active policies for a user
func (h *Handler) GetPoliciesByUserID(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.logger.Errorf("invalid user_id %s: %v", c.Param("user_id"), err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid user_id", nil})
		return
	}

	list, err := h.db.GetPoliciesByUserID(c.Request.Context(), uid)
	if err != nil {
		h.logger.Errorf("could not list policies for user %d: %v", uid, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "failed to fetch policies", nil})
		return
	}

	h.logger.Infof("listed %d policies for user %d", len(list), uid)
	c.JSON(http.StatusOK, StandardResponse{true, "policies list", list})
}

// DeletePolicy marks a policy inactive
func (h *Handler) DeletePolicy(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeletePolicy(c.Request.Context(), id); err != nil {
		h.logger.Errorf("failed to delete policy %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not delete policy", nil})
		return
	}

	h.logger.Infof("deleted policy %s", id)
	c.JSON(http.StatusOK, StandardResponse{true, "policies deleted", nil})
}

// UpdatePolicy updates an existing policy and returns it
func (h *Handler) UpdatePolicy(c *gin.Context) {
	id := c.Param("id")
	var input models.Policy
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Errorf("invalid update payload for policy %s: %v", id, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid request body", nil})
		return
	}

	// enforce path ID
	if parsed, err := uuid.Parse(id); err == nil {
		copy(input.ID[:], parsed[:])
	} else {
		h.logger.Errorf("invalid policy ID %s: %v", id, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid policy ID", nil})
		return
	}

	if err := h.db.UpdatePolicy(c.Request.Context(), input); err != nil {
		h.logger.Errorf("failed to update policy %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not update policy", nil})
		return
	}

	updated, err := h.db.GetPolicyByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("update succeeded but retrieval failed for %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "update succeeded but retrieval failed", nil})
		return
	}

	h.logger.Infof("updated policy %s", id)
	c.JSON(http.StatusOK, StandardResponse{true, "policy updated", updated})
}

// GetNotificationsByUserID lists notifications with pagination
func (h *Handler) GetNotificationsByUserID(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.logger.Errorf("invalid user_id %s: %v", c.Param("user_id"), err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid user_id", nil})
		return
	}

	status := c.DefaultQuery("status", "all")
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)

	items, total, err := h.db.GetNotificationsByUserID(c.Request.Context(), int(uid), limit, offset, status)
	if err != nil {
		h.logger.Errorf("failed to list notifications for user %d: %v", uid, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not fetch notifications", nil})
		return
	}

	h.logger.Infof("listed %d notifications for user %d (total %d)", len(items), uid, total)
	c.JSON(http.StatusOK, StandardResponse{true, "notifications list", PaginatedResponse{total, items}})
}

// GetAllNotifications lists all notifications with pagination
func (h *Handler) GetAllNotifications(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)

	items, total, err := h.db.GetAllNotifications(c.Request.Context(), status, limit, offset)
	if err != nil {
		h.logger.Errorf("failed to list all notifications: %v", err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not fetch notifications", nil})
		return
	}

	h.logger.Infof("listed %d notifications (total %d)", len(items), total)
	c.JSON(http.StatusOK, StandardResponse{true, "all notifications list", PaginatedResponse{total, items}})
}

// parseQueryInt is a helper to read integer query params with default
func parseQueryInt(c *gin.Context, key string, def int) int {
	if v := c.DefaultQuery(key, ""); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return def
}
