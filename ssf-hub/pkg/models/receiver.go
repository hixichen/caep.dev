package models

import (
	"fmt"
	"net/url"
	"time"
)

// Receiver represents a registered event receiver
type Receiver struct {
	ID          string           `json:"id" yaml:"id"`
	Name        string           `json:"name,omitempty" yaml:"name,omitempty"`
	Description string           `json:"description,omitempty" yaml:"description,omitempty"`
	WebhookURL  string           `json:"webhook_url,omitempty" yaml:"webhook_url,omitempty"`
	EventTypes  []string         `json:"event_types" yaml:"event_types"`
	Delivery    DeliveryConfig   `json:"delivery" yaml:"delivery"`
	Auth        AuthConfig       `json:"auth,omitempty" yaml:"auth,omitempty"`
	Filters     []EventFilter    `json:"filters,omitempty" yaml:"filters,omitempty"`
	Retry       RetryConfig      `json:"retry,omitempty" yaml:"retry,omitempty"`
	Status      ReceiverStatus   `json:"status" yaml:"status"`
	Metadata    ReceiverMetadata `json:"metadata" yaml:"metadata"`
}

// DeliveryConfig specifies how events should be delivered to the receiver
type DeliveryConfig struct {
	Method      DeliveryMethod `json:"method" yaml:"method"`                             // webhook, pull, push
	TopicName   string         `json:"topic_name,omitempty" yaml:"topic_name,omitempty"` // For pull delivery
	Subscription string        `json:"subscription,omitempty" yaml:"subscription,omitempty"` // For pull delivery
	BatchSize   int            `json:"batch_size,omitempty" yaml:"batch_size,omitempty"`
	Timeout     time.Duration  `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// AuthConfig specifies authentication for webhook delivery
type AuthConfig struct {
	Type         AuthType `json:"type" yaml:"type"`                                   // bearer, oauth2, hmac, none
	Token        string   `json:"token,omitempty" yaml:"token,omitempty"`             // Bearer token
	ClientID     string   `json:"client_id,omitempty" yaml:"client_id,omitempty"`     // OAuth2
	ClientSecret string   `json:"client_secret,omitempty" yaml:"client_secret,omitempty"` // OAuth2
	TokenURL     string   `json:"token_url,omitempty" yaml:"token_url,omitempty"`     // OAuth2
	Scopes       []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`           // OAuth2
	Secret       string   `json:"secret,omitempty" yaml:"secret,omitempty"`           // HMAC
	Algorithm    string   `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`     // HMAC (sha256, sha512)
}

// EventFilter allows receivers to filter events
type EventFilter struct {
	Field    string      `json:"field" yaml:"field"`       // e.g., "issuer", "subject.email", "event_type"
	Operator FilterOp    `json:"operator" yaml:"operator"` // equals, contains, matches, in
	Value    interface{} `json:"value" yaml:"value"`       // Filter value
}

// RetryConfig specifies retry behavior for failed deliveries
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
	InitialInterval time.Duration `json:"initial_interval,omitempty" yaml:"initial_interval,omitempty"`
	MaxInterval     time.Duration `json:"max_interval,omitempty" yaml:"max_interval,omitempty"`
	Multiplier      float64       `json:"multiplier,omitempty" yaml:"multiplier,omitempty"`
	EnableJitter    bool          `json:"enable_jitter,omitempty" yaml:"enable_jitter,omitempty"`
}

// ReceiverMetadata contains metadata about the receiver
type ReceiverMetadata struct {
	CreatedAt         time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" yaml:"updated_at"`
	LastEventAt       time.Time `json:"last_event_at,omitempty" yaml:"last_event_at,omitempty"`
	EventsReceived    int64     `json:"events_received" yaml:"events_received"`
	EventsDelivered   int64     `json:"events_delivered" yaml:"events_delivered"`
	EventsFailed      int64     `json:"events_failed" yaml:"events_failed"`
	LastDeliveryError string    `json:"last_delivery_error,omitempty" yaml:"last_delivery_error,omitempty"`
	Tags              map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Enums
type DeliveryMethod string

const (
	DeliveryMethodWebhook DeliveryMethod = "webhook"
	DeliveryMethodPull    DeliveryMethod = "pull"
	DeliveryMethodPush    DeliveryMethod = "push"
)

type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeBearer AuthType = "bearer"
	AuthTypeOAuth2 AuthType = "oauth2"
	AuthTypeHMAC   AuthType = "hmac"
)

type ReceiverStatus string

const (
	ReceiverStatusActive   ReceiverStatus = "active"
	ReceiverStatusInactive ReceiverStatus = "inactive"
	ReceiverStatusError    ReceiverStatus = "error"
	ReceiverStatusPaused   ReceiverStatus = "paused"
)

type FilterOp string

const (
	FilterOpEquals   FilterOp = "equals"
	FilterOpContains FilterOp = "contains"
	FilterOpMatches  FilterOp = "matches"
	FilterOpIn       FilterOp = "in"
	FilterOpExists   FilterOp = "exists"
)

// ReceiverRequest represents a request to register or update a receiver
type ReceiverRequest struct {
	ID          string         `json:"id"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	WebhookURL  string         `json:"webhook_url,omitempty"`
	EventTypes  []string       `json:"event_types"`
	Delivery    DeliveryConfig `json:"delivery"`
	Auth        AuthConfig     `json:"auth,omitempty"`
	Filters     []EventFilter  `json:"filters,omitempty"`
	Retry       RetryConfig    `json:"retry,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// Validate validates the receiver configuration
func (r *Receiver) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("receiver ID is required")
	}

	if len(r.EventTypes) == 0 {
		return fmt.Errorf("at least one event type must be specified")
	}

	// Validate delivery method
	switch r.Delivery.Method {
	case DeliveryMethodWebhook:
		if r.WebhookURL == "" {
			return fmt.Errorf("webhook_url is required for webhook delivery")
		}
		if _, err := url.Parse(r.WebhookURL); err != nil {
			return fmt.Errorf("invalid webhook_url: %w", err)
		}
	case DeliveryMethodPull:
		if r.Delivery.TopicName == "" {
			return fmt.Errorf("topic_name is required for pull delivery")
		}
	case DeliveryMethodPush:
		if r.Delivery.TopicName == "" {
			return fmt.Errorf("topic_name is required for push delivery")
		}
	default:
		return fmt.Errorf("invalid delivery method: %s", r.Delivery.Method)
	}

	// Validate auth configuration
	if err := r.Auth.Validate(); err != nil {
		return fmt.Errorf("auth validation failed: %w", err)
	}

	return nil
}

// Validate validates the auth configuration
func (a *AuthConfig) Validate() error {
	switch a.Type {
	case AuthTypeNone, "":
		// No validation needed for none or empty auth type
	case AuthTypeBearer:
		if a.Token == "" {
			return fmt.Errorf("token is required for bearer auth")
		}
	case AuthTypeOAuth2:
		if a.ClientID == "" || a.ClientSecret == "" || a.TokenURL == "" {
			return fmt.Errorf("client_id, client_secret, and token_url are required for oauth2 auth")
		}
	case AuthTypeHMAC:
		if a.Secret == "" {
			return fmt.Errorf("secret is required for hmac auth")
		}
		if a.Algorithm == "" {
			a.Algorithm = "sha256" // Default
		}
	default:
		return fmt.Errorf("invalid auth type: %s", a.Type)
	}

	return nil
}

// SetDefaults sets default values for the receiver
func (r *Receiver) SetDefaults() {
	if r.Status == "" {
		r.Status = ReceiverStatusActive
	}

	if r.Delivery.Method == "" {
		r.Delivery.Method = DeliveryMethodWebhook
	}

	if r.Delivery.BatchSize == 0 {
		r.Delivery.BatchSize = 1
	}

	if r.Delivery.Timeout == 0 {
		r.Delivery.Timeout = 30 * time.Second
	}

	if r.Auth.Type == "" {
		r.Auth.Type = AuthTypeNone
	}

	// Set retry defaults
	if r.Retry.MaxRetries == 0 {
		r.Retry.MaxRetries = 3
	}
	if r.Retry.InitialInterval == 0 {
		r.Retry.InitialInterval = 1 * time.Second
	}
	if r.Retry.MaxInterval == 0 {
		r.Retry.MaxInterval = 60 * time.Second
	}
	if r.Retry.Multiplier == 0 {
		r.Retry.Multiplier = 2.0
	}

	// Set metadata defaults
	now := time.Now()
	if r.Metadata.CreatedAt.IsZero() {
		r.Metadata.CreatedAt = now
	}
	r.Metadata.UpdatedAt = now

	if r.Metadata.Tags == nil {
		r.Metadata.Tags = make(map[string]string)
	}
}

// ToReceiver converts a ReceiverRequest to a Receiver
func (req *ReceiverRequest) ToReceiver() *Receiver {
	receiver := &Receiver{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		WebhookURL:  req.WebhookURL,
		EventTypes:  req.EventTypes,
		Delivery:    req.Delivery,
		Auth:        req.Auth,
		Filters:     req.Filters,
		Retry:       req.Retry,
		Metadata: ReceiverMetadata{
			Tags: req.Tags,
		},
	}

	receiver.SetDefaults()
	return receiver
}