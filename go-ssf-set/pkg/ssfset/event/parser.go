package event

import (
	"encoding/json"
	"fmt"
)

// Map of registered event parsers
var eventParsers = make(map[EventType]func(data json.RawMessage) (Event, error))

// RegisterEventParser allows users to register a parser for a custom event type
func RegisterEventParser(eventType EventType, parser func(data json.RawMessage) (Event, error)) {
	eventParsers[eventType] = parser
}

// ParseEvent parses a JSON object into the appropriate event type
func ParseEvent(data []byte) (Event, error) {
	var eventMap map[EventType]json.RawMessage
	if err := json.Unmarshal(data, &eventMap); err != nil {
		return nil, fmt.Errorf("failed to parse event map: %w", err) // Changed from "event container"
	}

	if len(eventMap) == 0 {
		return nil, NewError(ErrCodeMissingField, "event type is required", "") // Changed message and removed "events" path
	}

	if len(eventMap) > 1 {
		return nil, NewError(ErrCodeInvalidValue, "multiple events not supported", "") // Removed "events" path
	}

	// Extract the single event type and data
	var eventType EventType
	for k := range eventMap {
		eventType = k
		break
	}

	// Use the registered parser for the event type
	parser, exists := eventParsers[eventType]
	if !exists {
		return nil, NewError(ErrCodeInvalidEventType, fmt.Sprintf("unknown event type: %s", eventType), "type")
	}

	return parser(data)
}
