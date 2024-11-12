package caep

import (
	"encoding/json"
	"fmt"

	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// CredentialType represents the type of credential
type CredentialType string

const (
	CredentialTypePassword      CredentialType = "password"
	CredentialTypePin           CredentialType = "pin"
	CredentialTypeX509          CredentialType = "x509"
	CredentialTypeFIDO2Platform CredentialType = "fido2-platform"
	CredentialTypeFIDO2Roaming  CredentialType = "fido2-roaming"
	CredentialTypeFIDOU2F       CredentialType = "fido-u2f"
	CredentialTypeVerifiable    CredentialType = "verifiable-credential"
	CredentialTypePhoneVoice    CredentialType = "phone-voice"
	CredentialTypePhoneSMS      CredentialType = "phone-sms"
	CredentialTypeApp           CredentialType = "app"
)

// ChangeType represents the type of credential change
type ChangeType string

const (
	ChangeTypeCreate ChangeType = "create"
	ChangeTypeRevoke ChangeType = "revoke"
	ChangeTypeUpdate ChangeType = "update"
	ChangeTypeDelete ChangeType = "delete"
)

// CredentialChangeEvent represents a credential change event
type CredentialChangeEvent struct {
	BaseEvent
	CredentialType CredentialType `json:"credential_type"`         // REQUIRED
	ChangeType     ChangeType     `json:"change_type"`             // REQUIRED
	FriendlyName   *string        `json:"friendly_name,omitempty"` // OPTIONAL
	X509Issuer     *string        `json:"x509_issuer,omitempty"`   // OPTIONAL
	X509Serial     *string        `json:"x509_serial,omitempty"`   // OPTIONAL
	FIDO2AAGUID    *string        `json:"fido2_aaguid,omitempty"`  // OPTIONAL
}

// NewCredentialChangeEvent creates a new credential change event
func NewCredentialChangeEvent(credType CredentialType, changeType ChangeType) *CredentialChangeEvent {
	e := &CredentialChangeEvent{
		CredentialType: credType,
		ChangeType:     changeType,
	}

	e.SetType(EventTypeCredentialChange)

	return e
}

// WithFriendlyName sets the friendly name
func (e *CredentialChangeEvent) WithFriendlyName(name string) *CredentialChangeEvent {
	e.FriendlyName = &name

	return e
}

// WithX509Details sets the X.509 certificate details
func (e *CredentialChangeEvent) WithX509Details(issuer, serial string) *CredentialChangeEvent {
	e.X509Issuer = &issuer
	e.X509Serial = &serial

	return e
}

// WithFIDO2AAGUID sets the FIDO2 Authenticator Attestation GUID
func (e *CredentialChangeEvent) WithFIDO2AAGUID(aaguid string) *CredentialChangeEvent {
	e.FIDO2AAGUID = &aaguid

	return e
}

// Validate ensures the event is valid
func (e *CredentialChangeEvent) Validate() error {
	// Validate metadata
	if err := e.ValidateMetadata(); err != nil {
		return err
	}

	// Validate required fields
	if !IsValidCredentialType(e.CredentialType) {
		return event.NewError(event.ErrCodeInvalidValue,
			fmt.Sprintf("invalid credential type: %s", e.CredentialType),
			"credential_type")
	}

	if !IsValidChangeType(e.ChangeType) {
		return event.NewError(event.ErrCodeInvalidValue,
			fmt.Sprintf("invalid change type: %s", e.ChangeType),
			"change_type")
	}

	// Validate X.509 details consistency
	if (e.X509Issuer != nil && e.X509Serial == nil) ||
		(e.X509Issuer == nil && e.X509Serial != nil) {
		return event.NewError(event.ErrCodeInvalidValue,
			"both x509_issuer and x509_serial must be provided together",
			"x509")
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (e *CredentialChangeEvent) MarshalJSON() ([]byte, error) {
	// Create the event payload
	payload := map[string]interface{}{
		"credential_type": e.CredentialType,
		"change_type":     e.ChangeType,
	}

	if e.FriendlyName != nil {
		payload["friendly_name"] = *e.FriendlyName
	}

	if e.X509Issuer != nil {
		payload["x509_issuer"] = *e.X509Issuer
	}

	if e.X509Serial != nil {
		payload["x509_serial"] = *e.X509Serial
	}

	if e.FIDO2AAGUID != nil {
		payload["fido2_aaguid"] = *e.FIDO2AAGUID
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
func (e *CredentialChangeEvent) UnmarshalJSON(data []byte) error {
	var eventMap map[event.EventType]json.RawMessage

	if err := json.Unmarshal(data, &eventMap); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to unmarshal event wrapper", "")
	}

	eventData, ok := eventMap[EventTypeCredentialChange]
	if !ok {
		return event.NewError(event.ErrCodeInvalidEventType,
			"credential change event not found", "events")
	}

	var payload struct {
		CredentialType CredentialType `json:"credential_type"`
		ChangeType     ChangeType     `json:"change_type"`
		FriendlyName   *string        `json:"friendly_name,omitempty"`
		X509Issuer     *string        `json:"x509_issuer,omitempty"`
		X509Serial     *string        `json:"x509_serial,omitempty"`
		FIDO2AAGUID    *string        `json:"fido2_aaguid,omitempty"`
		Metadata       *EventMetadata `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(eventData, &payload); err != nil {
		return event.NewError(event.ErrCodeParseError,
			"failed to parse credential change event data", "")
	}

	e.SetType(EventTypeCredentialChange)

	e.CredentialType = payload.CredentialType
	e.ChangeType = payload.ChangeType
	e.FriendlyName = payload.FriendlyName
	e.X509Issuer = payload.X509Issuer
	e.X509Serial = payload.X509Serial
	e.FIDO2AAGUID = payload.FIDO2AAGUID
	e.Metadata = payload.Metadata

	if err := e.Validate(); err != nil {
		return err
	}

	return nil
}

// WithEventTimestamp sets the event timestamp
func (e *CredentialChangeEvent) WithEventTimestamp(timestamp int64) *CredentialChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithEventTimestamp(timestamp)

	return e
}

// WithInitiatingEntity sets who/what initiated the event
func (e *CredentialChangeEvent) WithInitiatingEntity(entity InitiatingEntity) *CredentialChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithInitiatingEntity(entity)

	return e
}

// WithReasonAdmin adds an admin reason
func (e *CredentialChangeEvent) WithReasonAdmin(language, reason string) *CredentialChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonAdmin(language, reason)

	return e
}

// WithReasonUser adds a user reason
func (e *CredentialChangeEvent) WithReasonUser(language, reason string) *CredentialChangeEvent {
	if e.Metadata == nil {
		e.Metadata = NewEventMetadata()
	}

	e.Metadata.WithReasonUser(language, reason)

	return e
}

// IsValidCredentialType checks if the provided credential type is valid
func IsValidCredentialType(ct CredentialType) bool {
	switch ct {
	case CredentialTypePassword, CredentialTypePin, CredentialTypeX509,
		CredentialTypeFIDO2Platform, CredentialTypeFIDO2Roaming,
		CredentialTypeFIDOU2F, CredentialTypeVerifiable,
		CredentialTypePhoneVoice, CredentialTypePhoneSMS,
		CredentialTypeApp:
		return true
	default:
		return false
	}
}

// IsValidChangeType checks if the provided change type is valid
func IsValidChangeType(ct ChangeType) bool {
	switch ct {
	case ChangeTypeCreate, ChangeTypeRevoke,
		ChangeTypeUpdate, ChangeTypeDelete:
		return true
	default:
		return false
	}
}

func parseCredentialChangeEvent(data json.RawMessage) (event.Event, error) {
	var e CredentialChangeEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, event.NewError(event.ErrCodeParseError,
			"failed to parse credential change event", "")
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	return &e, nil
}

func init() {
	event.RegisterEventParser(EventTypeCredentialChange, parseCredentialChangeEvent)
}
