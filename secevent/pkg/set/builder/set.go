// pkg/set/builder/set.go
package builder

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sgnl-ai/caep.dev-receiver/secevent/pkg/set/event"
	"github.com/sgnl-ai/caep.dev-receiver/secevent/pkg/subject"
)

// SET represents a base Security Event Token that can contain multiple events
type SET struct {
	jwt.RegisteredClaims

	// SET-specific Claims
	Events  map[event.EventType]event.Event `json:"events"` // REQUIRED
	Subject subject.Subject                 `json:"sub_id"` // REQUIRED

	// Optional Claims
	TransactionID *string `json:"txn,omitempty"` // OPTIONAL
}

func newSET() *SET {
	return &SET{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
		Events: make(map[event.EventType]event.Event),
	}
}

func (s *SET) Valid() error {
	return s.Validate()
}

func (s *SET) Validate() error {
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

func (s *SET) WithIssuer(issuer string) *SET {
	s.Issuer = issuer

	return s
}

func (s *SET) WithID(id string) *SET {
	s.ID = id

	return s
}

func (s *SET) WithAudience(audience ...string) *SET {
	s.Audience = audience

	return s
}

func (s *SET) WithSubject(sub subject.Subject) *SET {
	s.Subject = sub

	return s
}

func (s *SET) WithEvent(evt event.Event) *SET {
	s.Events[evt.Type()] = evt

	return s
}

func (s *SET) WithTransactionID(txn string) *SET {
	s.TransactionID = &txn

	return s
}

func (s *SET) GetExpirationTime() (*jwt.NumericDate, error) {
	return nil, nil // SET doesn't use expiration time
}

func (s *SET) GetIssuedAt() (*jwt.NumericDate, error) {
	return s.IssuedAt, nil
}

func (s *SET) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil // SET doesn't use not before
}

func (s *SET) GetIssuer() (string, error) {
	return s.Issuer, nil
}

func (s *SET) GetSubject() (string, error) {
	return "", nil // SET doesn't use the standard sub claim
}

func (s *SET) GetAudience() (jwt.ClaimStrings, error) {
	return s.Audience, nil
}

func (s *SET) UnmarshalJSON(data []byte) error {
	type Alias SET

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

type SingleEventSET struct {
	jwt.RegisteredClaims

	// SET-specific Claims
	Event   event.Event     `json:"-"`      // Will be marshaled in events field
	Subject subject.Subject `json:"sub_id"` // REQUIRED

	// Optional Claims
	TransactionID *string `json:"txn,omitempty"` // OPTIONAL
}

func newSingleEventSET() *SingleEventSET {
	return &SingleEventSET{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
}

func (s *SingleEventSET) Valid() error {
	return s.Validate()
}

func (s *SingleEventSET) Validate() error {
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

func (s *SingleEventSET) WithIssuer(issuer string) *SingleEventSET {
	s.Issuer = issuer

	return s
}

func (s *SingleEventSET) WithID(id string) *SingleEventSET {
	s.ID = id

	return s
}

func (s *SingleEventSET) WithAudience(audience ...string) *SingleEventSET {
	s.Audience = audience

	return s
}

func (s *SingleEventSET) WithSubject(sub subject.Subject) *SingleEventSET {
	s.Subject = sub

	return s
}

func (s *SingleEventSET) WithEvent(evt event.Event) *SingleEventSET {
	s.Event = evt

	return s
}

func (s *SingleEventSET) WithTransactionID(txn string) *SingleEventSET {
	s.TransactionID = &txn

	return s
}

func (s *SingleEventSET) GetExpirationTime() (*jwt.NumericDate, error) {
	return nil, nil // SET doesn't use expiration time
}

func (s *SingleEventSET) GetIssuedAt() (*jwt.NumericDate, error) {
	return s.IssuedAt, nil
}

func (s *SingleEventSET) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil // SET doesn't use not before
}

func (s *SingleEventSET) GetIssuer() (string, error) {
	return s.Issuer, nil
}

func (s *SingleEventSET) GetSubject() (string, error) {
	return "", nil // SET doesn't use the standard sub claim
}

func (s *SingleEventSET) GetAudience() (jwt.ClaimStrings, error) {
	return s.Audience, nil
}

func (s *SingleEventSET) MarshalJSON() ([]byte, error) {
	type Alias SingleEventSET

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

func (s *SingleEventSET) UnmarshalJSON(data []byte) error {
	type Alias SingleEventSET

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
		return fmt.Errorf("exactly one event must be present in a single-event SET")
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
