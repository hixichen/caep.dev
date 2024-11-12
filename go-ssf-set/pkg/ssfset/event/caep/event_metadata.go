package caep

import (
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// InitiatingEntity represents who/what initiated an event
type InitiatingEntity string

const (
	InitiatingEntityAdmin  InitiatingEntity = "admin"
	InitiatingEntityUser   InitiatingEntity = "user"
	InitiatingEntityPolicy InitiatingEntity = "policy"
	InitiatingEntitySystem InitiatingEntity = "system"
)

// EventMetadata represents optional metadata for CAEP events
type EventMetadata struct {
	EventTimestamp   *int64            `json:"event_timestamp,omitempty"`
	InitiatingEntity *InitiatingEntity `json:"initiating_entity,omitempty"`
	ReasonAdmin      map[string]string `json:"reason_admin,omitempty"`
	ReasonUser       map[string]string `json:"reason_user,omitempty"`
}

// NewEventMetadata creates a new event metadata instance
func NewEventMetadata() *EventMetadata {
	return &EventMetadata{}
}

// WithEventTimestamp sets the event timestamp
func (m *EventMetadata) WithEventTimestamp(timestamp int64) *EventMetadata {
	m.EventTimestamp = &timestamp

	return m
}

// WithInitiatingEntity sets who/what initiated the event
func (m *EventMetadata) WithInitiatingEntity(entity InitiatingEntity) *EventMetadata {
	m.InitiatingEntity = &entity

	return m
}

// WithReasonAdmin adds an admin reason in the specified language
func (m *EventMetadata) WithReasonAdmin(language, reason string) *EventMetadata {
	if m.ReasonAdmin == nil {
		m.ReasonAdmin = make(map[string]string)
	}

	m.ReasonAdmin[language] = reason

	return m
}

// WithReasonUser adds a user reason in the specified language
func (m *EventMetadata) WithReasonUser(language, reason string) *EventMetadata {
	if m.ReasonUser == nil {
		m.ReasonUser = make(map[string]string)
	}

	m.ReasonUser[language] = reason

	return m
}

// Validate validates the metadata
func (m *EventMetadata) Validate() error {
	if m.EventTimestamp != nil && *m.EventTimestamp < 0 {
		return event.NewError(event.ErrCodeInvalidValue,
			"event timestamp cannot be negative",
			"event_timestamp")
	}

	if m.InitiatingEntity != nil {
		switch *m.InitiatingEntity {
		case InitiatingEntityAdmin, InitiatingEntityUser, InitiatingEntityPolicy, InitiatingEntitySystem:
			// Valid initiating entity
		default:
			return event.NewError(event.ErrCodeInvalidValue,
				"invalid initiating entity",
				"initiating_entity")
		}
	}

	return nil
}
