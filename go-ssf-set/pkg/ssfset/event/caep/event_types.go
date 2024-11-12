package caep

import (
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// CAEP Event Types
const (
	EventTypeTokenClaimsChange      event.EventType = "https://schemas.openid.net/secevent/caep/event-type/token-claims-change"
	EventTypeSessionRevoked         event.EventType = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	EventTypeCredentialChange       event.EventType = "https://schemas.openid.net/secevent/caep/event-type/credential-change"
	EventTypeAssuranceLevelChange   event.EventType = "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
	EventTypeDeviceComplianceChange event.EventType = "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change"
)
