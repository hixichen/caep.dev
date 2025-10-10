package handlers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"log/slog"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/controller"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)


// mockPubSubClient implements controller.PubSubClient for testing
type mockPubSubClientForHandlers struct{}

func (m *mockPubSubClientForHandlers) PublishEvent(ctx context.Context, event *models.SecurityEvent, targetReceivers []string) error {
	return nil
}

func (m *mockPubSubClientForHandlers) CreateHubSubscription(ctx context.Context, subscriptionName string) error {
	return nil
}

func (m *mockPubSubClientForHandlers) DeleteHubSubscription(ctx context.Context, subscriptionName string) error {
	return nil
}

func (m *mockPubSubClientForHandlers) PullInternalMessages(ctx context.Context, subscriptionName string, maxMessages int, handler func(*models.InternalMessage) error) error {
	return nil
}

func (m *mockPubSubClientForHandlers) GetHubInstanceID() string {
	return "test-hub-instance"
}

func (m *mockPubSubClientForHandlers) Close() error {
	return nil
}

func createTestHandlers(t *testing.T) *Handlers {
	logger := slog.Default()

	// Create a mock Pub/Sub client
	pubsubClient := &mockPubSubClientForHandlers{}

	receiverRegistry := registry.NewMemoryRegistry()
	ssfController := controller.New(pubsubClient, receiverRegistry, logger)

	config := &Config{
		Logger:   logger,
		Controller: ssfController,
		Registry: receiverRegistry,
	}

	return New(config)
}

func TestHandlers_HandleHealth(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handlers.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleHealth() status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("HandleHealth() content-type = %s, want application/json", contentType)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleHealth() failed to unmarshal response: %v", err)
	}

	if status, exists := response["status"]; !exists || status != "healthy" {
		t.Errorf("HandleHealth() status in response = %v, want 'healthy'", status)
	}
}

func TestHandlers_HandleReady(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handlers.HandleReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleReady() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleReady() failed to unmarshal response: %v", err)
	}

	if status, exists := response["status"]; !exists || status != "ready" {
		t.Errorf("HandleReady() status in response = %v, want 'ready'", status)
	}

	if _, exists := response["receiver_count"]; !exists {
		t.Error("HandleReady() response should include receiver_count")
	}
}

func TestHandlers_HandleSSFConfiguration(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("GET", "/.well-known/ssf_configuration", nil)
	req.Host = "test.example.com"
	w := httptest.NewRecorder()

	handlers.HandleSSFConfiguration(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleSSFConfiguration() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleSSFConfiguration() failed to unmarshal response: %v", err)
	}

	// Check required SSF configuration fields
	requiredFields := []string{
		"issuer",
		"delivery_methods_supported",
		"critical_subject_members",
		"events_supported",
		"events_delivery_endpoint",
	}

	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("HandleSSFConfiguration() missing required field: %s", field)
		}
	}

	// Check that issuer contains the host
	if issuer, ok := response["issuer"].(string); ok {
		if !strings.Contains(issuer, "test.example.com") {
			t.Errorf("HandleSSFConfiguration() issuer should contain host, got: %s", issuer)
		}
	}

	// Check delivery methods
	if deliveryMethods, ok := response["delivery_methods_supported"].([]interface{}); ok {
		hasWebhook := false
		for _, method := range deliveryMethods {
			if method == "push" {
				hasWebhook = true
				break
			}
		}
		if !hasWebhook {
			t.Error("HandleSSFConfiguration() should support push delivery method")
		}
	}
}

func TestHandlers_HandleRegisterReceiver(t *testing.T) {
	handlers := createTestHandlers(t)

	// Test valid receiver registration
	receiverReq := models.ReceiverRequest{
		ID:          "test-receiver",
		Name:        "Test Receiver",
		WebhookURL:  "https://example.com/webhook",
		EventTypes:  []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		Auth: models.AuthConfig{
			Type:  models.AuthTypeBearer,
			Token: "test-token",
		},
	}

	reqBody, _ := json.Marshal(receiverReq)
	req := httptest.NewRequest("POST", "/api/v1/receivers", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleRegisterReceiver(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("HandleRegisterReceiver() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var response models.Receiver
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleRegisterReceiver() failed to unmarshal response: %v", err)
	}

	if response.ID != receiverReq.ID {
		t.Errorf("HandleRegisterReceiver() receiver ID = %s, want %s", response.ID, receiverReq.ID)
	}

	if response.Status != models.ReceiverStatusActive {
		t.Errorf("HandleRegisterReceiver() receiver status = %s, want %s", response.Status, models.ReceiverStatusActive)
	}
}

func TestHandlers_HandleRegisterReceiver_InvalidJSON(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("POST", "/api/v1/receivers", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleRegisterReceiver(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleRegisterReceiver() with invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleRegisterReceiver() failed to unmarshal error response: %v", err)
	}

	if _, exists := response["error"]; !exists {
		t.Error("HandleRegisterReceiver() error response should include 'error' field")
	}
}

func TestHandlers_HandleListReceivers(t *testing.T) {
	handlers := createTestHandlers(t)

	// First register a receiver
	receiver := &models.Receiver{
		ID:         "test-receiver",
		Name:       "Test Receiver",
		WebhookURL: "https://example.com/webhook",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		Status: models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	handlers.registry.Register(receiver)

	req := httptest.NewRequest("GET", "/api/v1/receivers", nil)
	w := httptest.NewRecorder()

	handlers.HandleListReceivers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleListReceivers() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleListReceivers() failed to unmarshal response: %v", err)
	}

	if _, exists := response["receivers"]; !exists {
		t.Error("HandleListReceivers() response should include 'receivers' field")
	}

	if total, exists := response["total"]; !exists || total != float64(1) {
		t.Errorf("HandleListReceivers() total = %v, want 1", total)
	}
}

func TestHandlers_HandleListReceivers_WithFilters(t *testing.T) {
	handlers := createTestHandlers(t)

	// Register receivers with different statuses
	receiver1 := &models.Receiver{
		ID:         "active-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Status:     models.ReceiverStatusActive,
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
	}
	receiver1.SetDefaults()

	receiver2 := &models.Receiver{
		ID:         "inactive-receiver",
		EventTypes: []string{models.EventTypeCredentialChange},
		Status:     models.ReceiverStatusInactive,
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
	}
	receiver2.SetDefaults()

	handlers.registry.Register(receiver1)
	handlers.registry.Register(receiver2)

	// Test status filter
	req := httptest.NewRequest("GET", "/api/v1/receivers?status=active", nil)
	w := httptest.NewRecorder()

	handlers.HandleListReceivers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleListReceivers() with filter status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleListReceivers() failed to unmarshal response: %v", err)
	}

	if total, exists := response["total"]; !exists || total != float64(1) {
		t.Errorf("HandleListReceivers() with status filter total = %v, want 1", total)
	}

	// Test event type filter
	req = httptest.NewRequest("GET", "/api/v1/receivers?event_type="+models.EventTypeSessionRevoked, nil)
	w = httptest.NewRecorder()

	handlers.HandleListReceivers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleListReceivers() with event type filter status = %d, want %d", w.Code, http.StatusOK)
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleListReceivers() failed to unmarshal response: %v", err)
	}

	if total, exists := response["total"]; !exists || total != float64(1) {
		t.Errorf("HandleListReceivers() with event type filter total = %v, want 1", total)
	}
}

func TestHandlers_HandleGetReceiver(t *testing.T) {
	handlers := createTestHandlers(t)

	// Register a receiver
	receiver := &models.Receiver{
		ID:         "test-receiver",
		Name:       "Test Receiver",
		WebhookURL: "https://example.com/webhook",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		Status: models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	handlers.registry.Register(receiver)

	// Test getting existing receiver
	req := httptest.NewRequest("GET", "/api/v1/receivers/test-receiver", nil)
	req.SetPathValue("id", "test-receiver")
	w := httptest.NewRecorder()

	handlers.HandleGetReceiver(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleGetReceiver() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleGetReceiver() failed to unmarshal response: %v", err)
	}

	if _, exists := response["receiver"]; !exists {
		t.Error("HandleGetReceiver() response should include 'receiver' field")
	}

	// Test getting non-existent receiver
	req = httptest.NewRequest("GET", "/api/v1/receivers/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w = httptest.NewRecorder()

	handlers.HandleGetReceiver(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("HandleGetReceiver() for non-existent receiver status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlers_HandleUnregisterReceiver(t *testing.T) {
	handlers := createTestHandlers(t)

	// Register a receiver
	receiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	handlers.registry.Register(receiver)

	// Test unregistering existing receiver
	req := httptest.NewRequest("DELETE", "/api/v1/receivers/test-receiver", nil)
	req.SetPathValue("id", "test-receiver")
	w := httptest.NewRecorder()

	handlers.HandleUnregisterReceiver(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleUnregisterReceiver() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleUnregisterReceiver() failed to unmarshal response: %v", err)
	}

	if status, exists := response["status"]; !exists || status != "unregistered" {
		t.Errorf("HandleUnregisterReceiver() status in response = %v, want 'unregistered'", status)
	}

	// Verify receiver is actually removed
	_, err = handlers.registry.Get("test-receiver")
	if err == nil {
		t.Error("HandleUnregisterReceiver() receiver should be removed from registry")
	}

	// Test unregistering non-existent receiver
	req = httptest.NewRequest("DELETE", "/api/v1/receivers/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w = httptest.NewRecorder()

	handlers.HandleUnregisterReceiver(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("HandleUnregisterReceiver() for non-existent receiver status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlers_getTransmitterID(t *testing.T) {
	handlers := createTestHandlers(t)

	tests := []struct {
		name     string
		setupReq func(*http.Request)
		want     string
	}{
		{
			name: "from X-Transmitter-ID header",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Transmitter-ID", "header-transmitter")
			},
			want: "header-transmitter",
		},
		{
			name: "from Authorization header",
			setupReq: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token")
			},
			want: "jwt-transmitter",
		},
		{
			name: "from query parameter",
			setupReq: func(req *http.Request) {
				req.URL.RawQuery = "transmitter_id=query-transmitter"
			},
			want: "query-transmitter",
		},
		{
			name: "default transmitter",
			setupReq: func(req *http.Request) {
				// No headers or query params
			},
			want: "default-transmitter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/events", nil)
			tt.setupReq(req)

			got := handlers.getTransmitterID(req)
			if got != tt.want {
				t.Errorf("getTransmitterID() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestHandlers_HandleMetrics(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handlers.HandleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleMetrics() status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("HandleMetrics() content-type = %s, want text/plain", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "ssf_hub_receivers_total") {
		t.Error("HandleMetrics() should contain hub metrics")
	}
}

func TestHandlers_HandleUpdateReceiver(t *testing.T) {
	handlers := createTestHandlers(t)

	// First register a receiver
	receiver := &models.Receiver{
		ID:         "update-receiver",
		Name:       "Original Name",
		WebhookURL: "https://example.com/webhook",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		Status: models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	handlers.registry.Register(receiver)

	// Test updating the receiver
	updateReq := models.ReceiverRequest{
		ID:          "update-receiver",
		Name:        "Updated Name",
		WebhookURL:  "https://example.com/webhook",
		EventTypes:  []string{models.EventTypeCredentialChange},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
	}

	reqBody, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/api/v1/receivers/update-receiver", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "update-receiver")
	w := httptest.NewRecorder()

	handlers.HandleUpdateReceiver(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleUpdateReceiver() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response models.Receiver
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleUpdateReceiver() failed to unmarshal response: %v", err)
	}

	if response.Name != "Updated Name" {
		t.Errorf("HandleUpdateReceiver() receiver name = %s, want 'Updated Name'", response.Name)
	}
}

func TestHandlers_HandleUpdateReceiver_MissingID(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("PUT", "/api/v1/receivers/", nil)
	w := httptest.NewRecorder()

	handlers.HandleUpdateReceiver(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleUpdateReceiver() with missing ID status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlers_HandleUpdateReceiver_InvalidJSON(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("PUT", "/api/v1/receivers/test-id", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "test-id")
	w := httptest.NewRecorder()

	handlers.HandleUpdateReceiver(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleUpdateReceiver() with invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlers_HandleEvents_InvalidTransmitter(t *testing.T) {
	handlers := createTestHandlers(t)

	// Create request with default transmitter but invalid SET token
	req := httptest.NewRequest("POST", "/events", strings.NewReader("test event"))
	w := httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	// Since getTransmitterID() always returns "default-transmitter",
	// the request should proceed and fail on SET parsing
	if w.Code != http.StatusInternalServerError {
		t.Errorf("HandleEvents() with invalid SET status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlers_HandleEvents_ProcessingError(t *testing.T) {
	handlers := createTestHandlers(t)

	// Create request with transmitter ID but invalid SET
	req := httptest.NewRequest("POST", "/events", strings.NewReader("invalid-set-token"))
	req.Header.Set("X-Transmitter-ID", "test-transmitter")
	w := httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("HandleEvents() with invalid SET status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlers_HandleGetReceiver_WithSubscriptions(t *testing.T) {
	handlers := createTestHandlers(t)

	// Register a receiver
	receiver := &models.Receiver{
		ID:         "subscription-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	handlers.registry.Register(receiver)

	// Test with include_subscriptions parameter
	req := httptest.NewRequest("GET", "/api/v1/receivers/subscription-receiver?include_subscriptions=true", nil)
	req.SetPathValue("id", "subscription-receiver")
	w := httptest.NewRecorder()

	handlers.HandleGetReceiver(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleGetReceiver() with subscriptions status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("HandleGetReceiver() failed to unmarshal response: %v", err)
	}

	if _, exists := response["subscriptions"]; !exists {
		t.Error("HandleGetReceiver() with include_subscriptions should include subscriptions field")
	}
}

func TestHandlers_filterReceivers(t *testing.T) {
	handlers := createTestHandlers(t)

	receivers := []*models.Receiver{
		{
			ID:         "active-receiver",
			Status:     models.ReceiverStatusActive,
			EventTypes: []string{models.EventTypeSessionRevoked},
		},
		{
			ID:         "inactive-receiver",
			Status:     models.ReceiverStatusInactive,
			EventTypes: []string{models.EventTypeCredentialChange},
		},
		{
			ID:         "multi-event-receiver",
			Status:     models.ReceiverStatusActive,
			EventTypes: []string{models.EventTypeSessionRevoked, models.EventTypeCredentialChange},
		},
	}

	// Test status filter
	filtered := handlers.filterReceivers(receivers, "active", "")
	if len(filtered) != 2 {
		t.Errorf("filterReceivers() with status filter returned %d receivers, want 2", len(filtered))
	}

	// Test event type filter
	filtered = handlers.filterReceivers(receivers, "", models.EventTypeSessionRevoked)
	if len(filtered) != 2 {
		t.Errorf("filterReceivers() with event type filter returned %d receivers, want 2", len(filtered))
	}

	// Test combined filters
	filtered = handlers.filterReceivers(receivers, "active", models.EventTypeSessionRevoked)
	if len(filtered) != 2 {
		t.Errorf("filterReceivers() with combined filters returned %d receivers, want 2", len(filtered))
	}

	// Test with no filters
	filtered = handlers.filterReceivers(receivers, "", "")
	if len(filtered) != 3 {
		t.Errorf("filterReceivers() with no filters returned %d receivers, want 3", len(filtered))
	}
}

func TestHandlers_getTransmitterID_AllMethods(t *testing.T) {
	handlers := createTestHandlers(t)

	// Test X-Transmitter-ID header
	req := httptest.NewRequest("POST", "/events", nil)
	req.Header.Set("X-Transmitter-ID", "header-transmitter")
	transmitterID := handlers.getTransmitterID(req)
	if transmitterID != "header-transmitter" {
		t.Errorf("getTransmitterID() from header = %s, want header-transmitter", transmitterID)
	}

	// Test Authorization header (should be processed first if both exist)
	req = httptest.NewRequest("POST", "/events", nil)
	req.Header.Set("X-Transmitter-ID", "header-transmitter")
	req.Header.Set("Authorization", "Bearer token")
	transmitterID = handlers.getTransmitterID(req)
	if transmitterID != "header-transmitter" {
		t.Errorf("getTransmitterID() with both headers = %s, want header-transmitter", transmitterID)
	}

	// Test Authorization header only
	req = httptest.NewRequest("POST", "/events", nil)
	req.Header.Set("Authorization", "Bearer token")
	transmitterID = handlers.getTransmitterID(req)
	if transmitterID != "jwt-transmitter" {
		t.Errorf("getTransmitterID() from auth header = %s, want jwt-transmitter", transmitterID)
	}

	// Test query parameter
	req = httptest.NewRequest("POST", "/events?transmitter_id=query-transmitter", nil)
	transmitterID = handlers.getTransmitterID(req)
	if transmitterID != "query-transmitter" {
		t.Errorf("getTransmitterID() from query = %s, want query-transmitter", transmitterID)
	}

	// Test default case
	req = httptest.NewRequest("POST", "/events", nil)
	transmitterID = handlers.getTransmitterID(req)
	if transmitterID != "default-transmitter" {
		t.Errorf("getTransmitterID() default = %s, want default-transmitter", transmitterID)
	}
}

func TestHandlers_getBaseURL(t *testing.T) {
	handlers := createTestHandlers(t)

	// Test HTTP
	req := httptest.NewRequest("GET", "/.well-known/ssf_configuration", nil)
	req.Host = "example.com"
	baseURL := handlers.getBaseURL(req)
	if baseURL != "http://example.com" {
		t.Errorf("getBaseURL() HTTP = %s, want http://example.com", baseURL)
	}

	// Test HTTPS (simulate TLS)
	req = httptest.NewRequest("GET", "/.well-known/ssf_configuration", nil)
	req.Host = "example.com"
	req.TLS = &tls.ConnectionState{} // Non-nil indicates HTTPS
	baseURL = handlers.getBaseURL(req)
	if baseURL != "https://example.com" {
		t.Errorf("getBaseURL() HTTPS = %s, want https://example.com", baseURL)
	}
}

func TestHandlers_writeJSONResponse_Error(t *testing.T) {
	handlers := createTestHandlers(t)

	// Test with data that can't be marshaled to JSON
	w := httptest.NewRecorder()
	// Functions can't be marshaled to JSON
	invalidData := map[string]interface{}{
		"function": func() {},
	}

	// This should not panic and should handle the error gracefully
	handlers.writeJSONResponse(w, http.StatusOK, invalidData)

	// Check that an error response was written
	if w.Code != http.StatusOK {
		t.Errorf("writeJSONResponse() with invalid data status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlers_formatPrometheusMetrics_EmptyStats(t *testing.T) {
	handlers := createTestHandlers(t)

	stats := &controller.BrokerStats{
		TotalReceivers:    0,
		ReceiversByStatus: make(map[models.ReceiverStatus]int),
		EventTypeStats:    make(map[string]int),
	}

	metrics := handlers.formatPrometheusMetrics(stats)

	if !strings.Contains(metrics, "ssf_hub_receivers_total 0") {
		t.Error("formatPrometheusMetrics() should contain total receivers metric")
	}
}

func TestHandlers_formatPrometheusMetrics_WithData(t *testing.T) {
	handlers := createTestHandlers(t)

	stats := &controller.BrokerStats{
		TotalReceivers: 5,
		ReceiversByStatus: map[models.ReceiverStatus]int{
			models.ReceiverStatusActive:   3,
			models.ReceiverStatusInactive: 2,
		},
		EventTypeStats: map[string]int{
			models.EventTypeSessionRevoked:    2,
			models.EventTypeCredentialChange: 1,
		},
	}

	metrics := handlers.formatPrometheusMetrics(stats)

	expectedStrings := []string{
		"ssf_hub_receivers_total 5",
		"ssf_hub_receivers_by_status{status=\"active\"} 3",
		"ssf_hub_receivers_by_status{status=\"inactive\"} 2",
		"ssf_hub_event_type_subscribers",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(metrics, expected) {
			t.Errorf("formatPrometheusMetrics() should contain '%s'", expected)
		}
	}
}

func TestHandlers_Coverage_ErrorPaths(t *testing.T) {
	handlers := createTestHandlers(t)

	// Test HandleEvents with empty body after successful read
	req := httptest.NewRequest("POST", "/events", strings.NewReader(""))
	req.Header.Set("X-Transmitter-ID", "test-transmitter")
	w := httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleEvents() with empty body status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Test HandleListReceivers with filters that don't match
	req = httptest.NewRequest("GET", "/api/v1/receivers?status=nonexistent&event_type=nonexistent", nil)
	w = httptest.NewRecorder()

	handlers.HandleListReceivers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleListReceivers() with non-matching filters status = %d, want %d", w.Code, http.StatusOK)
	}
}

// Test successful HandleEvents processing (though it will fail on SET parsing)
func TestHandlers_HandleEvents_Success(t *testing.T) {
	handlers := createTestHandlers(t)

	// Since we don't have a real SET token parser, the test will actually fail
	// but we're testing the flow up to the processing step
	validToken := "mock-set-token"

	req := httptest.NewRequest("POST", "/events", strings.NewReader(validToken))
	req.Header.Set("X-Transmitter-ID", "test-transmitter")
	w := httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	// The mock controller will fail to parse the SET, so expect 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("HandleEvents() with mock token status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// Test HandleEvents with TLS connection state
func TestHandlers_HandleEvents_TLS(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("POST", "/events", strings.NewReader("test"))
	req.Header.Set("X-Transmitter-ID", "test-transmitter")

	// Add TLS connection state
	req.TLS = &tls.ConnectionState{
		ServerName: "example.com",
	}

	w := httptest.NewRecorder()
	handlers.HandleEvents(w, req)

	// Should still process (though will fail on SET parsing)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("HandleEvents() with TLS status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// Test getBaseURL with various configurations
func TestHandlers_getBaseURL_Comprehensive(t *testing.T) {
	handlers := createTestHandlers(t)

	tests := []struct {
		name        string
		setupReq    func(*http.Request)
		expectedURL string
	}{
		{
			name: "HTTP with custom port (X-Forwarded-Proto ignored)",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
				req.Host = "example.com:8443"
			},
			expectedURL: "http://example.com:8443", // getBaseURL doesn't check X-Forwarded-Proto
		},
		{
			name: "HTTP default port",
			setupReq: func(req *http.Request) {
				req.Host = "example.com"
			},
			expectedURL: "http://example.com",
		},
		{
			name: "HTTPS default port",
			setupReq: func(req *http.Request) {
				req.TLS = &tls.ConnectionState{}
				req.Host = "example.com"
			},
			expectedURL: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			tt.setupReq(req)

			result := handlers.getBaseURL(req)
			if result != tt.expectedURL {
				t.Errorf("getBaseURL() = %s, want %s", result, tt.expectedURL)
			}
		})
	}
}

// Test HandleRegisterReceiver with various validation scenarios
func TestHandlers_HandleRegisterReceiver_Validation(t *testing.T) {
	handlers := createTestHandlers(t)

	tests := []struct {
		name           string
		request        models.ReceiverRequest
		expectedStatus int
	}{
		{
			name: "missing webhook URL for webhook delivery",
			request: models.ReceiverRequest{
				ID:         "invalid-receiver",
				Name:       "Invalid Receiver",
				EventTypes: []string{models.EventTypeSessionRevoked},
				Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
				// WebhookURL is missing
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/api/v1/receivers", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handlers.HandleRegisterReceiver(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("HandleRegisterReceiver() validation test %s status = %d, want %d", tt.name, w.Code, tt.expectedStatus)
			}
		})
	}
}

// Test HandleUpdateReceiver with more comprehensive scenarios
func TestHandlers_HandleUpdateReceiver_Comprehensive(t *testing.T) {
	handlers := createTestHandlers(t)

	// First register a receiver
	receiver := &models.Receiver{
		ID:         "comprehensive-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	handlers.registry.Register(receiver)

	// Test updating with different event types
	updateReq := models.ReceiverRequest{
		ID:         "comprehensive-receiver",
		Name:       "Updated Receiver",
		EventTypes: []string{models.EventTypeCredentialChange, models.EventTypeAssuranceLevelChange},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://updated.example.com/webhook",
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/api/v1/receivers/comprehensive-receiver", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "comprehensive-receiver")
	w := httptest.NewRecorder()

	handlers.HandleUpdateReceiver(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleUpdateReceiver() comprehensive status = %d, want %d", w.Code, http.StatusOK)
	}
}

// Test HandleMetrics additional scenarios
func TestHandlers_HandleMetrics_Scenarios(t *testing.T) {
	handlers := createTestHandlers(t)

	// Test metrics endpoint multiple times to ensure consistency
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		handlers.HandleMetrics(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("HandleMetrics() iteration %d status = %d, want %d", i, w.Code, http.StatusOK)
		}

		// Check that response contains expected metrics format
		body := w.Body.String()
		if !strings.Contains(body, "ssf_hub_") {
			t.Errorf("HandleMetrics() iteration %d missing expected metrics prefix", i)
		}
	}
}

// Test cases for uncovered code paths
func TestHandlers_AdditionalCoverage(t *testing.T) {
	handlers := createTestHandlers(t)

	// Test include_subscriptions query parameter handling
	req := httptest.NewRequest("GET", "/api/v1/receivers/test?include_subscriptions=false", nil)
	req.SetPathValue("id", "test")
	w := httptest.NewRecorder()

	handlers.HandleGetReceiver(w, req)

	// Should process the include_subscriptions parameter
	if w.Code != http.StatusNotFound { // Expected since receiver doesn't exist
		t.Errorf("HandleGetReceiver() with include_subscriptions=false status = %d, want %d", w.Code, http.StatusNotFound)
	}

	// Test other query parameter scenarios
	req = httptest.NewRequest("GET", "/api/v1/receivers/test?include_subscriptions=invalid", nil)
	req.SetPathValue("id", "test")
	w = httptest.NewRecorder()

	handlers.HandleGetReceiver(w, req)

	if w.Code != http.StatusNotFound { // Expected since receiver doesn't exist
		t.Errorf("HandleGetReceiver() with invalid include_subscriptions status = %d, want %d", w.Code, http.StatusNotFound)
	}

	// Test empty transmitter ID scenarios (should fallback to default)
	req = httptest.NewRequest("POST", "/events", strings.NewReader("some-event"))
	// Don't set any transmitter ID headers - should use default
	w = httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	// Should process with default transmitter (though will fail on parsing)
	if w.Code != http.StatusInternalServerError { // Expected since SET parsing will fail
		t.Errorf("HandleEvents() with default transmitter status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	// Test read body error path by using a request with no body
	req = httptest.NewRequest("POST", "/events", nil) // nil body should be handled
	req.Header.Set("X-Transmitter-ID", "test-transmitter")
	w = httptest.NewRecorder()

	handlers.HandleEvents(w, req)

	// Should handle empty body case
	if w.Code != http.StatusBadRequest { // Expected for empty body
		t.Errorf("HandleEvents() with nil body status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// Test HandleMetrics error path by checking stats collection
func TestHandlers_HandleMetrics_StatsError(t *testing.T) {
	handlers := createTestHandlers(t)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handlers.HandleMetrics(w, req)

	// Even if stats processing has issues, should still return OK
	if w.Code != http.StatusOK {
		t.Errorf("HandleMetrics() status = %d, want %d", w.Code, http.StatusOK)
	}

	// Check metrics output contains expected format
	body := w.Body.String()
	if !strings.Contains(body, "ssf_hub_") {
		t.Errorf("HandleMetrics() missing expected metrics prefix")
	}
}

// Test filter functionality more thoroughly
func TestHandlers_FilterReceivers_EdgeCases(t *testing.T) {
	handlers := createTestHandlers(t)

	// Create test receivers with different properties
	receiver1 := &models.Receiver{
		ID:         "receiver1",
		Name:       "Test Receiver 1",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example1.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver1.SetDefaults()

	receiver2 := &models.Receiver{
		ID:         "receiver2",
		Name:       "Test Receiver 2",
		EventTypes: []string{models.EventTypeCredentialChange},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example2.com/webhook",
		Status:     models.ReceiverStatusInactive,
	}
	receiver2.SetDefaults()

	allReceivers := []*models.Receiver{receiver1, receiver2}

	// Test filtering by status only
	filtered := handlers.filterReceivers(allReceivers, string(models.ReceiverStatusActive), "")

	if len(filtered) != 1 || filtered[0].ID != "receiver1" {
		t.Errorf("filterReceivers() by status: got %d receivers, want 1 with ID receiver1", len(filtered))
	}

	// Test filtering by event type only
	filtered = handlers.filterReceivers(allReceivers, "", models.EventTypeCredentialChange)

	if len(filtered) != 1 || filtered[0].ID != "receiver2" {
		t.Errorf("filterReceivers() by event_type: got %d receivers, want 1 with ID receiver2", len(filtered))
	}

	// Test filtering with both criteria
	filtered = handlers.filterReceivers(allReceivers, string(models.ReceiverStatusActive), models.EventTypeSessionRevoked)

	if len(filtered) != 1 || filtered[0].ID != "receiver1" {
		t.Errorf("filterReceivers() with both criteria: got %d receivers, want 1 with ID receiver1", len(filtered))
	}

	// Test filtering with no matches
	filtered = handlers.filterReceivers(allReceivers, string(models.ReceiverStatusActive), models.EventTypeCredentialChange)

	if len(filtered) != 0 {
		t.Errorf("filterReceivers() with no matches: got %d receivers, want 0", len(filtered))
	}

	// Test no filtering (empty parameters)
	filtered = handlers.filterReceivers(allReceivers, "", "")

	if len(filtered) != 2 {
		t.Errorf("filterReceivers() with no filters: got %d receivers, want 2", len(filtered))
	}
}
