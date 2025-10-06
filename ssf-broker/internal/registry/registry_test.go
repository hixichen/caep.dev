package registry

import (
	"testing"
	"time"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-broker/pkg/models"
)

func createTestReceiver(id string) *models.Receiver {
	return &models.Receiver{
		ID:         id,
		Name:       "Test Receiver " + id,
		WebhookURL: "https://example.com/webhook",
		EventTypes: []string{models.EventTypeSessionRevoked},
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook,
		},
		Auth: models.AuthConfig{
			Type:  models.AuthTypeBearer,
			Token: "test-token",
		},
		Status: models.ReceiverStatusActive,
		Metadata: models.ReceiverMetadata{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      make(map[string]string),
		},
	}
}

func TestMemoryRegistry_Register(t *testing.T) {
	registry := NewMemoryRegistry()

	receiver := createTestReceiver("test-1")

	// Test successful registration
	err := registry.Register(receiver)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test duplicate registration
	err = registry.Register(receiver)
	if err == nil {
		t.Error("Register() expected error for duplicate ID but got none")
	}

	// Test nil receiver
	err = registry.Register(nil)
	if err == nil {
		t.Error("Register() expected error for nil receiver but got none")
	}

	// Test invalid receiver
	invalidReceiver := &models.Receiver{
		ID: "", // Invalid: empty ID
	}
	err = registry.Register(invalidReceiver)
	if err == nil {
		t.Error("Register() expected error for invalid receiver but got none")
	}
}

func TestMemoryRegistry_Get(t *testing.T) {
	registry := NewMemoryRegistry()
	receiver := createTestReceiver("test-1")

	// Test get non-existent receiver
	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Get() expected error for non-existent receiver but got none")
	}

	// Register receiver
	err = registry.Register(receiver)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test get existing receiver
	retrieved, err := registry.Get("test-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.ID != receiver.ID {
		t.Errorf("Get() ID mismatch: got %v, want %v", retrieved.ID, receiver.ID)
	}

	if retrieved.Name != receiver.Name {
		t.Errorf("Get() Name mismatch: got %v, want %v", retrieved.Name, receiver.Name)
	}

	// Verify it returns a copy (modifications don't affect original)
	retrieved.Name = "Modified Name"
	original, _ := registry.Get("test-1")
	if original.Name == "Modified Name" {
		t.Error("Get() should return a copy, but modifications affected the original")
	}
}

func TestMemoryRegistry_List(t *testing.T) {
	registry := NewMemoryRegistry()

	// Test empty registry
	receivers, err := registry.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(receivers) != 0 {
		t.Errorf("List() expected 0 receivers, got %d", len(receivers))
	}

	// Add receivers
	receiver1 := createTestReceiver("test-1")
	receiver2 := createTestReceiver("test-2")

	registry.Register(receiver1)
	registry.Register(receiver2)

	// Test list with receivers
	receivers, err = registry.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(receivers) != 2 {
		t.Errorf("List() expected 2 receivers, got %d", len(receivers))
	}

	// Verify receiver IDs
	ids := make(map[string]bool)
	for _, r := range receivers {
		ids[r.ID] = true
	}

	if !ids["test-1"] || !ids["test-2"] {
		t.Error("List() missing expected receiver IDs")
	}
}

func TestMemoryRegistry_Update(t *testing.T) {
	registry := NewMemoryRegistry()
	receiver := createTestReceiver("test-1")

	// Test update non-existent receiver
	err := registry.Update(receiver)
	if err == nil {
		t.Error("Update() expected error for non-existent receiver but got none")
	}

	// Register receiver
	err = registry.Register(receiver)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test successful update
	receiver.Name = "Updated Name"
	receiver.EventTypes = []string{models.EventTypeCredentialChange}

	err = registry.Update(receiver)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	updated, err := registry.Get("test-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Update() name not updated: got %v, want %v", updated.Name, "Updated Name")
	}

	if len(updated.EventTypes) != 1 || updated.EventTypes[0] != models.EventTypeCredentialChange {
		t.Errorf("Update() event types not updated: got %v", updated.EventTypes)
	}

	// Test nil receiver update
	err = registry.Update(nil)
	if err == nil {
		t.Error("Update() expected error for nil receiver but got none")
	}
}

func TestMemoryRegistry_Unregister(t *testing.T) {
	registry := NewMemoryRegistry()
	receiver := createTestReceiver("test-1")

	// Test unregister non-existent receiver
	err := registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Unregister() expected error for non-existent receiver but got none")
	}

	// Register receiver
	err = registry.Register(receiver)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test successful unregister
	err = registry.Unregister("test-1")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	// Verify receiver is gone
	_, err = registry.Get("test-1")
	if err == nil {
		t.Error("Get() expected error for unregistered receiver but got none")
	}
}

func TestMemoryRegistry_GetByEventType(t *testing.T) {
	registry := NewMemoryRegistry()

	// Create receivers with different event types
	receiver1 := createTestReceiver("test-1")
	receiver1.EventTypes = []string{models.EventTypeSessionRevoked}

	receiver2 := createTestReceiver("test-2")
	receiver2.EventTypes = []string{models.EventTypeCredentialChange}

	receiver3 := createTestReceiver("test-3")
	receiver3.EventTypes = []string{models.EventTypeSessionRevoked, models.EventTypeCredentialChange}

	receiver4 := createTestReceiver("test-4")
	receiver4.EventTypes = []string{models.EventTypeSessionRevoked}
	receiver4.Status = models.ReceiverStatusInactive // Should be excluded

	registry.Register(receiver1)
	registry.Register(receiver2)
	registry.Register(receiver3)
	registry.Register(receiver4)

	// Test get by session revoked event type
	receivers, err := registry.GetByEventType(models.EventTypeSessionRevoked)
	if err != nil {
		t.Fatalf("GetByEventType() error = %v", err)
	}

	// Should return receiver1 and receiver3 (receiver4 is inactive)
	if len(receivers) != 2 {
		t.Errorf("GetByEventType() expected 2 receivers, got %d", len(receivers))
	}

	ids := make(map[string]bool)
	for _, r := range receivers {
		ids[r.ID] = true
	}

	if !ids["test-1"] || !ids["test-3"] {
		t.Error("GetByEventType() missing expected receiver IDs")
	}

	if ids["test-4"] {
		t.Error("GetByEventType() should not include inactive receivers")
	}

	// Test get by credential change event type
	receivers, err = registry.GetByEventType(models.EventTypeCredentialChange)
	if err != nil {
		t.Fatalf("GetByEventType() error = %v", err)
	}

	// Should return receiver2 and receiver3
	if len(receivers) != 2 {
		t.Errorf("GetByEventType() expected 2 receivers, got %d", len(receivers))
	}

	// Test get by non-existent event type
	receivers, err = registry.GetByEventType("https://example.com/nonexistent")
	if err != nil {
		t.Fatalf("GetByEventType() error = %v", err)
	}

	if len(receivers) != 0 {
		t.Errorf("GetByEventType() expected 0 receivers for non-existent event type, got %d", len(receivers))
	}
}

func TestMemoryRegistry_GetActiveReceivers(t *testing.T) {
	registry := NewMemoryRegistry()

	// Create receivers with different statuses
	receiver1 := createTestReceiver("test-1")
	receiver1.Status = models.ReceiverStatusActive

	receiver2 := createTestReceiver("test-2")
	receiver2.Status = models.ReceiverStatusInactive

	receiver3 := createTestReceiver("test-3")
	receiver3.Status = models.ReceiverStatusActive

	receiver4 := createTestReceiver("test-4")
	receiver4.Status = models.ReceiverStatusError

	registry.Register(receiver1)
	registry.Register(receiver2)
	registry.Register(receiver3)
	registry.Register(receiver4)

	// Test get active receivers
	activeReceivers, err := registry.GetActiveReceivers()
	if err != nil {
		t.Fatalf("GetActiveReceivers() error = %v", err)
	}

	// Should return only receiver1 and receiver3
	if len(activeReceivers) != 2 {
		t.Errorf("GetActiveReceivers() expected 2 active receivers, got %d", len(activeReceivers))
	}

	ids := make(map[string]bool)
	for _, r := range activeReceivers {
		ids[r.ID] = true
		if r.Status != models.ReceiverStatusActive {
			t.Errorf("GetActiveReceivers() returned non-active receiver: %s", r.ID)
		}
	}

	if !ids["test-1"] || !ids["test-3"] {
		t.Error("GetActiveReceivers() missing expected active receiver IDs")
	}
}

func TestMemoryRegistry_UpdateStats(t *testing.T) {
	registry := NewMemoryRegistry()
	receiver := createTestReceiver("test-1")

	// Test update stats for non-existent receiver
	err := registry.UpdateStats("nonexistent", models.ReceiverMetadata{})
	if err == nil {
		t.Error("UpdateStats() expected error for non-existent receiver but got none")
	}

	// Register receiver
	err = registry.Register(receiver)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test successful stats update
	newStats := models.ReceiverMetadata{
		EventsReceived:  100,
		EventsDelivered: 95,
		EventsFailed:    5,
		LastEventAt:     time.Now(),
	}

	err = registry.UpdateStats("test-1", newStats)
	if err != nil {
		t.Fatalf("UpdateStats() error = %v", err)
	}

	// Verify stats were updated
	updated, err := registry.Get("test-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if updated.Metadata.EventsReceived != 100 {
		t.Errorf("UpdateStats() events received not updated: got %d, want %d", updated.Metadata.EventsReceived, 100)
	}

	if updated.Metadata.EventsDelivered != 95 {
		t.Errorf("UpdateStats() events delivered not updated: got %d, want %d", updated.Metadata.EventsDelivered, 95)
	}

	if updated.Metadata.EventsFailed != 5 {
		t.Errorf("UpdateStats() events failed not updated: got %d, want %d", updated.Metadata.EventsFailed, 5)
	}
}

func TestMemoryRegistry_Count(t *testing.T) {
	registry := NewMemoryRegistry()

	// Test empty registry
	if count := registry.Count(); count != 0 {
		t.Errorf("Count() expected 0, got %d", count)
	}

	// Add receivers
	registry.Register(createTestReceiver("test-1"))
	registry.Register(createTestReceiver("test-2"))

	if count := registry.Count(); count != 2 {
		t.Errorf("Count() expected 2, got %d", count)
	}

	// Remove receiver
	registry.Unregister("test-1")

	if count := registry.Count(); count != 1 {
		t.Errorf("Count() expected 1, got %d", count)
	}
}

func TestMemoryRegistry_CountByStatus(t *testing.T) {
	registry := NewMemoryRegistry()

	// Add receivers with different statuses
	receiver1 := createTestReceiver("test-1")
	receiver1.Status = models.ReceiverStatusActive

	receiver2 := createTestReceiver("test-2")
	receiver2.Status = models.ReceiverStatusActive

	receiver3 := createTestReceiver("test-3")
	receiver3.Status = models.ReceiverStatusInactive

	receiver4 := createTestReceiver("test-4")
	receiver4.Status = models.ReceiverStatusError

	registry.Register(receiver1)
	registry.Register(receiver2)
	registry.Register(receiver3)
	registry.Register(receiver4)

	counts := registry.CountByStatus()

	expectedCounts := map[models.ReceiverStatus]int{
		models.ReceiverStatusActive:   2,
		models.ReceiverStatusInactive: 1,
		models.ReceiverStatusError:    1,
	}

	for status, expectedCount := range expectedCounts {
		if actualCount := counts[status]; actualCount != expectedCount {
			t.Errorf("CountByStatus() status %v: got %d, want %d", status, actualCount, expectedCount)
		}
	}
}

func TestMemoryRegistry_FilterReceivers(t *testing.T) {
	registry := NewMemoryRegistry()

	// Create test event
	event := &models.SecurityEvent{
		Type:   models.EventTypeSessionRevoked,
		Source: "test-issuer",
		Subject: models.Subject{
			Format:     models.SubjectFormatEmail,
			Identifier: "user@example.com",
		},
		Metadata: models.EventMetadata{
			TransmitterID: "test-transmitter",
		},
	}

	// Create receivers with different filters
	receiver1 := createTestReceiver("test-1")
	receiver1.EventTypes = []string{models.EventTypeSessionRevoked}
	receiver1.Filters = []models.EventFilter{} // No filters

	receiver2 := createTestReceiver("test-2")
	receiver2.EventTypes = []string{models.EventTypeSessionRevoked}
	receiver2.Filters = []models.EventFilter{
		{
			Field:    "source",
			Operator: models.FilterOpEquals,
			Value:    "test-issuer",
		},
	}

	receiver3 := createTestReceiver("test-3")
	receiver3.EventTypes = []string{models.EventTypeSessionRevoked}
	receiver3.Filters = []models.EventFilter{
		{
			Field:    "source",
			Operator: models.FilterOpEquals,
			Value:    "different-issuer", // Won't match
		},
	}

	receiver4 := createTestReceiver("test-4")
	receiver4.EventTypes = []string{models.EventTypeCredentialChange} // Different event type

	registry.Register(receiver1)
	registry.Register(receiver2)
	registry.Register(receiver3)
	registry.Register(receiver4)

	// Test filtering
	filteredReceivers, err := registry.FilterReceivers(event)
	if err != nil {
		t.Fatalf("FilterReceivers() error = %v", err)
	}

	// Should return receiver1 and receiver2 only
	if len(filteredReceivers) != 2 {
		t.Errorf("FilterReceivers() expected 2 receivers, got %d", len(filteredReceivers))
	}

	ids := make(map[string]bool)
	for _, r := range filteredReceivers {
		ids[r.ID] = true
	}

	if !ids["test-1"] || !ids["test-2"] {
		t.Error("FilterReceivers() missing expected receiver IDs")
	}

	if ids["test-3"] || ids["test-4"] {
		t.Error("FilterReceivers() should not include filtered out receivers")
	}
}