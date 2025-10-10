package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/broker"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// Config contains configuration for handlers
type Config struct {
	Logger   *slog.Logger
	Broker   *broker.Broker
	Registry registry.Registry
}

// Handlers contains all HTTP handlers for the SSF broker
type Handlers struct {
	logger   *slog.Logger
	broker   *broker.Broker
	registry registry.Registry
}

// New creates new handlers
func New(config *Config) *Handlers {
	return &Handlers{
		logger:   config.Logger,
		broker:   config.Broker,
		registry: config.Registry,
	}
}

// HandleEvents handles incoming security events (SSF standard endpoint)
func (h *Handlers) HandleEvents(w http.ResponseWriter, r *http.Request) {
	// Get transmitter ID from request (could be from headers, JWT claims, etc.)
	transmitterID := h.getTransmitterID(r)
	if transmitterID == "" {
		h.logger.Error("No transmitter ID found in request")
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing transmitter identification")
		return
	}

	// Read the SET from request body
	rawSET, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	if len(rawSET) == 0 {
		h.logger.Error("Empty request body")
		h.writeErrorResponse(w, http.StatusBadRequest, "Empty request body")
		return
	}

	h.logger.Info("Received security event",
		"transmitter_id", transmitterID,
		"content_length", len(rawSET),
		"content_type", r.Header.Get("Content-Type"))

	// Process the security event
	if err := h.broker.ProcessSecurityEvent(r.Context(), string(rawSET), transmitterID); err != nil {
		h.logger.Error("Failed to process security event",
			"transmitter_id", transmitterID,
			"error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to process security event")
		return
	}

	// Return success response
	response := map[string]interface{}{
		"status":         "accepted",
		"transmitter_id": transmitterID,
		"timestamp":      fmt.Sprintf("%d", r.Context().Value("timestamp")),
	}

	h.writeJSONResponse(w, http.StatusAccepted, response)
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
		"events_delivery_endpoint":    fmt.Sprintf("%s/events", baseURL),
		"management_endpoint":         fmt.Sprintf("%s/api/v1", baseURL),
		"registration_endpoint":       fmt.Sprintf("%s/api/v1/receivers", baseURL),
		"subject_formats_supported": []string{
			models.SubjectFormatEmail,
			models.SubjectFormatPhoneNumber,
			models.SubjectFormatIssSub,
			models.SubjectFormatOpaque,
			models.SubjectFormatDID,
			models.SubjectFormatURI,
		},
		"specification_version": "1.0",
		"vendor":               "SSF Broker Service",
		"version":              "1.0.0",
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
		"status":           "ready",
		"receiver_count":   receiverCount,
		"timestamp":        fmt.Sprintf("%d", 1234567890), // Use actual timestamp
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// HandleMetrics handles Prometheus metrics requests
func (h *Handlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	// Get broker statistics
	stats, err := h.broker.GetBrokerStats()
	if err != nil {
		h.logger.Error("Failed to get broker stats", "error", err)
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
	var receiverReq models.ReceiverRequest

	if err := json.NewDecoder(r.Body).Decode(&receiverReq); err != nil {
		h.logger.Error("Failed to decode receiver request", "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Register the receiver
	receiver, err := h.broker.RegisterReceiver(r.Context(), &receiverReq)
	if err != nil {
		h.logger.Error("Failed to register receiver",
			"receiver_id", receiverReq.ID,
			"error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to register receiver: %v", err))
		return
	}

	h.logger.Info("Receiver registered via API",
		"receiver_id", receiver.ID,
		"event_types", receiver.EventTypes)

	h.writeJSONResponse(w, http.StatusCreated, receiver)
}

// HandleListReceivers handles listing all receivers
func (h *Handlers) HandleListReceivers(w http.ResponseWriter, r *http.Request) {
	receivers, err := h.broker.ListReceivers()
	if err != nil {
		h.logger.Error("Failed to list receivers", "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list receivers")
		return
	}

	// Apply query filters if provided
	statusFilter := r.URL.Query().Get("status")
	eventTypeFilter := r.URL.Query().Get("event_type")

	filteredReceivers := h.filterReceivers(receivers, statusFilter, eventTypeFilter)

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
	if receiverID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing receiver ID")
		return
	}

	receiver, err := h.broker.GetReceiver(receiverID)
	if err != nil {
		h.logger.Error("Failed to get receiver", "receiver_id", receiverID, "error", err)
		h.writeErrorResponse(w, http.StatusNotFound, "Receiver not found")
		return
	}

	// Include subscription information if requested
	includeSubscriptions := r.URL.Query().Get("include_subscriptions") == "true"
	response := map[string]interface{}{
		"receiver": receiver,
	}

	if includeSubscriptions {
		subscriptions, err := h.broker.GetReceiverSubscriptionInfo(r.Context(), receiverID)
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
	receiver, err := h.broker.UpdateReceiver(r.Context(), &receiverReq)
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
	if receiverID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing receiver ID")
		return
	}

	if err := h.broker.UnregisterReceiver(r.Context(), receiverID); err != nil {
		h.logger.Error("Failed to unregister receiver",
			"receiver_id", receiverID,
			"error", err)
		h.writeErrorResponse(w, http.StatusNotFound, "Receiver not found")
		return
	}

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
		"error":   message,
		"status":  statusCode,
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

// formatPrometheusMetrics formats broker statistics as Prometheus metrics
func (h *Handlers) formatPrometheusMetrics(stats *broker.BrokerStats) string {
	metrics := fmt.Sprintf(`# HELP ssf_broker_receivers_total Total number of registered receivers
# TYPE ssf_broker_receivers_total gauge
ssf_broker_receivers_total %d

`, stats.TotalReceivers)

	// Receivers by status
	for status, count := range stats.ReceiversByStatus {
		metrics += fmt.Sprintf(`# HELP ssf_broker_receivers_by_status Number of receivers by status
# TYPE ssf_broker_receivers_by_status gauge
ssf_broker_receivers_by_status{status="%s"} %d

`, status, count)
	}

	// Event type statistics
	for eventType, count := range stats.EventTypeStats {
		metrics += fmt.Sprintf(`# HELP ssf_broker_event_type_subscribers Number of receivers subscribed to event type
# TYPE ssf_broker_event_type_subscribers gauge
ssf_broker_event_type_subscribers{event_type="%s"} %d

`, eventType, count)
	}

	return metrics
}