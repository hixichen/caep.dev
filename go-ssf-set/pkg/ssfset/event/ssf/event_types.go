package ssf

import (
	"github.com/sgnl-ai/go-ssf-set/pkg/ssfset/event"
)

// SSF Event Types
const (
	EventTypeVerification event.EventType = "https://schemas.openid.net/secevent/ssf/event-type/verification"
	EventTypeStreamUpdate event.EventType = "https://schemas.openid.net/secevent/ssf/event-type/stream-updated"
)
