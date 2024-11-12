package caep

import (
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// CAEPEvent is the interface that all CAEP events must implement
type CAEPEvent interface {
	event.Event

	// Metadata operations
	GetMetadata() *EventMetadata

	// Convenience methods for metadata fields
	WithEventTimestamp(int64) CAEPEvent
	WithInitiatingEntity(InitiatingEntity) CAEPEvent
	WithReasonAdmin(language, reason string) CAEPEvent
	WithReasonUser(language, reason string) CAEPEvent
}

// BaseEvent provides common CAEP event functionality
type BaseEvent struct {
	event.BaseEvent
	Metadata *EventMetadata `json:"metadata,omitempty"`
}

// GetMetadata returns the event metadata
func (e *BaseEvent) GetMetadata() *EventMetadata {
	return e.Metadata
}

// ValidateMetadata validates the CAEP metadata if present
func (e *BaseEvent) ValidateMetadata() error {
	if e.Metadata == nil {
		return nil
	}

	return e.Metadata.Validate()
}
