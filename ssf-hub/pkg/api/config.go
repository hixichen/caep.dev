package api

import (
	"fmt"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server" json:"server"`
	PubSub  PubSubConfig  `yaml:"pubsub" json:"pubsub"`
	Auth    AuthConfig    `yaml:"auth" json:"auth"`
	Logging LoggingConfig `yaml:"logging" json:"logging"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
}

// PubSubConfig contains Google Cloud Pub/Sub configuration
type PubSubConfig struct {
	ProjectID              string `yaml:"project_id" json:"project_id"`
	TopicPrefix            string `yaml:"topic_prefix" json:"topic_prefix"`
	CredentialsFile        string `yaml:"credentials_file,omitempty" json:"credentials_file,omitempty"`
	MaxConcurrentHandlers  int    `yaml:"max_concurrent_handlers,omitempty" json:"max_concurrent_handlers,omitempty"`
	MaxOutstandingMessages int    `yaml:"max_outstanding_messages,omitempty" json:"max_outstanding_messages,omitempty"`
	MaxOutstandingBytes    int    `yaml:"max_outstanding_bytes,omitempty" json:"max_outstanding_bytes,omitempty"`
	EnableMessageOrdering  bool   `yaml:"enable_message_ordering,omitempty" json:"enable_message_ordering,omitempty"`
	AckDeadline            int    `yaml:"ack_deadline,omitempty" json:"ack_deadline,omitempty"`             // seconds
	RetentionDuration      int    `yaml:"retention_duration,omitempty" json:"retention_duration,omitempty"` // hours
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	JWTSecret       string   `yaml:"jwt_secret" json:"jwt_secret"`
	TokenExpiration int      `yaml:"token_expiration,omitempty" json:"token_expiration,omitempty"` // hours
	RequireAuth     bool     `yaml:"require_auth,omitempty" json:"require_auth,omitempty"`
	AllowedIssuers  []string `yaml:"allowed_issuers,omitempty" json:"allowed_issuers,omitempty"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`   // debug, info, warn, error
	Format string `yaml:"format" json:"format"` // json, text
}

// SetDefaults sets default values for configuration
func (c *Config) SetDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 120 * time.Second
	}

	if c.PubSub.TopicPrefix == "" {
		c.PubSub.TopicPrefix = "ssf-events"
	}
	if c.PubSub.MaxConcurrentHandlers == 0 {
		c.PubSub.MaxConcurrentHandlers = 10
	}
	if c.PubSub.MaxOutstandingMessages == 0 {
		c.PubSub.MaxOutstandingMessages = 1000
	}
	if c.PubSub.MaxOutstandingBytes == 0 {
		c.PubSub.MaxOutstandingBytes = 1000000000 // 1GB
	}
	if c.PubSub.AckDeadline == 0 {
		c.PubSub.AckDeadline = 60 // seconds
	}
	if c.PubSub.RetentionDuration == 0 {
		c.PubSub.RetentionDuration = 168 // 7 days
	}

	if c.Auth.TokenExpiration == 0 {
		c.Auth.TokenExpiration = 24 // hours
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.PubSub.ProjectID == "" {
		return &ValidationError{Field: "pubsub.project_id", Message: "project ID is required"}
	}

	return nil
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}
