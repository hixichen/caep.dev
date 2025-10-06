package models

import (
	"encoding/json"
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