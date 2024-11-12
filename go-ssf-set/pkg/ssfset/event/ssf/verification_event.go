package ssf

import (
	"encoding/json"

	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// VerificationEvent represents a verification event
type VerificationEvent struct {
	BaseEvent
	State *string `json:"state,omitempty"`
}

// NewVerificationEvent creates a new verification event
func NewVerificationEvent() *VerificationEvent {
	e := &VerificationEvent{}

	e.SetType(EventTypeVerification)

	return e
}

// WithState sets the state value
func (e *VerificationEvent) WithState(state string) *VerificationEvent {
	e.State = &state

	return e
}

// GetState returns the state value if set
func (e *VerificationEvent) GetState() (string, bool) {
	if e.State == nil {
		return "", false
	}

	return *e.State, true
}

// Validate ensures the event is valid
func (e *VerificationEvent) Validate() error {
	// State is optional, so no validation needed
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (e *VerificationEvent) MarshalJSON() ([]byte, error) {
	payload := make(map[string]interface{})
	if e.State != nil {
		payload["state"] = *e.State
	}

	return MarshalEventJSON(e.Type(), payload)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (e *VerificationEvent) UnmarshalJSON(data []byte) error {
	eventData, err := UnmarshalEventJSON(data, EventTypeVerification)
	if err != nil {
		return err
	}

	var payload struct {
		State *string `json:"state,omitempty"`
	}

	if err := json.Unmarshal(eventData, &payload); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to parse verification event data", "")
	}

	e.SetType(EventTypeVerification)

	e.State = payload.State

	return nil
}

func parseVerificationEvent(data json.RawMessage) (event.Event, error) {
	var e VerificationEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, event.NewError(event.ErrCodeParseError,
			"failed to parse verification event", "")
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	return &e, nil
}

func init() {
	event.RegisterEventParser(EventTypeVerification, parseVerificationEvent)
}
