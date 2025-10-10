package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"log/slog"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/broker"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// mockPubSubClient implements broker.PubSubClient for testing
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
	ssfBroker := broker.New(pubsubClient, receiverRegistry, logger)

	config := &Config{
		Logger:   logger,
		Broker:   ssfBroker,
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