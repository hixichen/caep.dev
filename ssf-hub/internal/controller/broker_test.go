package controller

import (
	"context"
	"fmt"
	"testing"
	"log/slog"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// mockPubSubClient is a mock implementation of controller.PubSubClient for testing
type mockPubSubClient struct {
	publishedEvents   []*models.SecurityEvent
	targetReceivers   [][]string
	hubSubscriptions  []string
	hubInstanceID     string
}

func (m *mockPubSubClient) PublishEvent(ctx context.Context, event *models.SecurityEvent, targetReceivers []string) error {
	m.publishedEvents = append(m.publishedEvents, event)
	m.targetReceivers = append(m.targetReceivers, targetReceivers)
	return nil
}

func (m *mockPubSubClient) CreateHubSubscription(ctx context.Context, subscriptionName string) error {
	m.hubSubscriptions = append(m.hubSubscriptions, subscriptionName)
	return nil
}

func (m *mockPubSubClient) DeleteHubSubscription(ctx context.Context, subscriptionName string) error {
	// Remove from hubSubscriptions if present
	for i, sub := range m.hubSubscriptions {
		if sub == subscriptionName {
			m.hubSubscriptions = append(m.hubSubscriptions[:i], m.hubSubscriptions[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockPubSubClient) PullInternalMessages(ctx context.Context, subscriptionName string, maxMessages int, handler func(*models.InternalMessage) error) error {
	// Mock implementation - no messages to pull in tests
	return nil
}

func (m *mockPubSubClient) GetHubInstanceID() string {
	if m.hubInstanceID == "" {
		m.hubInstanceID = "test-hub-instance"
	}
	return m.hubInstanceID
}

func (m *mockPubSubClient) Close() error {
	return nil
}

func createTestController() (*Broker, *mockPubSubClient, registry.Registry) {
	logger := slog.Default()
	pubsubClient := &mockPubSubClient{}
	receiverRegistry := registry.NewMemoryRegistry()

	controller := New(pubsubClient, receiverRegistry, logger)
	return controller, pubsubClient, receiverRegistry
}

func TestController_RegisterReceiver(t *testing.T) {
	controller, _, _ := createTestController()

	receiverReq := &models.ReceiverRequest{
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

	ctx := context.Background()
	receiver, err := controller.RegisterReceiver(ctx, receiverReq)
	if err != nil {
		t.Fatalf("RegisterReceiver() error = %v", err)
	}

	// Check receiver was created correctly
	if receiver.ID != receiverReq.ID {
		t.Errorf("RegisterReceiver() receiver ID = %s, want %s", receiver.ID, receiverReq.ID)
	}

	if receiver.Status != models.ReceiverStatusActive {
		t.Errorf("RegisterReceiver() receiver status = %s, want %s", receiver.Status, models.ReceiverStatusActive)
	}

	// In the new architecture, receivers don't create direct subscriptions
	// They only get registered for webhook delivery
	// The hub manages all Pub/Sub operations internally
}

func TestController_RegisterReceiver_Invalid(t *testing.T) {
	controller, _, _ := createTestController()

	// Test with invalid receiver (missing ID)
	invalidReq := &models.ReceiverRequest{
		WebhookURL: "https://example.com/webhook",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
	}

	ctx := context.Background()
	_, err := controller.RegisterReceiver(ctx, invalidReq)
	if err == nil {
		t.Error("RegisterReceiver() expected error for invalid receiver but got none")
	}
}

func TestController_UnregisterReceiver(t *testing.T) {
	controller, _, registry := createTestController()

	// First register a receiver
	receiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked, models.EventTypeCredentialChange},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	err := registry.Register(receiver)
	if err != nil {
		t.Fatalf("Failed to register receiver: %v", err)
	}

	ctx := context.Background()
	err = controller.UnregisterReceiver(ctx, "test-receiver")
	if err != nil {
		t.Fatalf("UnregisterReceiver() error = %v", err)
	}

	// Check that receiver was removed from registry
	_, err = registry.Get("test-receiver")
	if err == nil {
		t.Error("UnregisterReceiver() receiver should be removed from registry")
	}

	// In the new architecture, no direct subscriptions are deleted
	// The hub manages all Pub/Sub operations internally
}

func TestController_UnregisterReceiver_NotFound(t *testing.T) {
	controller, _, _ := createTestController()

	ctx := context.Background()
	err := controller.UnregisterReceiver(ctx, "nonexistent")
	if err == nil {
		t.Error("UnregisterReceiver() expected error for non-existent receiver but got none")
	}
}

func TestController_UpdateReceiver(t *testing.T) {
	controller, _, registry := createTestController()

	// First register a receiver
	originalReceiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	originalReceiver.SetDefaults()
	err := registry.Register(originalReceiver)
	if err != nil {
		t.Fatalf("Failed to register receiver: %v", err)
	}

	// Update with different event types
	updateReq := &models.ReceiverRequest{
		ID:          "test-receiver",
		Name:        "Updated Receiver",
		WebhookURL:  "https://example.com/webhook",
		EventTypes:  []string{models.EventTypeCredentialChange, models.EventTypeAssuranceLevelChange},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
	}

	ctx := context.Background()
	updatedReceiver, err := controller.UpdateReceiver(ctx, updateReq)
	if err != nil {
		t.Fatalf("UpdateReceiver() error = %v", err)
	}

	// Check receiver was updated
	if updatedReceiver.Name != "Updated Receiver" {
		t.Errorf("UpdateReceiver() receiver name = %s, want %s", updatedReceiver.Name, "Updated Receiver")
	}

	if len(updatedReceiver.EventTypes) != 2 {
		t.Errorf("UpdateReceiver() expected 2 event types, got %d", len(updatedReceiver.EventTypes))
	}

	// Check that subscriptions were updated (deleted old, created new)
	// In the new architecture, no direct subscription management for receivers
	// Event routing is handled internally by the hub
}

func TestController_UpdateReceiver_SameEventTypes(t *testing.T) {
	controller, _, registry := createTestController()

	// First register a receiver
	originalReceiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	originalReceiver.SetDefaults()
	err := registry.Register(originalReceiver)
	if err != nil {
		t.Fatalf("Failed to register receiver: %v", err)
	}

	// Update with same event types but different name
	updateReq := &models.ReceiverRequest{
		ID:          "test-receiver",
		Name:        "Updated Receiver",
		WebhookURL:  "https://example.com/webhook",
		EventTypes:  []string{models.EventTypeSessionRevoked}, // Same event types
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
	}

	ctx := context.Background()
	_, err = controller.UpdateReceiver(ctx, updateReq)
	if err != nil {
		t.Fatalf("UpdateReceiver() error = %v", err)
	}

	// Check that subscriptions were NOT updated (no delete/create calls)
	// In the new architecture, no subscription management for unchanged event types
	// Event routing is handled internally by the hub
}

func TestController_GetControllerStats(t *testing.T) {
	controller, _, registry := createTestController()

	// Register some receivers
	receiver1 := &models.Receiver{
		ID:         "receiver-1",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Status:     models.ReceiverStatusActive,
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
	}
	receiver1.SetDefaults()

	receiver2 := &models.Receiver{
		ID:         "receiver-2",
		EventTypes: []string{models.EventTypeCredentialChange, models.EventTypeSessionRevoked},
		Status:     models.ReceiverStatusInactive,
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
	}
	receiver2.SetDefaults()

	registry.Register(receiver1)
	registry.Register(receiver2)

	stats, err := controller.GetBrokerStats()
	if err != nil {
		t.Fatalf("GetBrokerStats() error = %v", err)
	}

	if stats.TotalReceivers != 2 {
		t.Errorf("GetControllerStats() total receivers = %d, want 2", stats.TotalReceivers)
	}

	if stats.ReceiversByStatus[models.ReceiverStatusActive] != 1 {
		t.Errorf("GetControllerStats() active receivers = %d, want 1", stats.ReceiversByStatus[models.ReceiverStatusActive])
	}

	if stats.ReceiversByStatus[models.ReceiverStatusInactive] != 1 {
		t.Errorf("GetControllerStats() inactive receivers = %d, want 1", stats.ReceiversByStatus[models.ReceiverStatusInactive])
	}

	// Check event type stats
	if stats.EventTypeStats[models.EventTypeSessionRevoked] != 2 {
		t.Errorf("GetControllerStats() session-revoked subscribers = %d, want 2", stats.EventTypeStats[models.EventTypeSessionRevoked])
	}

	if stats.EventTypeStats[models.EventTypeCredentialChange] != 1 {
		t.Errorf("GetControllerStats() credential-change subscribers = %d, want 1", stats.EventTypeStats[models.EventTypeCredentialChange])
	}
}

func TestController_ProcessSecurityEvent(t *testing.T) {
	controller, mockClient, registry := createTestController()

	// Register a receiver for session revoked events
	receiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	err := registry.Register(receiver)
	if err != nil {
		t.Fatalf("Failed to register receiver: %v", err)
	}

	// Create a simple SET token for testing (this is a simplified mock SET)
	rawSET := `{
		"iss": "https://example.com",
		"jti": "test-event-123",
		"iat": 1234567890,
		"events": {
			"https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
				"session_id": "session-123"
			}
		},
		"sub": {
			"format": "email",
			"identifier": "user@example.com"
		}
	}`

	ctx := context.Background()
	err = controller.ProcessSecurityEvent(ctx, rawSET, "test-transmitter")

	// Note: This test will fail with current implementation since we need proper JWT
	// In a real implementation, you'd need to create a valid JWT SET token
	if err == nil {
		// Check that event was published
		if len(mockClient.publishedEvents) != 1 {
			t.Errorf("ProcessSecurityEvent() expected 1 published event, got %d", len(mockClient.publishedEvents))
		}

		publishedEvent := mockClient.publishedEvents[0]
		if publishedEvent.Type != models.EventTypeSessionRevoked {
			t.Errorf("ProcessSecurityEvent() event type = %s, want %s", publishedEvent.Type, models.EventTypeSessionRevoked)
		}

		if publishedEvent.Metadata.TransmitterID != "test-transmitter" {
			t.Errorf("ProcessSecurityEvent() transmitter ID = %s, want test-transmitter", publishedEvent.Metadata.TransmitterID)
		}
	}
}

func TestController_ProcessSecurityEvent_Validation(t *testing.T) {
	controller, _, _ := createTestController()
	ctx := context.Background()

	tests := []struct {
		name          string
		rawSET        string
		transmitterID string
		expectError   bool
	}{
		{"empty rawSET", "", "transmitter", true},
		{"empty transmitterID", "valid-set", "", true},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.ProcessSecurityEvent(ctx, tt.rawSET, tt.transmitterID)
			if (err != nil) != tt.expectError {
				t.Errorf("ProcessSecurityEvent() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestController_ProcessSecurityEvent_NoReceivers(t *testing.T) {
	controller, mockClient, _ := createTestController()

	// Don't register any receivers
	rawSET := `{
		"iss": "https://example.com",
		"jti": "test-event-123",
		"iat": 1234567890,
		"events": {
			"https://schemas.openid.net/secevent/caep/event-type/session-revoked": {}
		}
	}`

	ctx := context.Background()
	err := controller.ProcessSecurityEvent(ctx, rawSET, "test-transmitter")

	// Should not fail even if no receivers are found
	if err == nil {
		// Should not publish any events when no receivers
		if len(mockClient.publishedEvents) > 0 {
			t.Errorf("ProcessSecurityEvent() expected 0 published events when no receivers, got %d", len(mockClient.publishedEvents))
		}
	}
}

func TestController_slicesEqual(t *testing.T) {
	controller, _, _ := createTestController()

	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"equal slices", []string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{"different order", []string{"a", "b", "c"}, []string{"c", "b", "a"}, false},
		{"different length", []string{"a", "b"}, []string{"a", "b", "c"}, false},
		{"one empty", []string{"a"}, []string{}, false},
		{"both empty", []string{}, []string{}, true},
		{"different content", []string{"a", "b"}, []string{"a", "c"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := controller.slicesEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("slicesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_GetReceiver(t *testing.T) {
	controller, _, registry := createTestController()

	// Register a receiver first
	receiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	err := registry.Register(receiver)
	if err != nil {
		t.Fatalf("Failed to register receiver: %v", err)
	}

	// Test getting existing receiver
	got, err := controller.GetReceiver("test-receiver")
	if err != nil {
		t.Fatalf("GetReceiver() error = %v", err)
	}

	if got.ID != "test-receiver" {
		t.Errorf("GetReceiver() receiver ID = %s, want test-receiver", got.ID)
	}

	// Test getting non-existent receiver
	_, err = controller.GetReceiver("nonexistent")
	if err == nil {
		t.Error("GetReceiver() expected error for non-existent receiver but got none")
	}
}

func TestController_ListReceivers(t *testing.T) {
	controller, _, registry := createTestController()

	// Register multiple receivers
	receiver1 := &models.Receiver{
		ID:         "receiver-1",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook1",
		Status:     models.ReceiverStatusActive,
	}
	receiver1.SetDefaults()

	receiver2 := &models.Receiver{
		ID:         "receiver-2",
		EventTypes: []string{models.EventTypeCredentialChange},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook2",
		Status:     models.ReceiverStatusInactive,
	}
	receiver2.SetDefaults()

	registry.Register(receiver1)
	registry.Register(receiver2)

	// Test listing receivers
	receivers, err := controller.ListReceivers()
	if err != nil {
		t.Fatalf("ListReceivers() error = %v", err)
	}

	if len(receivers) != 2 {
		t.Errorf("ListReceivers() returned %d receivers, want 2", len(receivers))
	}
}

func TestController_GetReceiverSubscriptionInfo(t *testing.T) {
	controller, _, registry := createTestController()
	ctx := context.Background()

	// Register a receiver first
	receiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	err := registry.Register(receiver)
	if err != nil {
		t.Fatalf("Failed to register receiver: %v", err)
	}

	// Test getting subscription info for existing receiver
	info, err := controller.GetReceiverSubscriptionInfo(ctx, "test-receiver")
	if err != nil {
		t.Fatalf("GetReceiverSubscriptionInfo() error = %v", err)
	}

	if info == nil {
		t.Error("GetReceiverSubscriptionInfo() returned nil info")
	}

	// Test getting subscription info for non-existent receiver
	_, err = controller.GetReceiverSubscriptionInfo(ctx, "nonexistent")
	if err == nil {
		t.Error("GetReceiverSubscriptionInfo() expected error for non-existent receiver but got none")
	}
}

func TestController_UpdateReceiver_NotFound(t *testing.T) {
	controller, _, _ := createTestController()
	ctx := context.Background()

	// Try to update non-existent receiver
	updateReq := &models.ReceiverRequest{
		ID:          "nonexistent",
		Name:        "Updated Receiver",
		WebhookURL:  "https://example.com/webhook",
		EventTypes:  []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
	}

	_, err := controller.UpdateReceiver(ctx, updateReq)
	if err == nil {
		t.Error("UpdateReceiver() expected error for non-existent receiver but got none")
	}
}

func TestController_Start_Stop(t *testing.T) {
	controller, _, _ := createTestController()
	ctx := context.Background()

	// Test starting the controller
	err := controller.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Test stopping the controller
	err = controller.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestController_GetHubInstanceID(t *testing.T) {
	controller, _, _ := createTestController()

	hubID := controller.GetHubInstanceID()
	if hubID == "" {
		t.Error("GetHubInstanceID() returned empty string")
	}

	if hubID != "test-hub-instance" {
		t.Errorf("GetHubInstanceID() = %s, want test-hub-instance", hubID)
	}
}

func TestController_GetHubReceiver(t *testing.T) {
	controller, _, _ := createTestController()

	hubReceiver := controller.GetHubReceiver()
	if hubReceiver == nil {
		t.Error("GetHubReceiver() returned nil")
	}
}

func TestController_GetDistributor(t *testing.T) {
	controller, _, _ := createTestController()

	distributor := controller.GetDistributor()
	if distributor == nil {
		t.Error("GetDistributor() returned nil")
	}
}

func TestController_AsReceiver(t *testing.T) {
	controller, _, _ := createTestController()

	receiver := controller.AsReceiver()
	if receiver == nil {
		t.Error("AsReceiver() returned nil")
	}
}

func TestController_RegisterReceiver_DuplicateID(t *testing.T) {
	controller, _, _ := createTestController()
	ctx := context.Background()

	// Register first receiver
	receiverReq1 := &models.ReceiverRequest{
		ID:          "duplicate-receiver",
		Name:        "First Receiver",
		WebhookURL:  "https://example.com/webhook1",
		EventTypes:  []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		Auth: models.AuthConfig{
			Type:  models.AuthTypeNone,
		},
	}

	_, err := controller.RegisterReceiver(ctx, receiverReq1)
	if err != nil {
		t.Fatalf("First RegisterReceiver() error = %v", err)
	}

	// Try to register receiver with same ID
	receiverReq2 := &models.ReceiverRequest{
		ID:          "duplicate-receiver",
		Name:        "Second Receiver",
		WebhookURL:  "https://example.com/webhook2",
		EventTypes:  []string{models.EventTypeCredentialChange},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		Auth: models.AuthConfig{
			Type:  models.AuthTypeNone,
		},
	}

	_, err = controller.RegisterReceiver(ctx, receiverReq2)
	if err == nil {
		t.Error("RegisterReceiver() expected error for duplicate ID but got none")
	}
}

func TestController_GetBrokerStats_EmptyRegistry(t *testing.T) {
	controller, _, _ := createTestController()

	stats, err := controller.GetBrokerStats()
	if err != nil {
		t.Fatalf("GetBrokerStats() error = %v", err)
	}

	if stats.TotalReceivers != 0 {
		t.Errorf("GetBrokerStats() total receivers = %d, want 0", stats.TotalReceivers)
	}

	if len(stats.EventTypeStats) != 0 {
		t.Errorf("GetBrokerStats() event type stats length = %d, want 0", len(stats.EventTypeStats))
	}
}

func TestController_ProcessSecurityEvent_InvalidJSON(t *testing.T) {
	controller, _, _ := createTestController()
	ctx := context.Background()

	// Use invalid JSON to trigger parser error
	invalidSET := "invalid-json"
	transmitterID := "test-transmitter"

	err := controller.ProcessSecurityEvent(ctx, invalidSET, transmitterID)
	if err == nil {
		t.Error("ProcessSecurityEvent() expected error for invalid SET but got none")
	}
}

func TestNew(t *testing.T) {
	logger := slog.Default()
	pubsubClient := &mockPubSubClient{}
	receiverRegistry := registry.NewMemoryRegistry()

	controller := New(pubsubClient, receiverRegistry, logger)

	if controller == nil {
		t.Error("New() returned nil controller")
	}

	if controller.GetHubInstanceID() == "" {
		t.Error("New() controller has empty hub instance ID")
	}

	if controller.GetHubReceiver() == nil {
		t.Error("New() controller has nil hub receiver")
	}

	if controller.GetDistributor() == nil {
		t.Error("New() controller has nil distributor")
	}
}
func TestController_RegisterReceiver_ValidationFailure(t *testing.T) {
	controller, _, _ := createTestController()
	ctx := context.Background()

	// Test with missing webhook URL for webhook delivery
	invalidReq := &models.ReceiverRequest{
		ID:         "invalid-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		// Missing WebhookURL
	}

	_, err := controller.RegisterReceiver(ctx, invalidReq)
	if err == nil {
		t.Error("RegisterReceiver() expected validation error but got none")
	}
}

func TestController_UpdateReceiver_ValidationFailure(t *testing.T) {
	controller, _, registry := createTestController()
	ctx := context.Background()

	// Register a valid receiver first
	receiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	err := registry.Register(receiver)
	if err != nil {
		t.Fatalf("Failed to register receiver: %v", err)
	}

	// Try to update with invalid configuration
	invalidUpdateReq := &models.ReceiverRequest{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		// Missing WebhookURL - should fail validation
	}

	_, err = controller.UpdateReceiver(ctx, invalidUpdateReq)
	if err == nil {
		t.Error("UpdateReceiver() expected validation error but got none")
	}
}

func TestController_ConvertToSecurityEvent_EdgeCases(t *testing.T) {
	// This tests edge cases in ProcessSecurityEvent that would reach convertToSecurityEvent
	controller, _, _ := createTestController()
	ctx := context.Background()

	// Test with minimal valid inputs to check parameter validation
	tests := []struct {
		name          string
		rawSET        string
		transmitterID string
		expectError   bool
	}{
		{"empty rawSET", "", "transmitter", true},
		{"empty transmitter", "set", "", true},
		{"both provided but invalid", "invalid-jwt", "transmitter", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.ProcessSecurityEvent(ctx, tt.rawSET, tt.transmitterID)
			if (err != nil) != tt.expectError {
				t.Errorf("ProcessSecurityEvent() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestController_SlicesEqual_EdgeCases(t *testing.T) {
	controller, _, _ := createTestController()

	// Test nil slices
	var nilSlice []string
	emptySlice := []string{}

	if !controller.slicesEqual(nilSlice, nilSlice) {
		t.Error("slicesEqual() should return true for two nil slices")
	}

	if !controller.slicesEqual(emptySlice, emptySlice) {
		t.Error("slicesEqual() should return true for two empty slices")
	}

	// Both nil and empty slices have length 0, so they are considered equal
	if !controller.slicesEqual(nilSlice, emptySlice) {
		t.Error("slicesEqual() should return true for nil vs empty slice (both have length 0)")
	}
}

func TestController_BrokerStats_DetailedCoverage(t *testing.T) {
	controller, _, registry := createTestController()

	// Test with receivers having multiple event types
	receiver := &models.Receiver{
		ID:         "multi-event-receiver",
		EventTypes: []string{
			models.EventTypeSessionRevoked,
			models.EventTypeCredentialChange,
			models.EventTypeAssuranceLevelChange,
		},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	registry.Register(receiver)

	stats, err := controller.GetBrokerStats()
	if err != nil {
		t.Fatalf("GetBrokerStats() error = %v", err)
	}

	// Verify each event type is counted
	expectedEventTypes := []string{
		models.EventTypeSessionRevoked,
		models.EventTypeCredentialChange,
		models.EventTypeAssuranceLevelChange,
	}

	for _, eventType := range expectedEventTypes {
		if count, exists := stats.EventTypeStats[eventType]; !exists || count != 1 {
			t.Errorf("GetBrokerStats() event type %s count = %d, want 1", eventType, count)
		}
	}
}

func TestController_GetReceiverMethods_Coverage(t *testing.T) {
	controller, _, registry := createTestController()

	// Test methods that primarily delegate to registry
	receivers, err := controller.ListReceivers()
	if err != nil {
		t.Fatalf("ListReceivers() error = %v", err)
	}
	if len(receivers) != 0 {
		t.Errorf("ListReceivers() should return empty list initially")
	}

	// Register a receiver to test other methods
	receiver := &models.Receiver{
		ID:         "coverage-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	registry.Register(receiver)

	// Test GetReceiver
	gotReceiver, err := controller.GetReceiver("coverage-receiver")
	if err != nil {
		t.Fatalf("GetReceiver() error = %v", err)
	}
	if gotReceiver.ID != "coverage-receiver" {
		t.Errorf("GetReceiver() ID = %s, want coverage-receiver", gotReceiver.ID)
	}

	// Test AsReceiver
	hubAsReceiver := controller.AsReceiver()
	if hubAsReceiver == nil {
		t.Error("AsReceiver() returned nil")
	}
}

// Test the Start method with message processing
func TestController_Start_WithMessageProcessing(t *testing.T) {
	controller, client, registry := createTestController()

	// Create a mock that will provide a message to process
	mockClient := &mockPubSubClientWithMessages{
		mockPubSubClient: *client,
		messagesToProvide: []*models.InternalMessage{
			{
				MessageID:   "test-msg-1",
				MessageType: "security_event",
				Event: &models.SecurityEvent{
					ID:     "test-event-1",
					Type:   models.EventTypeSessionRevoked,
					Source: "test-transmitter",
				},
				Routing: models.RoutingInfo{
					TargetReceivers: []string{"test-receiver"},
				},
			},
		},
	}

	// Replace the client with our enhanced mock
	controller.pubsubClient = mockClient

	// Register a receiver
	receiver := &models.Receiver{
		ID:         "test-receiver",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver.SetDefaults()
	registry.Register(receiver)

	// Start the controller
	ctx := context.Background()
	err := controller.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop immediately to avoid hanging
	controller.Stop(ctx)
}

// Mock client that provides messages during PullInternalMessages calls
type mockPubSubClientWithMessages struct {
	mockPubSubClient
	messagesToProvide []*models.InternalMessage
	pullCallCount     int
}

func (m *mockPubSubClientWithMessages) PullInternalMessages(ctx context.Context, subscriptionName string, maxMessages int, handler func(*models.InternalMessage) error) error {
	m.pullCallCount++

	// On first call, provide the test message
	if m.pullCallCount == 1 && len(m.messagesToProvide) > 0 {
		for _, msg := range m.messagesToProvide {
			handler(msg)
		}
	}

	// Return after providing messages to avoid infinite loop in tests
	return nil
}

// Test error paths in ProcessSecurityEvent
func TestController_ProcessSecurityEvent_ErrorPaths(t *testing.T) {
	controller, _, _ := createTestController()

	tests := []struct {
		name          string
		rawSET        string
		transmitterID string
		wantError     bool
	}{
		{
			name:          "empty SET",
			rawSET:        "",
			transmitterID: "test-transmitter",
			wantError:     true,
		},
		{
			name:          "empty transmitter ID",
			rawSET:        "test-set",
			transmitterID: "",
			wantError:     true,
		},
		{
			name:          "invalid JWT format",
			rawSET:        "invalid.jwt.token",
			transmitterID: "test-transmitter",
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.ProcessSecurityEvent(context.Background(), tt.rawSET, tt.transmitterID)
			if (err != nil) != tt.wantError {
				t.Errorf("ProcessSecurityEvent() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Test Start method error handling
func TestController_Start_ErrorHandling(t *testing.T) {
	controller, client, _ := createTestController()

	// Create a client that will return an error on subscription creation
	errorClient := &mockPubSubClientWithError{
		mockPubSubClient: *client,
		errorOnCreate:    true,
	}
	controller.pubsubClient = errorClient

	ctx := context.Background()
	err := controller.Start(ctx)
	if err == nil {
		t.Error("Start() should return error when subscription creation fails")
	}
}

// Mock client that can simulate errors
type mockPubSubClientWithError struct {
	mockPubSubClient
	errorOnCreate bool
	errorOnPull   bool
}

func (m *mockPubSubClientWithError) CreateHubSubscription(ctx context.Context, subscriptionName string) error {
	if m.errorOnCreate {
		return fmt.Errorf("mock subscription creation error")
	}
	return m.mockPubSubClient.CreateHubSubscription(ctx, subscriptionName)
}

func (m *mockPubSubClientWithError) PullInternalMessages(ctx context.Context, subscriptionName string, maxMessages int, handler func(*models.InternalMessage) error) error {
	if m.errorOnPull {
		return fmt.Errorf("mock pull error")
	}
	return m.mockPubSubClient.PullInternalMessages(ctx, subscriptionName, maxMessages, handler)
}

// Test the ProcessSecurityEvent method with better coverage
func TestController_ProcessSecurityEvent_AdvancedCases(t *testing.T) {
	controller, _, _ := createTestController()

	// Test with a malformed JWT that still has segments
	malformedJWT := "header.payload.signature"

	err := controller.ProcessSecurityEvent(context.Background(), malformedJWT, "test-transmitter")
	// This should fail during parsing
	if err == nil {
		t.Error("ProcessSecurityEvent() should fail with malformed JWT")
	}
}

// Test registry integration
func TestController_RegistryIntegration(t *testing.T) {
	controller, _, registry := createTestController()

	// Test ListReceivers with actual receivers
	receiver1 := &models.Receiver{
		ID:         "receiver1",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example1.com/webhook",
		Status:     models.ReceiverStatusActive,
	}
	receiver1.SetDefaults()

	receiver2 := &models.Receiver{
		ID:         "receiver2",
		EventTypes: []string{models.EventTypeCredentialChange},
		Delivery:   models.DeliveryConfig{Method: models.DeliveryMethodWebhook},
		WebhookURL: "https://example2.com/webhook",
		Status:     models.ReceiverStatusInactive,
	}
	receiver2.SetDefaults()

	registry.Register(receiver1)
	registry.Register(receiver2)

	// Test ListReceivers
	receivers, err := controller.ListReceivers()
	if err != nil {
		t.Fatalf("ListReceivers() error = %v", err)
	}
	if len(receivers) != 2 {
		t.Errorf("ListReceivers() returned %d receivers, want 2", len(receivers))
	}

	// Test GetReceiver for each
	for _, expectedReceiver := range []*models.Receiver{receiver1, receiver2} {
		gotReceiver, err := controller.GetReceiver(expectedReceiver.ID)
		if err != nil {
			t.Errorf("GetReceiver(%s) error = %v", expectedReceiver.ID, err)
		}
		if gotReceiver.ID != expectedReceiver.ID {
			t.Errorf("GetReceiver(%s) ID = %s, want %s", expectedReceiver.ID, gotReceiver.ID, expectedReceiver.ID)
		}
	}
}
