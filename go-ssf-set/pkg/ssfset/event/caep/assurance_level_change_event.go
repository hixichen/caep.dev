package caep

import (
	"encoding/json"
	"fmt"

	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// AssuranceLevel represents NIST Authenticator Assurance Level (AAL)
type AssuranceLevel string

const (
	AssuranceLevelAAL1 AssuranceLevel = "nist-aal1"
	AssuranceLevelAAL2 AssuranceLevel = "nist-aal2"
	AssuranceLevelAAL3 AssuranceLevel = "nist-aal3"
)

// ChangeDirection represents the direction of assurance level change
type ChangeDirection string

const (
	ChangeDirectionIncrease ChangeDirection = "increase"
	ChangeDirectionDecrease ChangeDirection = "decrease"
)

// AssuranceLevelChangeEvent represents an assurance level change event
type AssuranceLevelChangeEvent struct {
	BaseEvent
	CurrentLevel    AssuranceLevel  `json:"current_level"`
	PreviousLevel   AssuranceLevel  `json:"previous_level"`
	ChangeDirection ChangeDirection `json:"change_direction"`
}

// NewAssuranceLevelChangeEvent creates a new assurance level change event
func NewAssuranceLevelChangeEvent(
	currentLevel, previousLevel AssuranceLevel,
	direction ChangeDirection,
) *AssuranceLevelChangeEvent {
	e := &AssuranceLevelChangeEvent{
		CurrentLevel:    currentLevel,
		PreviousLevel:   previousLevel,
		ChangeDirection: direction,
	}

	e.SetType(EventTypeAssuranceLevelChange)

	return e
}

// Validate ensures the event is valid
func (e *AssuranceLevelChangeEvent) Validate() error {
	if err := e.ValidateMetadata(); err != nil {
		return err
	}

	// Validate required fields
	validLevels := map[AssuranceLevel]bool{
		AssuranceLevelAAL1: true,
		AssuranceLevelAAL2: true,
		AssuranceLevelAAL3: true,
	}

	if !validLevels[e.CurrentLevel] {
		return event.NewError(event.ErrCodeInvalidValue,
			fmt.Sprintf("invalid current level: %s", e.CurrentLevel),
			"current_level")
	}

	if !validLevels[e.PreviousLevel] {
		return event.NewError(event.ErrCodeInvalidValue,
			fmt.Sprintf("invalid previous level: %s", e.PreviousLevel),
			"previous_level")
	}

	if e.ChangeDirection != ChangeDirectionIncrease && e.ChangeDirection != ChangeDirectionDecrease {
		return event.NewError(event.ErrCodeInvalidValue,
			fmt.Sprintf("invalid change direction: %s", e.ChangeDirection),
			"change_direction")
	}

	if e.CurrentLevel == e.PreviousLevel {
		return event.NewError(event.ErrCodeInvalidValue,
			"current and previous levels must be different",
			"levels")
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (e *AssuranceLevelChangeEvent) MarshalJSON() ([]byte, error) {
	payload := map[string]interface{}{
		"current_level":    e.CurrentLevel,
		"previous_level":   e.PreviousLevel,
		"change_direction": e.ChangeDirection,
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
func (e *AssuranceLevelChangeEvent) UnmarshalJSON(data []byte) error {
	var eventMap map[event.EventType]json.RawMessage

	if err := json.Unmarshal(data, &eventMap); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to unmarshal event wrapper", "")
	}

	eventData, ok := eventMap[EventTypeAssuranceLevelChange]
	if !ok {
		return event.NewError(event.ErrCodeInvalidEventType,
			"assurance level change event not found", "events")
	}

	var payload struct {
		CurrentLevel    AssuranceLevel  `json:"current_level"`
		PreviousLevel   AssuranceLevel  `json:"previous_level"`
		ChangeDirection ChangeDirection `json:"change_direction"`
		Metadata        *EventMetadata  `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(eventData, &payload); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to parse assurance level change event data", "")
	}

	e.SetType(EventTypeAssuranceLevelChange)
	e.CurrentLevel = payload.CurrentLevel
	e.PreviousLevel = payload.PreviousLevel
	e.ChangeDirection = payload.ChangeDirection
	e.Metadata = payload.Metadata

	if err := e.Validate(); err != nil {
		return err
	}

	return nil
}

// WithEventTimestamp sets the event timestamp
func (e *AssuranceLevelChangeEvent) WithEventTimestamp(timestamp int64) *AssuranceLevelChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithEventTimestamp(timestamp)

	return e
}

// WithInitiatingEntity sets who/what initiated the event
func (e *AssuranceLevelChangeEvent) WithInitiatingEntity(entity InitiatingEntity) *AssuranceLevelChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithInitiatingEntity(entity)

	return e
}

// WithReasonAdmin adds an admin reason
func (e *AssuranceLevelChangeEvent) WithReasonAdmin(language, reason string) *AssuranceLevelChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonAdmin(language, reason)

	return e
}

// WithReasonUser adds a user reason
func (e *AssuranceLevelChangeEvent) WithReasonUser(language, reason string) *AssuranceLevelChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonUser(language, reason)

	return e
}

func parseAssuranceLevelChangeEvent(data json.RawMessage) (event.Event, error) {
	var e AssuranceLevelChangeEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, event.NewError(event.ErrCodeParseError,
			"failed to parse assurance level change event", "")
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	return &e, nil
}

func init() {
	event.RegisterEventParser(EventTypeAssuranceLevelChange, parseAssuranceLevelChangeEvent)
}
