package caep

import (
	"encoding/json"

	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// SessionRevokedEvent represents a session revoked event
type SessionRevokedEvent struct {
	BaseEvent
	// No additional fields; uses only CAEP metadata
}

// NewSessionRevokedEvent creates a new session revoked event
func NewSessionRevokedEvent() *SessionRevokedEvent {
	e := &SessionRevokedEvent{}

	e.SetType(EventTypeSessionRevoked)

	return e
}

// Validate ensures the event is valid
func (e *SessionRevokedEvent) Validate() error {
	return e.ValidateMetadata()
}

// MarshalJSON implements the json.Marshaler interface
func (e *SessionRevokedEvent) MarshalJSON() ([]byte, error) {
	payload := make(map[string]interface{})

	if e.Metadata != nil {
		payload["metadata"] = e.Metadata
	}

	eventMap := map[event.EventType]interface{}{
		e.Type(): payload,
	}

	return json.Marshal(eventMap)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (e *SessionRevokedEvent) UnmarshalJSON(data []byte) error {
	var eventMap map[event.EventType]json.RawMessage

	if err := json.Unmarshal(data, &eventMap); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to unmarshal event wrapper", "")
	}

	eventData, ok := eventMap[EventTypeSessionRevoked]
	if !ok {
		return event.NewError(event.ErrCodeInvalidEventType,
			"session revoked event not found", "events")
	}

	// Since there are no specific fields, we only need to parse metadata
	var payload struct {
		Metadata *EventMetadata `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(eventData, &payload); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to parse session revoked event data", "")
	}

	e.SetType(EventTypeSessionRevoked)
	e.Metadata = payload.Metadata

	if err := e.Validate(); err != nil {
		return err
	}

	return nil
}

// WithEventTimestamp sets the event timestamp
func (e *SessionRevokedEvent) WithEventTimestamp(timestamp int64) *SessionRevokedEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithEventTimestamp(timestamp)

	return e
}

// WithInitiatingEntity sets who/what initiated the event
func (e *SessionRevokedEvent) WithInitiatingEntity(entity InitiatingEntity) *SessionRevokedEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithInitiatingEntity(entity)

	return e
}

// WithReasonAdmin adds an admin reason
func (e *SessionRevokedEvent) WithReasonAdmin(language, reason string) *SessionRevokedEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonAdmin(language, reason)

	return e
}

// WithReasonUser adds a user reason
func (e *SessionRevokedEvent) WithReasonUser(language, reason string) *SessionRevokedEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonUser(language, reason)

	return e
}

func parseSessionRevokedEvent(data json.RawMessage) (event.Event, error) {
	var e SessionRevokedEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, event.NewError(event.ErrCodeParseError,
			"failed to parse session revoked event", err.Error())
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	return &e, nil
}

func init() {
	event.RegisterEventParser(EventTypeSessionRevoked, parseSessionRevokedEvent)
}
