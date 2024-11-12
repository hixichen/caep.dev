package ssf

import (
	"encoding/json"
	"fmt"

	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// BaseEvent provides common SSF event functionality
type BaseEvent struct {
	event.BaseEvent
}

// MarshalEventJSON is a helper function for SSF events to marshal their JSON
func MarshalEventJSON(eventType event.EventType, payload interface{}) ([]byte, error) {
	eventMap := map[event.EventType]interface{}{
		eventType: payload,
	}

	return json.Marshal(eventMap)
}

// UnmarshalEventJSON is a helper function for SSF events to unmarshal their JSON
func UnmarshalEventJSON(data []byte, eventType event.EventType) (json.RawMessage, error) {
	var eventMap map[event.EventType]json.RawMessage

	if err := json.Unmarshal(data, &eventMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event wrapper: %w", err)
	}

	eventData, ok := eventMap[eventType]
	if !ok {
		return nil, event.NewError(event.ErrCodeInvalidEventType,
			fmt.Sprintf("event type %s not found", eventType),
			"events")
	}

	return eventData, nil
}
