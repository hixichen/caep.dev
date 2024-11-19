package builder

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sgnl-ai/caep.dev-receiver/secevent/pkg/event"
	"github.com/sgnl-ai/caep.dev-receiver/secevent/pkg/subject"
)

// SecEvent represents a base Security Event Token that can contain multiple events
type SecEvent struct {
	jwt.RegisteredClaims

	// SecEvent-specific Claims
	Events  map[event.EventType]event.Event `json:"events"` // REQUIRED
	Subject subject.Subject                 `json:"sub_id"` // REQUIRED

	// Optional Claims
	TransactionID *string `json:"txn,omitempty"` // OPTIONAL
}

func newSecEvent() *SecEvent {
	return &SecEvent{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
		Events: make(map[event.EventType]event.Event),
	}
}

func (s *SecEvent) Valid() error {
	return s.Validate()
}

func (s *SecEvent) Validate() error {
	if s.Issuer == "" {
		return fmt.Errorf("issuer (iss) claim is required")
	}

	if s.ID == "" {
		return fmt.Errorf("JWT ID (jti) claim is required")
	}

	if len(s.Events) == 0 {
		return fmt.Errorf("at least one event is required")
	}

	if s.Subject == nil {
		return fmt.Errorf("subject is required")
	}

	for _, evt := range s.Events {
		if err := evt.Validate(); err != nil {
			return fmt.Errorf("invalid event: %w", err)
		}
	}

	if err := s.Subject.Validate(); err != nil {
		return fmt.Errorf("invalid subject: %w", err)
	}

	return nil
}

func (s *SecEvent) WithIssuer(issuer string) *SecEvent {
	s.Issuer = issuer

	return s
}

func (s *SecEvent) WithID(id string) *SecEvent {
	s.ID = id

	return s
}

func (s *SecEvent) WithAudience(audience ...string) *SecEvent {
	s.Audience = audience

	return s
}

func (s *SecEvent) WithSubject(sub subject.Subject) *SecEvent {
	s.Subject = sub

	return s
}

func (s *SecEvent) WithEvent(evt event.Event) *SecEvent {
	s.Events[evt.Type()] = evt

	return s
}

func (s *SecEvent) WithTransactionID(txn string) *SecEvent {
	s.TransactionID = &txn

	return s
}

func (s *SecEvent) GetExpirationTime() (*jwt.NumericDate, error) {
	return nil, nil // SecEvent doesn't use expiration time
}

func (s *SecEvent) GetIssuedAt() (*jwt.NumericDate, error) {
	return s.IssuedAt, nil
}

func (s *SecEvent) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil // SecEvent doesn't use not before
}

func (s *SecEvent) GetIssuer() (string, error) {
	return s.Issuer, nil
}

func (s *SecEvent) GetSubject() (string, error) {
	return "", nil // SecEvent doesn't use the standard sub claim
}

func (s *SecEvent) GetAudience() (jwt.ClaimStrings, error) {
	return s.Audience, nil
}

func (s *SecEvent) UnmarshalJSON(data []byte) error {
	type Alias SecEvent

	aux := &struct {
		*Alias
		Events  map[event.EventType]json.RawMessage `json:"events"`
		Subject json.RawMessage                     `json:"sub_id"`
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if len(aux.Subject) > 0 {
		parsedSubject, err := subject.ParseSubject(aux.Subject)
		if err != nil {
			return fmt.Errorf("failed to parse subject: %w", err)
		}

		s.Subject = parsedSubject
	}

	if s.Events == nil {
		s.Events = make(map[event.EventType]event.Event)
	}

	for eventType, eventData := range aux.Events {
		parsedEvent, err := event.ParseEvent(eventType, eventData)
		if err != nil {
			return fmt.Errorf("failed to parse event of type %s: %w", eventType, err)
		}

		s.Events[eventType] = parsedEvent
	}

	return s.Validate()
}

type SingleEventSecEvent struct {
	jwt.RegisteredClaims

	// SecEvent-specific Claims
	Event   event.Event     `json:"-"`      // Will be marshaled in events field
	Subject subject.Subject `json:"sub_id"` // REQUIRED

	// Optional Claims
	TransactionID *string `json:"txn,omitempty"` // OPTIONAL
}

func newSingleEventSecEvent() *SingleEventSecEvent {
	return &SingleEventSecEvent{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
}

func (s *SingleEventSecEvent) Valid() error {
	return s.Validate()
}

func (s *SingleEventSecEvent) Validate() error {
	if s.Issuer == "" {
		return fmt.Errorf("issuer (iss) claim is required")
	}

	if s.ID == "" {
		return fmt.Errorf("JWT ID (jti) claim is required")
	}

	if s.Event == nil {
		return fmt.Errorf("event is required")
	}

	if s.Subject == nil {
		return fmt.Errorf("subject is required")
	}

	if err := s.Event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	if err := s.Subject.Validate(); err != nil {
		return fmt.Errorf("invalid subject: %w", err)
	}

	return nil
}

func (s *SingleEventSecEvent) WithIssuer(issuer string) *SingleEventSecEvent {
	s.Issuer = issuer

	return s
}

func (s *SingleEventSecEvent) WithID(id string) *SingleEventSecEvent {
	s.ID = id

	return s
}

func (s *SingleEventSecEvent) WithAudience(audience ...string) *SingleEventSecEvent {
	s.Audience = audience

	return s
}

func (s *SingleEventSecEvent) WithSubject(sub subject.Subject) *SingleEventSecEvent {
	s.Subject = sub

	return s
}

func (s *SingleEventSecEvent) WithEvent(evt event.Event) *SingleEventSecEvent {
	s.Event = evt

	return s
}

func (s *SingleEventSecEvent) WithTransactionID(txn string) *SingleEventSecEvent {
	s.TransactionID = &txn

	return s
}

func (s *SingleEventSecEvent) GetExpirationTime() (*jwt.NumericDate, error) {
	return nil, nil // SecEvent doesn't use expiration time
}

func (s *SingleEventSecEvent) GetIssuedAt() (*jwt.NumericDate, error) {
	return s.IssuedAt, nil
}

func (s *SingleEventSecEvent) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil // SecEvent doesn't use not before
}

func (s *SingleEventSecEvent) GetIssuer() (string, error) {
	return s.Issuer, nil
}

func (s *SingleEventSecEvent) GetSubject() (string, error) {
	return "", nil // SecEvent doesn't use the standard sub claim
}

func (s *SingleEventSecEvent) GetAudience() (jwt.ClaimStrings, error) {
	return s.Audience, nil
}

func (s *SingleEventSecEvent) MarshalJSON() ([]byte, error) {
	type Alias SingleEventSecEvent

	temp := struct {
		*Alias
		Events map[event.EventType]interface{} `json:"events"`
	}{
		Alias:  (*Alias)(s),
		Events: make(map[event.EventType]interface{}),
	}

	if s.Event != nil {
		temp.Events[s.Event.Type()] = s.Event.Payload()
	}

	return json.Marshal(temp)
}

func (s *SingleEventSecEvent) UnmarshalJSON(data []byte) error {
	type Alias SingleEventSecEvent

	aux := &struct {
		*Alias
		Events  map[event.EventType]json.RawMessage `json:"events"`
		Subject json.RawMessage                     `json:"sub_id"`
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if len(aux.Subject) > 0 {
		parsedSubject, err := subject.ParseSubject(aux.Subject)
		if err != nil {
			return fmt.Errorf("failed to parse subject: %w", err)
		}

		s.Subject = parsedSubject
	}

	if len(aux.Events) != 1 {
		return fmt.Errorf("exactly one event must be present in a single-event SecEvent")
	}

	var eventType event.EventType
	var eventData json.RawMessage
	for t, d := range aux.Events {
		eventType = t
		eventData = d
	}

	parsedEvent, err := event.ParseEvent(eventType, eventData)
	if err != nil {
		return err
	}

	s.Event = parsedEvent

	return s.Validate()
}
