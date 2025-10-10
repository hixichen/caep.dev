package models

import (
	"encoding/json"
	"time"
)

// SecurityEvent represents a processed security event ready for distribution
type SecurityEvent struct {
	ID            string            `json:"id"`                      // Unique event ID (JTI)
	Type          string            `json:"type"`                    // Event type URI
	Source        string            `json:"source"`                  // Source transmitter/issuer
	SpecVersion   string            `json:"spec_version,omitempty"`  // SSF spec version
	Time          time.Time         `json:"time"`                    // Event timestamp
	Subject       Subject           `json:"subject"`                 // Event subject
	Data          map[string]interface{} `json:"data"`           // Event-specific data
	Extensions    map[string]interface{} `json:"extensions,omitempty"` // Custom extensions
	Metadata      EventMetadata     `json:"metadata"`                // Processing metadata
}

// Subject represents the subject of a security event
type Subject struct {
	Format     string                 `json:"format"`               // Subject format (email, phone, iss_sub, etc.)
	Identifier string                 `json:"identifier"`           // Subject identifier
	Claims     map[string]interface{} `json:"claims,omitempty"`     // Additional subject claims
}

// EventMetadata contains metadata about event processing
type EventMetadata struct {
	ReceivedAt    time.Time         `json:"received_at"`             // When broker received the event
	ProcessedAt   time.Time         `json:"processed_at"`            // When broker processed the event
	TransmitterID string            `json:"transmitter_id"`          // ID of the transmitter
	RawSET        string            `json:"raw_set,omitempty"`       // Original SET token
	ProcessingID  string            `json:"processing_id"`           // Unique processing ID
	Tags          map[string]string `json:"tags,omitempty"`          // Processing tags
}

// EventDelivery represents an event delivery attempt to a receiver
type EventDelivery struct {
	DeliveryID    string              `json:"delivery_id"`
	ReceiverID    string              `json:"receiver_id"`
	EventID       string              `json:"event_id"`
	Attempt       int                 `json:"attempt"`
	Status        DeliveryStatus      `json:"status"`
	DeliveredAt   time.Time           `json:"delivered_at,omitempty"`
	ErrorMessage  string              `json:"error_message,omitempty"`
	ResponseCode  int                 `json:"response_code,omitempty"`
	ResponseBody  string              `json:"response_body,omitempty"`
	Duration      time.Duration       `json:"duration,omitempty"`
	NextRetryAt   time.Time           `json:"next_retry_at,omitempty"`
	Metadata      map[string]string   `json:"metadata,omitempty"`
}

// DeliveryStatus represents the status of an event delivery
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
	DeliveryStatusAbandoned DeliveryStatus = "abandoned"
)

// Common event types
const (
	EventTypeSessionRevoked         = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	EventTypeAssuranceLevelChange   = "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
	EventTypeCredentialChange       = "https://schemas.openid.net/secevent/caep/event-type/credential-change"
	EventTypeDeviceComplianceChange = "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change"
	EventTypeVerification           = "https://schemas.openid.net/secevent/ssf/event-type/verification"
)

// Common subject formats
const (
	SubjectFormatEmail       = "email"
	SubjectFormatPhoneNumber = "phone_number"
	SubjectFormatIssSub      = "iss_sub"
	SubjectFormatOpaque      = "opaque"
	SubjectFormatDID         = "did"
	SubjectFormatURI         = "uri"
)

// EventFilter evaluates whether an event matches the filter criteria
func (f *EventFilter) Matches(event *SecurityEvent) bool {
	value := f.extractFieldValue(event)
	if value == nil {
		return f.Operator != FilterOpExists
	}

	switch f.Operator {
	case FilterOpEquals:
		return f.equals(value, f.Value)
	case FilterOpContains:
		return f.contains(value, f.Value)
	case FilterOpMatches:
		return f.matches(value, f.Value)
	case FilterOpIn:
		return f.in(value, f.Value)
	case FilterOpExists:
		return true
	default:
		return false
	}
}

// extractFieldValue extracts a field value from the event using dot notation
func (f *EventFilter) extractFieldValue(event *SecurityEvent) interface{} {
	switch f.Field {
	case "id":
		return event.ID
	case "type":
		return event.Type
	case "source":
		return event.Source
	case "subject.format":
		return event.Subject.Format
	case "subject.identifier":
		return event.Subject.Identifier
	case "metadata.transmitter_id":
		return event.Metadata.TransmitterID
	default:
		// Handle nested fields in data or extensions
		if len(f.Field) > 5 && f.Field[:5] == "data." {
			fieldName := f.Field[5:]
			return event.Data[fieldName]
		}
		if len(f.Field) > 11 && f.Field[:11] == "extensions." {
			fieldName := f.Field[11:]
			return event.Extensions[fieldName]
		}
		if len(f.Field) > 8 && f.Field[:8] == "subject." {
			fieldName := f.Field[8:]
			return event.Subject.Claims[fieldName]
		}
		return nil
	}
}

// equals checks if two values are equal
func (f *EventFilter) equals(a, b interface{}) bool {
	return a == b
}

// contains checks if value a contains value b (for strings)
func (f *EventFilter) contains(a, b interface{}) bool {
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if !aOk || !bOk {
		return false
	}
	return len(aStr) >= len(bStr) && aStr[len(aStr)-len(bStr):] == bStr
}

// matches checks if value a matches pattern b (basic pattern matching)
func (f *EventFilter) matches(a, b interface{}) bool {
	// Simple implementation - could be enhanced with regex
	return f.contains(a, b)
}

// in checks if value a is in slice b
func (f *EventFilter) in(a, b interface{}) bool {
	bSlice, ok := b.([]interface{})
	if !ok {
		return false
	}
	for _, item := range bSlice {
		if a == item {
			return true
		}
	}
	return false
}

// ToJSON serializes the event to JSON
func (e *SecurityEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ToPubSubMessage converts the event to a format suitable for Pub/Sub publishing
func (e *SecurityEvent) ToPubSubMessage() ([]byte, map[string]string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, nil, err
	}

	attributes := map[string]string{
		"event_id":       e.ID,
		"event_type":     e.Type,
		"source":         e.Source,
		"subject_format": e.Subject.Format,
		"transmitter_id": e.Metadata.TransmitterID,
		"processing_id":  e.Metadata.ProcessingID,
	}

	return data, attributes, nil
}

// FromJSON deserializes an event from JSON
func FromJSON(data []byte) (*SecurityEvent, error) {
	var event SecurityEvent
	err := json.Unmarshal(data, &event)
	return &event, err
}