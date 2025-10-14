package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/controller"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// Config contains configuration for handlers
type Config struct {
	Logger     *slog.Logger
	Controller *controller.Broker
	Registry   registry.Registry
}

// Handlers contains all HTTP handlers for the SSF hub
type Handlers struct {
	logger     *slog.Logger
	controller *controller.Broker
	registry   registry.Registry
}

// New creates new handlers
func New(config *Config) *Handlers {
	return &Handlers{
		logger:     config.Logger,
		controller: config.Controller,
		registry:   config.Registry,
	}
}

// HandleEvents handles incoming security events (SSF standard endpoint)
func (h *Handlers) HandleEvents(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Received SSF event request",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"),
		"content_type", r.Header.Get("Content-Type"))

	// Get transmitter ID from request (could be from headers, JWT claims, etc.)
	transmitterID := h.getTransmitterID(r)
	h.logger.Debug("Extracted transmitter ID", "transmitter_id", transmitterID)

	if transmitterID == "" {
		h.logger.Error("No transmitter ID found in request",
			"headers", map[string]string{
				"X-Transmitter-ID": r.Header.Get("X-Transmitter-ID"),
				"Authorization":    r.Header.Get("Authorization"),
			},
			"query_params", r.URL.Query())
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing transmitter identification")
		return
	}

	// Read the SET from request body
	h.logger.Debug("Reading request body")
	rawSET, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body",
			"error", err,
			"transmitter_id", transmitterID)
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	h.logger.Debug("Request body read successfully",
		"body_length", len(rawSET),
		"transmitter_id", transmitterID)

	if len(rawSET) == 0 {
		h.logger.Error("Empty request body",
			"transmitter_id", transmitterID,
			"content_length_header", r.Header.Get("Content-Length"))
		h.writeErrorResponse(w, http.StatusBadRequest, "Empty request body")
		return
	}

	h.logger.Info("Received security event",
		"transmitter_id", transmitterID,
		"content_length", len(rawSET),
		"content_type", r.Header.Get("Content-Type"))

	// Process the security event
	h.logger.Debug("Delegating to controller for processing",
		"transmitter_id", transmitterID,
		"body_length", len(rawSET))

	if err := h.controller.ProcessSecurityEvent(r.Context(), string(rawSET), transmitterID); err != nil {
		h.logger.Error("Failed to process security event",
			"transmitter_id", transmitterID,
			"error", err,
			"body_length", len(rawSET))
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to process security event")
		return
	}

	h.logger.Debug("Security event processed successfully by controller",
		"transmitter_id", transmitterID)

	// Return success response
	response := map[string]interface{}{
		"status":         "accepted",
		"transmitter_id": transmitterID,
		"timestamp":      fmt.Sprintf("%d", r.Context().Value("timestamp")),
	}

	h.logger.Debug("Sending success response",
		"transmitter_id", transmitterID,
		"response_status", "accepted")

	h.writeJSONResponse(w, http.StatusAccepted, response)

	h.logger.Debug("HandleEvents completed successfully",
		"transmitter_id", transmitterID,
		"response_code", http.StatusAccepted)
}

// HandleSSFConfiguration handles SSF metadata discovery (SSF standard endpoint)
func (h *Handlers) HandleSSFConfiguration(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)

	config := map[string]interface{}{
		"issuer":                     baseURL,
		"delivery_methods_supported": []string{"push", "pull", "urn:google:cloud:pubsub"},
		"critical_subject_members":   []string{"sub", "email", "phone_number"},
		"events_supported": []string{
			models.EventTypeSessionRevoked,
			models.EventTypeAssuranceLevelChange,
			models.EventTypeCredentialChange,
			models.EventTypeDeviceComplianceChange,
			models.EventTypeVerification,
		},
		"events_delivery_endpoint": fmt.Sprintf("%s/events", baseURL),
		"management_endpoint":      fmt.Sprintf("%s/api/v1", baseURL),
		"registration_endpoint":    fmt.Sprintf("%s/api/v1/receivers", baseURL),
		"subject_formats_supported": []string{
			models.SubjectFormatEmail,
			models.SubjectFormatPhoneNumber,
			models.SubjectFormatIssSub,
			models.SubjectFormatOpaque,
			models.SubjectFormatDID,
			models.SubjectFormatURI,
		},
		"specification_version": "1.0",
		"vendor":                "SSF Hub Service",
		"version":               "1.0.0",
	}

	h.writeJSONResponse(w, http.StatusOK, config)
}

// HandleHealth handles health check requests
func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": fmt.Sprintf("%d", 1234567890), // Use actual timestamp
		"version":   "1.0.0",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// HandleReady handles readiness check requests
func (h *Handlers) HandleReady(w http.ResponseWriter, r *http.Request) {
	// Check if critical services are ready
	receiverCount := h.registry.Count()

	response := map[string]interface{}{
		"status":         "ready",
		"receiver_count": receiverCount,
		"timestamp":      fmt.Sprintf("%d", 1234567890), // Use actual timestamp
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// HandleMetrics handles Prometheus metrics requests
func (h *Handlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	// Get controller statistics
	stats, err := h.controller.GetBrokerStats()
	if err != nil {
		h.logger.Error("Failed to get controller stats", "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get metrics")
		return
	}

	// Convert to Prometheus format
	metrics := h.formatPrometheusMetrics(stats)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(metrics))
}

// HandleRegisterReceiver handles receiver registration requests
func (h *Handlers) HandleRegisterReceiver(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Received receiver registration request",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
		"content_type", r.Header.Get("Content-Type"))

	var receiverReq models.ReceiverRequest

	if err := json.NewDecoder(r.Body).Decode(&receiverReq); err != nil {
		h.logger.Error("Failed to decode receiver request",
			"error", err,
			"content_type", r.Header.Get("Content-Type"),
			"remote_addr", r.RemoteAddr)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	h.logger.Debug("Receiver request decoded successfully",
		"receiver_id", receiverReq.ID,
		"name", receiverReq.Name,
		"event_types", receiverReq.EventTypes,
		"delivery_method", receiverReq.Delivery.Method)

	// Register the receiver
	h.logger.Debug("Delegating to controller for receiver registration",
		"receiver_id", receiverReq.ID)

	receiver, err := h.controller.RegisterReceiver(r.Context(), &receiverReq)
	if err != nil {
		h.logger.Error("Failed to register receiver",
			"receiver_id", receiverReq.ID,
			"error", err,
			"webhook_url", receiverReq.WebhookURL,
			"event_types", receiverReq.EventTypes)
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to register receiver: %v", err))
		return
	}

	h.logger.Debug("Receiver registered successfully by controller",
		"receiver_id", receiver.ID,
		"created_at", receiver.Metadata.CreatedAt)

	h.logger.Info("Receiver registered via API",
		"receiver_id", receiver.ID,
		"event_types", receiver.EventTypes)

	h.writeJSONResponse(w, http.StatusCreated, receiver)
}

// HandleListReceivers handles listing all receivers
func (h *Handlers) HandleListReceivers(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Received list receivers request",
		"method", r.Method,
		"query_params", r.URL.Query(),
		"remote_addr", r.RemoteAddr)

	receivers, err := h.controller.ListReceivers()
	if err != nil {
		h.logger.Error("Failed to list receivers",
			"error", err,
			"remote_addr", r.RemoteAddr)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list receivers")
		return
	}

	h.logger.Debug("Retrieved receivers from controller",
		"receiver_count", len(receivers))

	// Apply query filters if provided
	statusFilter := r.URL.Query().Get("status")
	eventTypeFilter := r.URL.Query().Get("event_type")

	h.logger.Debug("Applying filters to receiver list",
		"status_filter", statusFilter,
		"event_type_filter", eventTypeFilter,
		"total_receivers", len(receivers))

	filteredReceivers := h.filterReceivers(receivers, statusFilter, eventTypeFilter)

	h.logger.Debug("Filtering completed",
		"filtered_count", len(filteredReceivers),
		"original_count", len(receivers))

	response := map[string]interface{}{
		"receivers": filteredReceivers,
		"total":     len(filteredReceivers),
		"filters": map[string]string{
			"status":     statusFilter,
			"event_type": eventTypeFilter,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// HandleGetReceiver handles getting a specific receiver
func (h *Handlers) HandleGetReceiver(w http.ResponseWriter, r *http.Request) {
	receiverID := r.PathValue("id")
	h.logger.Debug("Received get receiver request",
		"receiver_id", receiverID,
		"query_params", r.URL.Query(),
		"remote_addr", r.RemoteAddr)

	if receiverID == "" {
		h.logger.Error("Missing receiver ID in path",
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr)
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing receiver ID")
		return
	}

	receiver, err := h.controller.GetReceiver(receiverID)
	if err != nil {
		h.logger.Error("Failed to get receiver",
			"receiver_id", receiverID,
			"error", err,
			"remote_addr", r.RemoteAddr)
		h.writeErrorResponse(w, http.StatusNotFound, "Receiver not found")
		return
	}

	h.logger.Debug("Retrieved receiver successfully",
		"receiver_id", receiverID,
		"status", receiver.Status,
		"event_types", receiver.EventTypes)

	// Include subscription information if requested
	includeSubscriptions := r.URL.Query().Get("include_subscriptions") == "true"
	response := map[string]interface{}{
		"receiver": receiver,
	}

	if includeSubscriptions {
		subscriptions, err := h.controller.GetReceiverSubscriptionInfo(r.Context(), receiverID)
		if err != nil {
			h.logger.Error("Failed to get subscription info", "receiver_id", receiverID, "error", err)
		} else {
			response["subscriptions"] = subscriptions
		}
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// HandleUpdateReceiver handles updating a receiver
func (h *Handlers) HandleUpdateReceiver(w http.ResponseWriter, r *http.Request) {
	receiverID := r.PathValue("id")
	if receiverID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing receiver ID")
		return
	}

	var receiverReq models.ReceiverRequest
	if err := json.NewDecoder(r.Body).Decode(&receiverReq); err != nil {
		h.logger.Error("Failed to decode receiver request", "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Ensure ID matches URL parameter
	receiverReq.ID = receiverID

	// Update the receiver
	receiver, err := h.controller.UpdateReceiver(r.Context(), &receiverReq)
	if err != nil {
		h.logger.Error("Failed to update receiver",
			"receiver_id", receiverID,
			"error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to update receiver: %v", err))
		return
	}

	h.logger.Info("Receiver updated via API", "receiver_id", receiverID)

	h.writeJSONResponse(w, http.StatusOK, receiver)
}

// HandleUnregisterReceiver handles unregistering a receiver
func (h *Handlers) HandleUnregisterReceiver(w http.ResponseWriter, r *http.Request) {
	receiverID := r.PathValue("id")
	h.logger.Debug("Received unregister receiver request",
		"receiver_id", receiverID,
		"method", r.Method,
		"remote_addr", r.RemoteAddr)

	if receiverID == "" {
		h.logger.Error("Missing receiver ID in unregister request",
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr)
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing receiver ID")
		return
	}

	if err := h.controller.UnregisterReceiver(r.Context(), receiverID); err != nil {
		h.logger.Error("Failed to unregister receiver",
			"receiver_id", receiverID,
			"error", err,
			"remote_addr", r.RemoteAddr)
		h.writeErrorResponse(w, http.StatusNotFound, "Receiver not found")
		return
	}

	h.logger.Debug("Receiver unregistered successfully",
		"receiver_id", receiverID)

	h.logger.Info("Receiver unregistered via API", "receiver_id", receiverID)

	response := map[string]interface{}{
		"status":      "unregistered",
		"receiver_id": receiverID,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper methods

// getTransmitterID extracts transmitter ID from request
func (h *Handlers) getTransmitterID(r *http.Request) string {
	// Check for development mode bypass
	if h.isDevMode(r) {
		h.logger.Debug("Development mode detected, using dev auth bypass")

		// In dev mode, prefer X-Transmitter-ID header
		if transmitterID := r.Header.Get("X-Transmitter-ID"); transmitterID != "" {
			h.logger.Debug("Using transmitter ID from dev mode header", "transmitter_id", transmitterID)
			return transmitterID
		}

		// Fall back to environment variable or default dev transmitter
		if devTransmitter := os.Getenv("DEV_DEFAULT_TRANSMITTER"); devTransmitter != "" {
			h.logger.Debug("Using transmitter ID from DEV_DEFAULT_TRANSMITTER", "transmitter_id", devTransmitter)
			return devTransmitter
		}

		h.logger.Debug("Using default dev transmitter ID")
		return "dev-transmitter"
	}

	// Try to get from custom header first
	if transmitterID := r.Header.Get("X-Transmitter-ID"); transmitterID != "" {
		return transmitterID
	}

	// Try to get from Authorization header (extract from JWT claims)
	// In a real implementation, you would decode the JWT and extract
	// the transmitter ID from the claims
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		return "jwt-transmitter" // Placeholder
	}

	// Try to get from query parameter
	if transmitterID := r.URL.Query().Get("transmitter_id"); transmitterID != "" {
		return transmitterID
	}

	// Default transmitter ID for demo purposes
	return "default-transmitter"
}

// isDevMode checks if development mode is enabled
func (h *Handlers) isDevMode(r *http.Request) bool {
	// Check environment variable
	if os.Getenv("DEV_DEBUG") == "true" {
		return true
	}

	// Check request header
	if r.Header.Get("X-Dev-Mode") == "true" {
		return true
	}

	return false
}

// getBaseURL constructs the base URL from the request
func (h *Handlers) getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// writeJSONResponse writes a JSON response
func (h *Handlers) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// writeErrorResponse writes an error response
func (h *Handlers) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"error":  message,
		"status": statusCode,
	}

	h.writeJSONResponse(w, statusCode, response)
}

// filterReceivers applies filters to receivers list
func (h *Handlers) filterReceivers(receivers []*models.Receiver, statusFilter, eventTypeFilter string) []*models.Receiver {
	if statusFilter == "" && eventTypeFilter == "" {
		return receivers
	}

	var filtered []*models.Receiver

	for _, receiver := range receivers {
		// Apply status filter
		if statusFilter != "" && string(receiver.Status) != statusFilter {
			continue
		}

		// Apply event type filter
		if eventTypeFilter != "" {
			hasEventType := false
			for _, eventType := range receiver.EventTypes {
				if eventType == eventTypeFilter {
					hasEventType = true
					break
				}
			}
			if !hasEventType {
				continue
			}
		}

		filtered = append(filtered, receiver)
	}

	return filtered
}

// formatPrometheusMetrics formats controller statistics as Prometheus metrics
func (h *Handlers) formatPrometheusMetrics(stats *controller.BrokerStats) string {
	metrics := fmt.Sprintf(`# HELP ssf_hub_receivers_total Total number of registered receivers
# TYPE ssf_hub_receivers_total gauge
ssf_hub_receivers_total %d

`, stats.TotalReceivers)

	// Receivers by status
	for status, count := range stats.ReceiversByStatus {
		metrics += fmt.Sprintf(`# HELP ssf_hub_receivers_by_status Number of receivers by status
# TYPE ssf_hub_receivers_by_status gauge
ssf_hub_receivers_by_status{status="%s"} %d

`, status, count)
	}

	// Event type statistics
	for eventType, count := range stats.EventTypeStats {
		metrics += fmt.Sprintf(`# HELP ssf_hub_event_type_subscribers Number of receivers subscribed to event type
# TYPE ssf_hub_event_type_subscribers gauge
ssf_hub_event_type_subscribers{event_type="%s"} %d

`, eventType, count)
	}

	return metrics
}
