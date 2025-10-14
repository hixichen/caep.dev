package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSecurityEvent_ToJSON(t *testing.T) {
	event := &SecurityEvent{
		ID:          "test-event-123",
		Type:        EventTypeSessionRevoked,
		Source:      "test-issuer",
		SpecVersion: "1.0",
		Time:        time.Date(2023, 10, 5, 12, 0, 0, 0, time.UTC),
		Subject: Subject{
			Format:     SubjectFormatEmail,
			Identifier: "user@example.com",
			Claims:     map[string]interface{}{"sub": "user123"},
		},
		Data: map[string]interface{}{
			"session_id": "session-123",
			"reason":     "admin_action",
		},
		Extensions: map[string]interface{}{
			"custom_field": "custom_value",
		},
		Metadata: EventMetadata{
			ReceivedAt:    time.Date(2023, 10, 5, 12, 0, 0, 0, time.UTC),
			ProcessedAt:   time.Date(2023, 10, 5, 12, 0, 1, 0, time.UTC),
			TransmitterID: "test-transmitter",
			ProcessingID:  "proc-123",
			Tags:          map[string]string{"env": "test"},
		},
	}

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify the JSON can be unmarshaled back
	var unmarshaled SecurityEvent
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check key fields
	if unmarshaled.ID != event.ID {
		t.Errorf("ID mismatch: got %v, want %v", unmarshaled.ID, event.ID)
	}

	if unmarshaled.Type != event.Type {
		t.Errorf("Type mismatch: got %v, want %v", unmarshaled.Type, event.Type)
	}

	if unmarshaled.Subject.Identifier != event.Subject.Identifier {
		t.Errorf("Subject identifier mismatch: got %v, want %v", unmarshaled.Subject.Identifier, event.Subject.Identifier)
	}
}

func TestSecurityEvent_ToPubSubMessage(t *testing.T) {
	event := &SecurityEvent{
		ID:          "test-event-123",
		Type:        EventTypeSessionRevoked,
		Source:      "test-issuer",
		SpecVersion: "1.0",
		Time:        time.Date(2023, 10, 5, 12, 0, 0, 0, time.UTC),
		Subject: Subject{
			Format:     SubjectFormatEmail,
			Identifier: "user@example.com",
		},
		Metadata: EventMetadata{
			TransmitterID: "test-transmitter",
			ProcessingID:  "proc-123",
		},
	}

	data, attributes, err := event.ToPubSubMessage()
	if err != nil {
		t.Fatalf("ToPubSubMessage() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty message data")
	}

	// Check required attributes
	expectedAttrs := map[string]string{
		"event_id":       event.ID,
		"event_type":     event.Type,
		"source":         event.Source,
		"subject_format": event.Subject.Format,
		"transmitter_id": event.Metadata.TransmitterID,
		"processing_id":  event.Metadata.ProcessingID,
	}

	for key, expectedVal := range expectedAttrs {
		if actualVal, exists := attributes[key]; !exists {
			t.Errorf("Missing attribute %s", key)
		} else if actualVal != expectedVal {
			t.Errorf("Attribute %s mismatch: got %v, want %v", key, actualVal, expectedVal)
		}
	}

	// Verify the data can be unmarshaled back to the event
	var unmarshaled SecurityEvent
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Pub/Sub message data: %v", err)
	}

	if unmarshaled.ID != event.ID {
		t.Errorf("Unmarshaled event ID mismatch: got %v, want %v", unmarshaled.ID, event.ID)
	}
}

func TestFromJSON(t *testing.T) {
	originalEvent := &SecurityEvent{
		ID:          "test-event-123",
		Type:        EventTypeSessionRevoked,
		Source:      "test-issuer",
		SpecVersion: "1.0",
		Time:        time.Date(2023, 10, 5, 12, 0, 0, 0, time.UTC),
		Subject: Subject{
			Format:     SubjectFormatEmail,
			Identifier: "user@example.com",
			Claims:     map[string]interface{}{"sub": "user123"},
		},
		Data: map[string]interface{}{
			"session_id": "session-123",
			"reason":     "admin_action",
		},
		Metadata: EventMetadata{
			TransmitterID: "test-transmitter",
			ProcessingID:  "proc-123",
		},
	}

	// Convert to JSON
	data, err := originalEvent.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Convert back from JSON
	deserializedEvent, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	// Compare key fields
	if deserializedEvent.ID != originalEvent.ID {
		t.Errorf("ID mismatch: got %v, want %v", deserializedEvent.ID, originalEvent.ID)
	}

	if deserializedEvent.Type != originalEvent.Type {
		t.Errorf("Type mismatch: got %v, want %v", deserializedEvent.Type, originalEvent.Type)
	}

	if deserializedEvent.Subject.Identifier != originalEvent.Subject.Identifier {
		t.Errorf("Subject identifier mismatch: got %v, want %v", deserializedEvent.Subject.Identifier, originalEvent.Subject.Identifier)
	}

	if deserializedEvent.Metadata.TransmitterID != originalEvent.Metadata.TransmitterID {
		t.Errorf("Transmitter ID mismatch: got %v, want %v", deserializedEvent.Metadata.TransmitterID, originalEvent.Metadata.TransmitterID)
	}
}

func TestEventFilter_Matches(t *testing.T) {
	event := &SecurityEvent{
		ID:     "test-event-123",
		Type:   EventTypeSessionRevoked,
		Source: "test-issuer",
		Subject: Subject{
			Format:     SubjectFormatEmail,
			Identifier: "user@example.com",
			Claims: map[string]interface{}{
				"department": "engineering",
			},
		},
		Data: map[string]interface{}{
			"session_id": "session-123",
			"reason":     "admin_action",
		},
		Metadata: EventMetadata{
			TransmitterID: "test-transmitter",
		},
	}

	tests := []struct {
		name   string
		filter EventFilter
		want   bool
	}{
		{
			name: "equals - match",
			filter: EventFilter{
				Field:    "type",
				Operator: FilterOpEquals,
				Value:    EventTypeSessionRevoked,
			},
			want: true,
		},
		{
			name: "equals - no match",
			filter: EventFilter{
				Field:    "type",
				Operator: FilterOpEquals,
				Value:    EventTypeCredentialChange,
			},
			want: false,
		},
		{
			name: "contains - match",
			filter: EventFilter{
				Field:    "subject.identifier",
				Operator: FilterOpContains,
				Value:    "example.com",
			},
			want: true,
		},
		{
			name: "contains - no match",
			filter: EventFilter{
				Field:    "subject.identifier",
				Operator: FilterOpContains,
				Value:    "different.com",
			},
			want: false,
		},
		{
			name: "in - match",
			filter: EventFilter{
				Field:    "type",
				Operator: FilterOpIn,
				Value:    []interface{}{EventTypeSessionRevoked, EventTypeCredentialChange},
			},
			want: true,
		},
		{
			name: "in - no match",
			filter: EventFilter{
				Field:    "type",
				Operator: FilterOpIn,
				Value:    []interface{}{EventTypeCredentialChange, EventTypeAssuranceLevelChange},
			},
			want: false,
		},
		{
			name: "exists - field exists",
			filter: EventFilter{
				Field:    "id",
				Operator: FilterOpExists,
			},
			want: true,
		},
		{
			name: "exists - field doesn't exist",
			filter: EventFilter{
				Field:    "nonexistent_field",
				Operator: FilterOpExists,
			},
			want: false,
		},
		{
			name: "data field - match",
			filter: EventFilter{
				Field:    "data.reason",
				Operator: FilterOpEquals,
				Value:    "admin_action",
			},
			want: true,
		},
		{
			name: "subject claims - match",
			filter: EventFilter{
				Field:    "subject.department",
				Operator: FilterOpEquals,
				Value:    "engineering",
			},
			want: true,
		},
		{
			name: "metadata field - match",
			filter: EventFilter{
				Field:    "metadata.transmitter_id",
				Operator: FilterOpEquals,
				Value:    "test-transmitter",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Matches(event)
			if got != tt.want {
				t.Errorf("EventFilter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventFilter_extractFieldValue(t *testing.T) {
	event := &SecurityEvent{
		ID:     "test-event-123",
		Type:   EventTypeSessionRevoked,
		Source: "test-issuer",
		Subject: Subject{
			Format:     SubjectFormatEmail,
			Identifier: "user@example.com",
			Claims: map[string]interface{}{
				"department": "engineering",
			},
		},
		Data: map[string]interface{}{
			"session_id": "session-123",
		},
		Metadata: EventMetadata{
			TransmitterID: "test-transmitter",
		},
	}

	tests := []struct {
		name  string
		field string
		want  interface{}
	}{
		{"id", "id", "test-event-123"},
		{"type", "type", EventTypeSessionRevoked},
		{"source", "source", "test-issuer"},
		{"subject.format", "subject.format", SubjectFormatEmail},
		{"subject.identifier", "subject.identifier", "user@example.com"},
		{"data.session_id", "data.session_id", "session-123"},
		{"subject.department", "subject.department", "engineering"},
		{"metadata.transmitter_id", "metadata.transmitter_id", "test-transmitter"},
		{"nonexistent", "nonexistent", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &EventFilter{Field: tt.field}
			got := filter.extractFieldValue(event)
			if got != tt.want {
				t.Errorf("extractFieldValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Test all CAEP event type constants
	caepEvents := map[string]string{
		"EventTypeSessionRevoked":         EventTypeSessionRevoked,
		"EventTypeAssuranceLevelChange":   EventTypeAssuranceLevelChange,
		"EventTypeCredentialChange":       EventTypeCredentialChange,
		"EventTypeDeviceComplianceChange": EventTypeDeviceComplianceChange,
		"EventTypeTokenClaimsChange":      EventTypeTokenClaimsChange,
	}

	expectedCAEPPrefix := "https://schemas.openid.net/secevent/caep/event-type/"
	for name, uri := range caepEvents {
		if !strings.HasPrefix(uri, expectedCAEPPrefix) {
			t.Errorf("CAEP event %s URI %s does not have expected prefix %s", name, uri, expectedCAEPPrefix)
		}
		if uri == "" {
			t.Errorf("CAEP event %s URI is empty", name)
		}
	}

	// Test all RISC event type constants
	riscEvents := map[string]string{
		"EventTypeAccountCredentialChangeRequired": EventTypeAccountCredentialChangeRequired,
		"EventTypeAccountPurged":                   EventTypeAccountPurged,
		"EventTypeAccountDisabled":                 EventTypeAccountDisabled,
		"EventTypeAccountEnabled":                  EventTypeAccountEnabled,
		"EventTypeIdentifierChanged":               EventTypeIdentifierChanged,
		"EventTypeIdentifierRecycled":              EventTypeIdentifierRecycled,
		"EventTypeCredentialCompromise":            EventTypeCredentialCompromise,
		"EventTypeOptIn":                           EventTypeOptIn,
		"EventTypeOptOut":                          EventTypeOptOut,
		"EventTypeRecoveryActivated":               EventTypeRecoveryActivated,
		"EventTypeRecoveryInformationChanged":      EventTypeRecoveryInformationChanged,
	}

	expectedRISCPrefix := "https://schemas.openid.net/secevent/risc/event-type/"
	for name, uri := range riscEvents {
		if !strings.HasPrefix(uri, expectedRISCPrefix) {
			t.Errorf("RISC event %s URI %s does not have expected prefix %s", name, uri, expectedRISCPrefix)
		}
		if uri == "" {
			t.Errorf("RISC event %s URI is empty", name)
		}
	}

	// Test SSF event type constants
	ssfEvents := map[string]string{
		"EventTypeVerification": EventTypeVerification,
	}

	expectedSSFPrefix := "https://schemas.openid.net/secevent/ssf/event-type/"
	for name, uri := range ssfEvents {
		if !strings.HasPrefix(uri, expectedSSFPrefix) {
			t.Errorf("SSF event %s URI %s does not have expected prefix %s", name, uri, expectedSSFPrefix)
		}
		if uri == "" {
			t.Errorf("SSF event %s URI is empty", name)
		}
	}
}

func TestEventTypeUniqueness(t *testing.T) {
	// Collect all event type constants to ensure they are unique
	allEventTypes := []string{
		// CAEP Events
		EventTypeSessionRevoked,
		EventTypeAssuranceLevelChange,
		EventTypeCredentialChange,
		EventTypeDeviceComplianceChange,
		EventTypeTokenClaimsChange,
		// RISC Events
		EventTypeAccountCredentialChangeRequired,
		EventTypeAccountPurged,
		EventTypeAccountDisabled,
		EventTypeAccountEnabled,
		EventTypeIdentifierChanged,
		EventTypeIdentifierRecycled,
		EventTypeCredentialCompromise,
		EventTypeOptIn,
		EventTypeOptOut,
		EventTypeRecoveryActivated,
		EventTypeRecoveryInformationChanged,
		// SSF Events
		EventTypeVerification,
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, eventType := range allEventTypes {
		if seen[eventType] {
			t.Errorf("Duplicate event type found: %s", eventType)
		}
		seen[eventType] = true
	}

	// Verify we have the expected number of unique event types
	expectedCount := 17 // 5 CAEP + 11 RISC + 1 SSF
	if len(allEventTypes) != expectedCount {
		t.Errorf("Expected %d event types, got %d", expectedCount, len(allEventTypes))
	}
}

func TestNewEventTypesInFiltering(t *testing.T) {
	// Test that new event types work with the filtering system
	event := &SecurityEvent{
		ID:     "test-event-123",
		Type:   EventTypeTokenClaimsChange, // Using one of the new event types
		Source: "test-issuer",
		Subject: Subject{
			Format:     SubjectFormatEmail,
			Identifier: "user@example.com",
		},
		Data: map[string]interface{}{
			"previous_claims": map[string]interface{}{"role": "user"},
			"current_claims":  map[string]interface{}{"role": "admin"},
		},
		Metadata: EventMetadata{
			TransmitterID: "test-transmitter",
		},
	}

	tests := []struct {
		name      string
		filter    EventFilter
		eventType string
		want      bool
	}{
		{
			name: "token claims change - exact match",
			filter: EventFilter{
				Field:    "type",
				Operator: FilterOpEquals,
				Value:    EventTypeTokenClaimsChange,
			},
			eventType: EventTypeTokenClaimsChange,
			want:      true,
		},
		{
			name: "account purged - no match",
			filter: EventFilter{
				Field:    "type",
				Operator: FilterOpEquals,
				Value:    EventTypeAccountPurged,
			},
			eventType: EventTypeTokenClaimsChange,
			want:      false,
		},
		{
			name: "multiple new types - in filter",
			filter: EventFilter{
				Field:    "type",
				Operator: FilterOpIn,
				Value: []interface{}{
					EventTypeTokenClaimsChange,
					EventTypeAccountPurged,
					EventTypeCredentialCompromise,
				},
			},
			eventType: EventTypeTokenClaimsChange,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event.Type = tt.eventType
			got := tt.filter.Matches(event)
			if got != tt.want {
				t.Errorf("EventFilter.Matches() with %s = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}
