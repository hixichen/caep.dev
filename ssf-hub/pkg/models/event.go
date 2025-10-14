package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SecurityEvent represents a processed security event ready for distribution
type SecurityEvent struct {
	ID          string                 `json:"id"`                     // Unique event ID (JTI)
	Type        string                 `json:"type"`                   // Event type URI
	Source      string                 `json:"source"`                 // Source transmitter/issuer
	SpecVersion string                 `json:"spec_version,omitempty"` // SSF spec version
	Time        time.Time              `json:"time"`                   // Event timestamp
	Subject     Subject                `json:"subject"`                // Event subject
	Data        map[string]interface{} `json:"data"`                   // Event-specific data
	Extensions  map[string]interface{} `json:"extensions,omitempty"`   // Custom extensions
	Metadata    EventMetadata          `json:"metadata"`               // Processing metadata
}

// Subject represents the subject of a security event
type Subject struct {
	Format     string                 `json:"format"`           // Subject format (email, phone, iss_sub, etc.)
	Identifier string                 `json:"identifier"`       // Subject identifier
	Claims     map[string]interface{} `json:"claims,omitempty"` // Additional subject claims
}

// EventMetadata contains metadata about event processing
type EventMetadata struct {
	ReceivedAt    time.Time         `json:"received_at"`       // When hub received the event
	ProcessedAt   time.Time         `json:"processed_at"`      // When hub processed the event
	TransmitterID string            `json:"transmitter_id"`    // ID of the transmitter
	RawSET        string            `json:"raw_set,omitempty"` // Original SET token
	ProcessingID  string            `json:"processing_id"`     // Unique processing ID
	Tags          map[string]string `json:"tags,omitempty"`    // Processing tags
}

// EventDelivery represents an event delivery attempt to a receiver
type EventDelivery struct {
	DeliveryID   string            `json:"delivery_id"`
	ReceiverID   string            `json:"receiver_id"`
	EventID      string            `json:"event_id"`
	Attempt      int               `json:"attempt"`
	Status       DeliveryStatus    `json:"status"`
	DeliveredAt  time.Time         `json:"delivered_at,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	ResponseCode int               `json:"response_code,omitempty"`
	ResponseBody string            `json:"response_body,omitempty"`
	Duration     time.Duration     `json:"duration,omitempty"`
	NextRetryAt  time.Time         `json:"next_retry_at,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
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
	// CAEP Events - Continuous Access Evaluation Profile
	EventTypeSessionRevoked         = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	EventTypeAssuranceLevelChange   = "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
	EventTypeCredentialChange       = "https://schemas.openid.net/secevent/caep/event-type/credential-change"
	EventTypeDeviceComplianceChange = "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change"
	EventTypeTokenClaimsChange      = "https://schemas.openid.net/secevent/caep/event-type/token-claims-change"

	// RISC Events - Risk Incident Sharing and Coordination
	EventTypeAccountCredentialChangeRequired = "https://schemas.openid.net/secevent/risc/event-type/account-credential-change-required"
	EventTypeAccountPurged                   = "https://schemas.openid.net/secevent/risc/event-type/account-purged"
	EventTypeAccountDisabled                 = "https://schemas.openid.net/secevent/risc/event-type/account-disabled"
	EventTypeAccountEnabled                  = "https://schemas.openid.net/secevent/risc/event-type/account-enabled"
	EventTypeIdentifierChanged               = "https://schemas.openid.net/secevent/risc/event-type/identifier-changed"
	EventTypeIdentifierRecycled              = "https://schemas.openid.net/secevent/risc/event-type/identifier-recycled"
	EventTypeCredentialCompromise            = "https://schemas.openid.net/secevent/risc/event-type/credential-compromise"
	EventTypeOptIn                           = "https://schemas.openid.net/secevent/risc/event-type/opt-in"
	EventTypeOptOut                          = "https://schemas.openid.net/secevent/risc/event-type/opt-out"
	EventTypeRecoveryActivated               = "https://schemas.openid.net/secevent/risc/event-type/recovery-activated"
	EventTypeRecoveryInformationChanged      = "https://schemas.openid.net/secevent/risc/event-type/recovery-information-changed"

	// SSF Events - Shared Signals Framework
	EventTypeVerification = "https://schemas.openid.net/secevent/ssf/event-type/verification"
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

// InternalMessage represents the internal message schema used in the unified Pub/Sub topic
type InternalMessage struct {
	MessageID   string          `json:"message_id"`   // Unique message ID
	MessageType string          `json:"message_type"` // Always "security_event" for events
	Version     string          `json:"version"`      // Schema version
	Timestamp   time.Time       `json:"timestamp"`    // Message creation time
	Event       *SecurityEvent  `json:"event"`        // The actual security event
	Routing     RoutingInfo     `json:"routing"`      // Routing and delivery information
	Metadata    MessageMetadata `json:"metadata"`     // Message metadata
}

// RoutingInfo contains information for event routing and delivery
type RoutingInfo struct {
	TargetReceivers []string          `json:"target_receivers"` // List of receiver IDs that should receive this event
	EventType       string            `json:"event_type"`       // Event type for routing
	Subject         string            `json:"subject"`          // Subject identifier for routing
	Priority        int               `json:"priority"`         // Message priority (0=normal, 1=high, 2=urgent)
	TTL             time.Duration     `json:"ttl"`              // Time-to-live for the message
	Tags            map[string]string `json:"tags"`             // Custom routing tags
}

// MessageMetadata contains metadata about the internal message
type MessageMetadata struct {
	HubInstanceID string    `json:"hub_instance_id"` // ID of the hub instance that created this message
	ProcessingID  string    `json:"processing_id"`   // Links to the original event processing
	RetryCount    int       `json:"retry_count"`     // Number of retries for this message
	OriginalTopic string    `json:"original_topic"`  // For migration/debugging (can be removed later)
	CreatedAt     time.Time `json:"created_at"`      // When this internal message was created
	UpdatedAt     time.Time `json:"updated_at"`      // Last update time
}

// ToInternalMessage converts a SecurityEvent to an InternalMessage for the unified topic
func (e *SecurityEvent) ToInternalMessage(targetReceivers []string, hubInstanceID string) *InternalMessage {
	return &InternalMessage{
		MessageID:   generateMessageID(),
		MessageType: "security_event",
		Version:     "1.0",
		Timestamp:   time.Now(),
		Event:       e,
		Routing: RoutingInfo{
			TargetReceivers: targetReceivers,
			EventType:       e.Type,
			Subject:         e.Subject.Identifier,
			Priority:        0,              // Default normal priority
			TTL:             24 * time.Hour, // Default 24 hour TTL
			Tags:            make(map[string]string),
		},
		Metadata: MessageMetadata{
			HubInstanceID: hubInstanceID,
			ProcessingID:  e.Metadata.ProcessingID,
			RetryCount:    0,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}
}

// ToUnifiedPubSubMessage converts InternalMessage to Pub/Sub format for the unified topic
func (m *InternalMessage) ToUnifiedPubSubMessage() ([]byte, map[string]string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, nil, err
	}

	attributes := map[string]string{
		"message_id":      m.MessageID,
		"message_type":    m.MessageType,
		"version":         m.Version,
		"event_id":        m.Event.ID,
		"event_type":      m.Event.Type,
		"source":          m.Event.Source,
		"subject_format":  m.Event.Subject.Format,
		"transmitter_id":  m.Event.Metadata.TransmitterID,
		"processing_id":   m.Event.Metadata.ProcessingID,
		"hub_instance_id": m.Metadata.HubInstanceID,
		"priority":        string(rune(m.Routing.Priority + '0')), // Convert int to string
		"retry_count":     string(rune(m.Metadata.RetryCount + '0')),
	}

	// Add target receivers as a comma-separated attribute for filtering
	if len(m.Routing.TargetReceivers) > 0 {
		attributes["target_receivers"] = strings.Join(m.Routing.TargetReceivers, ",")
	}

	return data, attributes, nil
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return "msg_" + time.Now().Format("20060102150405") + "_" + generateShortID()
}

// generateShortID generates a short unique ID
func generateShortID() string {
	// Simple implementation - could use UUID library for better uniqueness
	return fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
}

// FromJSON deserializes an event from JSON
func FromJSON(data []byte) (*SecurityEvent, error) {
	var event SecurityEvent
	err := json.Unmarshal(data, &event)
	return &event, err
}

// FromInternalMessageJSON deserializes an InternalMessage from JSON
func FromInternalMessageJSON(data []byte) (*InternalMessage, error) {
	var message InternalMessage
	err := json.Unmarshal(data, &message)
	return &message, err
}
