package caep

import (
	"encoding/json"
	"fmt"

	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// ComplianceStatus represents the compliance status of a device
type ComplianceStatus string

const (
	ComplianceStatusCompliant    ComplianceStatus = "compliant"
	ComplianceStatusNotCompliant ComplianceStatus = "not-compliant"
)

// DeviceComplianceChangeEvent represents a device compliance change event
type DeviceComplianceChangeEvent struct {
	BaseEvent
	CurrentStatus  ComplianceStatus `json:"current_status"`
	PreviousStatus ComplianceStatus `json:"previous_status"`
}

// NewDeviceComplianceChangeEvent creates a new device compliance change event
func NewDeviceComplianceChangeEvent(
	currentStatus, previousStatus ComplianceStatus,
) *DeviceComplianceChangeEvent {
	e := &DeviceComplianceChangeEvent{
		CurrentStatus:  currentStatus,
		PreviousStatus: previousStatus,
	}

	e.SetType(EventTypeDeviceComplianceChange)

	return e
}

// Validate ensures the event is valid
func (e *DeviceComplianceChangeEvent) Validate() error {
	if err := e.ValidateMetadata(); err != nil {
		return err
	}

	validStatuses := map[ComplianceStatus]bool{
		ComplianceStatusCompliant:    true,
		ComplianceStatusNotCompliant: true,
	}

	if !validStatuses[e.CurrentStatus] {
		return event.NewError(event.ErrCodeInvalidValue,
			fmt.Sprintf("invalid current status: %s", e.CurrentStatus),
			"current_status")
	}

	if !validStatuses[e.PreviousStatus] {
		return event.NewError(event.ErrCodeInvalidValue,
			fmt.Sprintf("invalid previous status: %s", e.PreviousStatus),
			"previous_status")
	}

	if e.CurrentStatus == e.PreviousStatus {
		return event.NewError(event.ErrCodeInvalidValue,
			"current and previous status must be different",
			"status")
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (e *DeviceComplianceChangeEvent) MarshalJSON() ([]byte, error) {
	payload := map[string]interface{}{
		"current_status":  e.CurrentStatus,
		"previous_status": e.PreviousStatus,
	}

	if e.Metadata != nil {
		payload["metadata"] = e.Metadata
	}

	eventMap := map[event.EventType]interface{}{
		e.Type(): payload,
	}

	return json.Marshal(eventMap)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (e *DeviceComplianceChangeEvent) UnmarshalJSON(data []byte) error {
	var eventMap map[event.EventType]json.RawMessage

	if err := json.Unmarshal(data, &eventMap); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to unmarshal event wrapper", "")
	}

	eventData, ok := eventMap[EventTypeDeviceComplianceChange]
	if !ok {
		return event.NewError(event.ErrCodeInvalidEventType,
			"device compliance change event not found", "events")
	}

	var payload struct {
		CurrentStatus  ComplianceStatus `json:"current_status"`
		PreviousStatus ComplianceStatus `json:"previous_status"`
		Metadata       *EventMetadata   `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(eventData, &payload); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to parse device compliance change event data", "")
	}

	e.SetType(EventTypeDeviceComplianceChange)

	e.CurrentStatus = payload.CurrentStatus
	e.PreviousStatus = payload.PreviousStatus
	e.Metadata = payload.Metadata

	if err := e.Validate(); err != nil {
		return err
	}

	return nil
}

// WithEventTimestamp sets the event timestamp
func (e *DeviceComplianceChangeEvent) WithEventTimestamp(timestamp int64) *DeviceComplianceChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithEventTimestamp(timestamp)

	return e
}

// WithInitiatingEntity sets who/what initiated the event
func (e *DeviceComplianceChangeEvent) WithInitiatingEntity(entity InitiatingEntity) *DeviceComplianceChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithInitiatingEntity(entity)

	return e
}

// WithReasonAdmin adds an admin reason
func (e *DeviceComplianceChangeEvent) WithReasonAdmin(language, reason string) *DeviceComplianceChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonAdmin(language, reason)

	return e
}

// WithReasonUser adds a user reason
func (e *DeviceComplianceChangeEvent) WithReasonUser(language, reason string) *DeviceComplianceChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonUser(language, reason)

	return e
}

func parseDeviceComplianceChangeEvent(data json.RawMessage) (event.Event, error) {
	var e DeviceComplianceChangeEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, event.NewError(event.ErrCodeParseError,
			"failed to parse device compliance change event", "")
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	return &e, nil
}

func init() {
	event.RegisterEventParser(EventTypeDeviceComplianceChange, parseDeviceComplianceChangeEvent)
}
