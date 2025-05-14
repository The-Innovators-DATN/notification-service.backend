package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"notification-service/internal/db"
	"notification-service/internal/logging"
	"notification-service/internal/models"
	"notification-service/internal/services"
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
	svc    *services.Service
}

// NewHandler constructs a new API handler
func NewHandler(db *db.DB, logger *logging.Logger, svc *services.Service) *Handler {
	return &Handler{db: db, logger: logger, svc: svc}
}

// WebSocketHandler handles WebSocket connections with ping-pong mechanism
func (h *Handler) WebSocketHandler(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		h.logger.Errorf("user_id not found in context")
		c.JSON(http.StatusUnauthorized, StandardResponse{false, "unauthorized", nil})
		return
	}
	userID, err := strconv.Atoi(userIDStr.(string))
	if err != nil {
		h.logger.Errorf("invalid user_id: %v", err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid user_id", nil})
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf("WebSocket upgrade failed: %v", err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "failed to upgrade to WebSocket", nil})
		return
	}

	h.svc.AddWebSocketConnection(userID, conn)
	defer h.svc.RemoveWebSocketConnection(userID, conn)

	// Ping-pong mechanism
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				h.logger.Errorf("Ping failed for user %d: %v", userID, err)
				return
			}
		case <-c.Done():
			return
		}
	}
}

// CreateContactPoint creates and returns a new contact point
func (h *Handler) CreateContactPoint(c *gin.Context) {
	var input models.ContactPointCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Errorf("invalid create contact point payload: %v", err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid request body", nil})
		return
	}

	contactPoint := models.ContactPoint{
		Name:          input.Name,
		UserID:        input.UserID,
		Type:          input.Type,
		Configuration: input.Configuration,
		Status:        "active",
	}

	created, err := h.db.CreateContactPoint(c.Request.Context(), contactPoint)
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
	var input models.ContactPointUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Errorf("invalid update payload for contact point %s: %v", id, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid request body", nil})
		return
	}

	parsedPathID, err := uuid.Parse(id)
	if err != nil {
		h.logger.Errorf("invalid contact point ID %s: %v", id, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid contact point ID", nil})
		return
	}

	parsedInputID, err := uuid.Parse(input.ID)
	if err != nil {
		h.logger.Errorf("invalid input ID %s: %v", input.ID, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid input ID", nil})
		return
	}

	if parsedPathID != parsedInputID {
		h.logger.Errorf("path ID %s does not match input ID %s", id, input.ID)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "path ID does not match input ID", nil})
		return
	}

	existing, err := h.db.GetContactPointByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("contact point %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, StandardResponse{false, "contact point not found", nil})
		return
	}

	contactPoint := models.ContactPoint{
		ID:            existing.ID,
		Name:          existing.Name,
		UserID:        existing.UserID,
		Type:          existing.Type,
		Configuration: existing.Configuration,
		Status:        existing.Status,
		CreatedAt:     existing.CreatedAt,
		UpdatedAt:     existing.UpdatedAt,
	}

	if input.Name != "" {
		contactPoint.Name = input.Name
	}
	if input.UserID != nil && *input.UserID != 0 {
		contactPoint.UserID = *input.UserID
	}
	if input.Type != "" {
		contactPoint.Type = input.Type
	}
	if input.Configuration != nil {
		contactPoint.Configuration = input.Configuration
	}
	if input.Status != "" {
		contactPoint.Status = input.Status
	}

	copy(contactPoint.ID[:], parsedPathID[:])

	if err := h.db.UpdateContactPoint(c.Request.Context(), contactPoint); err != nil {
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
	var input models.PolicyCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Errorf("invalid create policy payload: %v", err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid request body", nil})
		return
	}

	parsedContactPointID, err := uuid.Parse(input.ContactPointID)
	if err != nil {
		h.logger.Errorf("invalid contact point ID %s: %v", input.ContactPointID, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid contact point ID", nil})
		return
	}

	policy := models.Policy{
		ContactPointID: parsedContactPointID,
		Severity:       input.Severity,
		Status:         "active",
		Action:         input.Action,
		ConditionType:  input.ConditionType,
	}

	policy, err = h.db.CreatePolicy(c.Request.Context(), policy)
	if err != nil {
		h.logger.Errorf("failed to create policy: %v", err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not create policy", nil})
		return
	}

	createdPolicy, err := h.db.GetPolicyByID(c.Request.Context(), uuid.UUID(policy.ID).String())
	if err != nil {
		h.logger.Errorf("policy created but fetch failed: %v", err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "policy created but retrieval failed", nil})
		return
	}

	h.logger.Infof("created policy %s", uuid.UUID(createdPolicy.ID).String())
	c.JSON(http.StatusCreated, StandardResponse{true, "policy created", createdPolicy})
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
	userId, err := strconv.ParseInt(c.Param("user_id"), 10, 32)
	if err != nil {
		h.logger.Errorf("invalid user_id %s: %v", c.Param("user_id"), err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid user_id", nil})
		return
	}

	list, err := h.db.GetPoliciesByUserID(c.Request.Context(), int(userId))
	if err != nil {
		h.logger.Errorf("could not list policies for user %d: %v", userId, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "failed to fetch policies", nil})
		return
	}

	h.logger.Infof("listed %d policies for user %d", len(list), userId)
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
	c.JSON(http.StatusOK, StandardResponse{true, "policy deleted", nil})
}

// UpdatePolicy updates an existing policy and returns it
func (h *Handler) UpdatePolicy(c *gin.Context) {
	id := c.Param("id")
	var input models.PolicyUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Errorf("invalid update payload for policy %s: %v", id, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid request body", nil})
		return
	}

	parsedPathID, err := uuid.Parse(id)
	if err != nil {
		h.logger.Errorf("invalid policy ID %s: %v", id, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid policy ID", nil})
		return
	}

	parsedInputID, err := uuid.Parse(input.ID)
	if err != nil {
		h.logger.Errorf("invalid input ID %s: %v", input.ID, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid input ID", nil})
		return
	}

	if parsedPathID != parsedInputID {
		h.logger.Errorf("path ID %s does not match input ID %s", id, input.ID)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "path ID does not match input ID", nil})
		return
	}

	parsedContactPointID, err := uuid.Parse(input.ContactPointID)
	if err != nil {
		h.logger.Errorf("invalid contact point ID %s: %v", input.ContactPointID, err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid contact point ID", nil})
		return
	}

	existing, err := h.db.GetPolicyByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("policy %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, StandardResponse{false, "policy not found", nil})
		return
	}

	policy := models.Policy{
		ID:             existing.ID,
		ContactPointID: parsedContactPointID,
		Severity:       existing.Severity,
		Status:         existing.Status,
		Action:         existing.Action,
		ConditionType:  existing.ConditionType,
		CreatedAt:      existing.CreatedAt,
		UpdatedAt:      existing.UpdatedAt,
	}

	if input.Severity != 0 {
		policy.Severity = input.Severity
	}
	if input.Status != "" {
		policy.Status = input.Status
	}
	if input.Action != "" {
		policy.Action = input.Action
	}
	if input.ConditionType != "" {
		policy.ConditionType = input.ConditionType
	}

	copy(policy.ID[:], parsedPathID[:])

	if err := h.db.UpdatePolicy(c.Request.Context(), policy); err != nil {
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

func (h *Handler) GetAlertByUserID(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.logger.Errorf("invalid user_id %s: %v", c.Param("user_id"), err)
		c.JSON(http.StatusBadRequest, StandardResponse{false, "invalid user_id", nil})
		return
	}

	status := c.DefaultQuery("status", "all")
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)

	items, total, err := h.db.GetAlertsByUserID(c.Request.Context(), int(uid), limit, offset, status)
	if err != nil {
		h.logger.Errorf("failed to list alerts for user %d: %v", uid, err)
		c.JSON(http.StatusInternalServerError, StandardResponse{false, "could not fetch alerts", nil})
		return
	}

	h.logger.Infof("listed %d alert for user %d (total %d)", len(items), uid, total)
	c.JSON(http.StatusOK, StandardResponse{true, "alert list", PaginatedResponse{total, items}})
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
