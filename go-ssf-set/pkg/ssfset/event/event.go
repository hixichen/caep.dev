package event

import (
	"encoding/json"
)

// EventType represents the type of an event
type EventType string

// Event is the interface that all event types must implement
type Event interface {
	// Type returns the event type URI
	Type() EventType

	// Validate checks if the event is valid
	Validate() error

	// MarshalJSON implements the json.Marshaler interface
	json.Marshaler

	// UnmarshalJSON implements the json.Unmarshaler interface
	json.Unmarshaler
}

// BaseEvent provides common functionality for all events
type BaseEvent struct {
	eventType EventType
}

// Type returns the event type URI
func (e *BaseEvent) Type() EventType {
	return e.eventType
}

// SetType sets the event type
func (e *BaseEvent) SetType(eventType EventType) {
	e.eventType = eventType
}