package builder

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/subject"
)

// SET represents a Security Event Token
type SET struct {
	jwt.RegisteredClaims

	// SET-specific Claims
	Event   event.Event     `json:"events"` // REQUIRED
	Subject subject.Subject `json:"sub_id"` // REQUIRED

	// Optional Claims
	TransactionID *string `json:"txn,omitempty"` // OPTIONAL
}

// NewSET creates a new SET with default values
func NewSET() *SET {
	return &SET{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
}

// Implement the Valid() method required by jwt.Claims
func (s *SET) Valid() error {
	return s.Validate()
}

// Validate checks if the SET is valid
func (s *SET) Validate() error {
	now := time.Now()

	// Validate expiration time (exp)
	if s.ExpiresAt != nil && now.After(s.ExpiresAt.Time) {
		return fmt.Errorf("token is expired")
	}

	// Validate not before (nbf)
	if s.NotBefore != nil && now.Before(s.NotBefore.Time) {
		return fmt.Errorf("token is not valid yet")
	}

	// Validate issued at (iat)
	if s.IssuedAt != nil && now.Before(s.IssuedAt.Time) {
		return fmt.Errorf("token used before issued")
	}

	// Perform additional custom validations
	if s.Issuer == "" {
		return fmt.Errorf("issuer (iss) claim is required")
	}

	if s.ID == "" {
		return fmt.Errorf("JWT ID (jti) claim is required")
	}

	if s.Event == nil {
		return fmt.Errorf("event is required")
	}

	if err := s.Event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	if s.Subject == nil {
		return fmt.Errorf("subject is required")
	}

	if err := s.Subject.Validate(); err != nil {
		return fmt.Errorf("invalid subject: %w", err)
	}

	return nil
}

// UnmarshalJSON implements custom JSON unmarshalling to handle the events map
func (s *SET) UnmarshalJSON(data []byte) error {
	// Create an anonymous struct to match the JSON structure
	aux := struct {
		jwt.RegisteredClaims
		Event         json.RawMessage `json:"events"`
		Subject       json.RawMessage `json:"sub_id"`
		TransactionID *string         `json:"txn,omitempty"`
	}{}

	// Unmarshal the data into the auxiliary struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set the registered claims and transaction ID
	s.RegisteredClaims = aux.RegisteredClaims
	s.TransactionID = aux.TransactionID

	// Parse the Event using the ParseEvent method
	event, err := event.ParseEvent(aux.Event)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	s.Event = event

	// Parse the Subject using the ParseSubject method
	subj, err := subject.ParseSubject(aux.Subject)
	if err != nil {
		return fmt.Errorf("failed to parse subject: %w", err)
	}

	s.Subject = subj

	return nil
}

// WithIssuer sets the issuer
func (s *SET) WithIssuer(issuer string) *SET {
	s.Issuer = issuer

	return s
}

// WithIssuedAt sets the issuance time
func (s *SET) WithIssuedAt(iat int64) *SET {
	s.IssuedAt = jwt.NewNumericDate(time.Unix(iat, 0))

	return s
}

// WithID sets the JWT ID
func (s *SET) WithID(id string) *SET {
	s.ID = id

	return s
}

// WithAudience sets the audience
func (s *SET) WithAudience(audience ...string) *SET {
	s.Audience = jwt.ClaimStrings(audience)

	return s
}

// WithSubject sets the subject
func (s *SET) WithSubject(sub subject.Subject) *SET {
	s.Subject = sub

	return s
}

// WithEvent adds an event
func (s *SET) WithEvent(evt event.Event) *SET {
	s.Event = evt

	return s
}

// WithTransactionID sets the transaction ID
func (s *SET) WithTransactionID(txn string) *SET {
	s.TransactionID = &txn

	return s
}
