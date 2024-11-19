// pkg/schemes/caep/event.go
package caep

import (
	"encoding/json"

	"github.com/sgnl-ai/caep.dev-receiver/secevent/pkg/set/event"
)

// CAEPEvent is the interface that all CAEP events must implement
type CAEPEvent interface {
	event.Event

	// Metadata operations
	GetMetadata() *EventMetadata
	SetMetadata(*EventMetadata)

	// Convenience methods for metadata fields
	WithEventTimestamp(int64) CAEPEvent
	WithInitiatingEntity(InitiatingEntity) CAEPEvent
	WithReasonAdmin(language, reason string) CAEPEvent
	WithReasonUser(language, reason string) CAEPEvent
}

// BaseCAEPEvent provides common CAEP event functionality
type BaseCAEPEvent struct {
	event.BaseEvent
	Metadata *EventMetadata `json:"metadata,omitempty"`
}

func (e *BaseCAEPEvent) GetMetadata() *EventMetadata {
	return e.Metadata
}

func (e *BaseCAEPEvent) SetMetadata(metadata *EventMetadata) {
	e.Metadata = metadata
}

func (e *BaseCAEPEvent) WithEventTimestamp(timestamp int64) CAEPEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithEventTimestamp(timestamp)

	return e
}

func (e *BaseCAEPEvent) WithInitiatingEntity(entity InitiatingEntity) CAEPEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithInitiatingEntity(entity)

	return e
}

func (e *BaseCAEPEvent) WithReasonAdmin(language, reason string) CAEPEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonAdmin(language, reason)

	return e
}

func (e *BaseCAEPEvent) WithReasonUser(language, reason string) CAEPEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonUser(language, reason)

	return e
}

func (e *BaseCAEPEvent) ValidateMetadata() error {
	if e.Metadata == nil {
		return nil
	}

	return e.Metadata.Validate()
}

func (e *BaseCAEPEvent) Validate() error {
	if err := e.ValidateMetadata(); err != nil {
		return err
	}

	return nil
}

func (e *BaseCAEPEvent) Payload() interface{} {
	return e.Metadata
}

func (e *BaseCAEPEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Payload())
}

func (e *BaseCAEPEvent) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Metadata *EventMetadata `json:"metadata,omitempty"`
	}

	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	e.Metadata = aux.Metadata

	return nil
}
