package caep

import (
	"encoding/json"

	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// TokenClaimsChangeEvent represents a token claims change event
type TokenClaimsChangeEvent struct {
	BaseEvent
	Claims map[string]interface{} `json:"claims"` // REQUIRED
}

// NewTokenClaimsChangeEvent creates a new token claims change event
func NewTokenClaimsChangeEvent() *TokenClaimsChangeEvent {
	e := &TokenClaimsChangeEvent{
		Claims: make(map[string]interface{}),
	}

	e.SetType(EventTypeTokenClaimsChange)

	return e
}

// WithClaim adds a claim with its new value
func (e *TokenClaimsChangeEvent) WithClaim(name string, value interface{}) *TokenClaimsChangeEvent {
	e.Claims[name] = value

	return e
}

// Validate ensures the event is valid
func (e *TokenClaimsChangeEvent) Validate() error {
	// Validate metadata
	if err := e.ValidateMetadata(); err != nil {
		return err
	}

	// Claims are required and must not be empty
	if len(e.Claims) == 0 {
		return event.NewError(event.ErrCodeMissingValue,
			"claims are required and must not be empty",
			"claims")
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (e *TokenClaimsChangeEvent) MarshalJSON() ([]byte, error) {
	payload := map[string]interface{}{
		"claims": e.Claims,
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
func (e *TokenClaimsChangeEvent) UnmarshalJSON(data []byte) error {
	var eventMap map[event.EventType]json.RawMessage

	if err := json.Unmarshal(data, &eventMap); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to unmarshal event wrapper", "")
	}

	eventData, ok := eventMap[EventTypeTokenClaimsChange]
	if !ok {
		return event.NewError(event.ErrCodeInvalidEventType,
			"token claims change event not found", "events")
	}

	var payload struct {
		Claims   map[string]interface{} `json:"claims"`
		Metadata *EventMetadata         `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(eventData, &payload); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to parse token claims change event data", "")
	}

	e.SetType(EventTypeTokenClaimsChange)

	e.Claims = payload.Claims
	e.Metadata = payload.Metadata

	if err := e.Validate(); err != nil {
		return err
	}

	return nil
}

// WithEventTimestamp sets the event timestamp
func (e *TokenClaimsChangeEvent) WithEventTimestamp(timestamp int64) *TokenClaimsChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithEventTimestamp(timestamp)

	return e
}

// WithInitiatingEntity sets who/what initiated the event
func (e *TokenClaimsChangeEvent) WithInitiatingEntity(entity InitiatingEntity) *TokenClaimsChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithInitiatingEntity(entity)

	return e
}

// WithReasonAdmin adds an admin reason
func (e *TokenClaimsChangeEvent) WithReasonAdmin(language, reason string) *TokenClaimsChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonAdmin(language, reason)

	return e
}

// WithReasonUser adds a user reason
func (e *TokenClaimsChangeEvent) WithReasonUser(language, reason string) *TokenClaimsChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonUser(language, reason)

	return e
}

func parseTokenClaimsChangeEvent(data json.RawMessage) (event.Event, error) {
	var e TokenClaimsChangeEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, event.NewError(event.ErrCodeParseError,
			"failed to parse token claims change event", "")
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	return &e, nil
}

func init() {
	event.RegisterEventParser(EventTypeTokenClaimsChange, parseTokenClaimsChangeEvent)
}
