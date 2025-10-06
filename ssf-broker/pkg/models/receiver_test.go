package models

import (
	"strings"
	"testing"
	"time"
)

func TestReceiver_Validate(t *testing.T) {
	tests := []struct {
		name    string
		receiver *Receiver
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid webhook receiver",
			receiver: &Receiver{
				ID:         "test-receiver",
				WebhookURL: "https://example.com/webhook",
				EventTypes: []string{EventTypeSessionRevoked},
				Delivery: DeliveryConfig{
					Method: DeliveryMethodWebhook,
				},
				Auth: AuthConfig{
					Type: AuthTypeBearer,
					Token: "test-token",
				},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			receiver: &Receiver{
				WebhookURL: "https://example.com/webhook",
				EventTypes: []string{EventTypeSessionRevoked},
				Delivery: DeliveryConfig{
					Method: DeliveryMethodWebhook,
				},
			},
			wantErr: true,
			errMsg:  "receiver ID is required",
		},
		{
			name: "missing event types",
			receiver: &Receiver{
				ID:         "test-receiver",
				WebhookURL: "https://example.com/webhook",
				EventTypes: []string{},
				Delivery: DeliveryConfig{
					Method: DeliveryMethodWebhook,
				},
			},
			wantErr: true,
			errMsg:  "at least one event type must be specified",
		},
		{
			name: "webhook without URL",
			receiver: &Receiver{
				ID:         "test-receiver",
				EventTypes: []string{EventTypeSessionRevoked},
				Delivery: DeliveryConfig{
					Method: DeliveryMethodWebhook,
				},
			},
			wantErr: true,
			errMsg:  "webhook_url is required for webhook delivery",
		},
		{
			name: "invalid webhook URL",
			receiver: &Receiver{
				ID:         "test-receiver",
				WebhookURL: "://invalid-url",
				EventTypes: []string{EventTypeSessionRevoked},
				Delivery: DeliveryConfig{
					Method: DeliveryMethodWebhook,
				},
			},
			wantErr: true,
			errMsg:  "invalid webhook_url",
		},
		{
			name: "pull delivery without topic",
			receiver: &Receiver{
				ID:         "test-receiver",
				EventTypes: []string{EventTypeSessionRevoked},
				Delivery: DeliveryConfig{
					Method: DeliveryMethodPull,
				},
			},
			wantErr: true,
			errMsg:  "topic_name is required for pull delivery",
		},
		{
			name: "valid pull receiver",
			receiver: &Receiver{
				ID:         "test-receiver",
				EventTypes: []string{EventTypeSessionRevoked},
				Delivery: DeliveryConfig{
					Method:    DeliveryMethodPull,
					TopicName: "test-topic",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.receiver.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Receiver.Validate() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Receiver.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Receiver.Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		auth    *AuthConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid bearer auth",
			auth: &AuthConfig{
				Type:  AuthTypeBearer,
				Token: "test-token",
			},
			wantErr: false,
		},
		{
			name: "bearer auth without token",
			auth: &AuthConfig{
				Type: AuthTypeBearer,
			},
			wantErr: true,
			errMsg:  "token is required for bearer auth",
		},
		{
			name: "valid oauth2 auth",
			auth: &AuthConfig{
				Type:         AuthTypeOAuth2,
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				TokenURL:     "https://example.com/token",
			},
			wantErr: false,
		},
		{
			name: "oauth2 auth without client_id",
			auth: &AuthConfig{
				Type:         AuthTypeOAuth2,
				ClientSecret: "client-secret",
				TokenURL:     "https://example.com/token",
			},
			wantErr: true,
			errMsg:  "client_id, client_secret, and token_url are required for oauth2 auth",
		},
		{
			name: "valid hmac auth",
			auth: &AuthConfig{
				Type:   AuthTypeHMAC,
				Secret: "hmac-secret",
			},
			wantErr: false,
		},
		{
			name: "hmac auth without secret",
			auth: &AuthConfig{
				Type: AuthTypeHMAC,
			},
			wantErr: true,
			errMsg:  "secret is required for hmac auth",
		},
		{
			name: "none auth type",
			auth: &AuthConfig{
				Type: AuthTypeNone,
			},
			wantErr: false,
		},
		{
			name: "invalid auth type",
			auth: &AuthConfig{
				Type: AuthType("invalid"),
			},
			wantErr: true,
			errMsg:  "invalid auth type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.auth.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("AuthConfig.Validate() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("AuthConfig.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("AuthConfig.Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestReceiver_SetDefaults(t *testing.T) {
	receiver := &Receiver{
		ID:         "test-receiver",
		EventTypes: []string{EventTypeSessionRevoked},
	}

	receiver.SetDefaults()

	// Check defaults
	if receiver.Status != ReceiverStatusActive {
		t.Errorf("Expected status to be %v, got %v", ReceiverStatusActive, receiver.Status)
	}

	if receiver.Delivery.Method != DeliveryMethodWebhook {
		t.Errorf("Expected delivery method to be %v, got %v", DeliveryMethodWebhook, receiver.Delivery.Method)
	}

	if receiver.Delivery.BatchSize != 1 {
		t.Errorf("Expected batch size to be 1, got %d", receiver.Delivery.BatchSize)
	}

	if receiver.Delivery.Timeout != 30*time.Second {
		t.Errorf("Expected timeout to be 30s, got %v", receiver.Delivery.Timeout)
	}

	if receiver.Auth.Type != AuthTypeNone {
		t.Errorf("Expected auth type to be %v, got %v", AuthTypeNone, receiver.Auth.Type)
	}

	if receiver.Retry.MaxRetries != 3 {
		t.Errorf("Expected max retries to be 3, got %d", receiver.Retry.MaxRetries)
	}

	if receiver.Retry.InitialInterval != 1*time.Second {
		t.Errorf("Expected initial interval to be 1s, got %v", receiver.Retry.InitialInterval)
	}

	if receiver.Retry.MaxInterval != 60*time.Second {
		t.Errorf("Expected max interval to be 60s, got %v", receiver.Retry.MaxInterval)
	}

	if receiver.Retry.Multiplier != 2.0 {
		t.Errorf("Expected multiplier to be 2.0, got %f", receiver.Retry.Multiplier)
	}

	if receiver.Metadata.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}

	if receiver.Metadata.UpdatedAt.IsZero() {
		t.Error("Expected updated_at to be set")
	}

	if receiver.Metadata.Tags == nil {
		t.Error("Expected tags to be initialized")
	}
}

func TestReceiverRequest_ToReceiver(t *testing.T) {
	req := &ReceiverRequest{
		ID:          "test-receiver",
		Name:        "Test Receiver",
		Description: "A test receiver",
		WebhookURL:  "https://example.com/webhook",
		EventTypes:  []string{EventTypeSessionRevoked, EventTypeCredentialChange},
		Delivery: DeliveryConfig{
			Method: DeliveryMethodWebhook,
		},
		Auth: AuthConfig{
			Type:  AuthTypeBearer,
			Token: "test-token",
		},
		Tags: map[string]string{
			"environment": "test",
		},
	}

	receiver := req.ToReceiver()

	if receiver.ID != req.ID {
		t.Errorf("Expected ID %v, got %v", req.ID, receiver.ID)
	}

	if receiver.Name != req.Name {
		t.Errorf("Expected Name %v, got %v", req.Name, receiver.Name)
	}

	if receiver.Description != req.Description {
		t.Errorf("Expected Description %v, got %v", req.Description, receiver.Description)
	}

	if receiver.WebhookURL != req.WebhookURL {
		t.Errorf("Expected WebhookURL %v, got %v", req.WebhookURL, receiver.WebhookURL)
	}

	if len(receiver.EventTypes) != len(req.EventTypes) {
		t.Errorf("Expected %d event types, got %d", len(req.EventTypes), len(receiver.EventTypes))
	}

	if receiver.Delivery.Method != req.Delivery.Method {
		t.Errorf("Expected delivery method %v, got %v", req.Delivery.Method, receiver.Delivery.Method)
	}

	if receiver.Auth.Type != req.Auth.Type {
		t.Errorf("Expected auth type %v, got %v", req.Auth.Type, receiver.Auth.Type)
	}

	if receiver.Auth.Token != req.Auth.Token {
		t.Errorf("Expected auth token %v, got %v", req.Auth.Token, receiver.Auth.Token)
	}

	if receiver.Metadata.Tags["environment"] != "test" {
		t.Errorf("Expected environment tag to be 'test', got %v", receiver.Metadata.Tags["environment"])
	}

	// Check that defaults were set
	if receiver.Status != ReceiverStatusActive {
		t.Errorf("Expected status to be set to %v", ReceiverStatusActive)
	}
}