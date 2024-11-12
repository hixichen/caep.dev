package ssf

import (
	"encoding/json"

	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// StreamStatus represents the status of a stream
type StreamStatus string

const (
	StreamStatusEnabled  StreamStatus = "enabled"
	StreamStatusPaused   StreamStatus = "paused"
	StreamStatusDisabled StreamStatus = "disabled"
)

// StreamUpdateEvent represents a stream update event
type StreamUpdateEvent struct {
	BaseEvent
	Status StreamStatus `json:"status"`
	Reason *string      `json:"reason,omitempty"`
}

// NewStreamUpdateEvent creates a new stream update event
func NewStreamUpdateEvent(status StreamStatus) *StreamUpdateEvent {
	e := &StreamUpdateEvent{
		Status: status,
	}

	e.SetType(EventTypeStreamUpdate)

	return e
}

// WithReason sets the reason for the status change
func (e *StreamUpdateEvent) WithReason(reason string) *StreamUpdateEvent {
	e.Reason = &reason

	return e
}

// GetReason returns the reason if set
func (e *StreamUpdateEvent) GetReason() (string, bool) {
	if e.Reason == nil {
		return "", false
	}

	return *e.Reason, true
}

// GetStatus returns the current status
func (e *StreamUpdateEvent) GetStatus() StreamStatus {
	return e.Status
}

// Validate ensures the event is valid
func (e *StreamUpdateEvent) Validate() error {
	switch e.Status {
	case StreamStatusEnabled, StreamStatusPaused, StreamStatusDisabled:
		return nil
	default:
		return event.NewError(event.ErrCodeInvalidValue,
			"invalid stream status",
			"status")
	}
}

// MarshalJSON implements the json.Marshaler interface
func (e *StreamUpdateEvent) MarshalJSON() ([]byte, error) {
	payload := map[string]interface{}{
		"status": e.Status,
	}

	if e.Reason != nil {
		payload["reason"] = *e.Reason
	}

	return MarshalEventJSON(e.Type(), payload)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (e *StreamUpdateEvent) UnmarshalJSON(data []byte) error {
	eventData, err := UnmarshalEventJSON(data, EventTypeStreamUpdate)
	if err != nil {
		return err
	}

	var payload struct {
		Status StreamStatus `json:"status"`
		Reason *string      `json:"reason,omitempty"`
	}

	if err := json.Unmarshal(eventData, &payload); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to parse stream update event data", "")
	}

	e.SetType(EventTypeStreamUpdate)

	e.Status = payload.Status
	e.Reason = payload.Reason

	if err := e.Validate(); err != nil {
		return err
	}

	return nil
}

func parseStreamUpdateEvent(data json.RawMessage) (event.Event, error) {
	var e StreamUpdateEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, event.NewError(event.ErrCodeParseError,
			"failed to parse stream update event", "")
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	return &e, nil
}

func init() {
	event.RegisterEventParser(EventTypeStreamUpdate, parseStreamUpdateEvent)
}
