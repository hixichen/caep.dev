package broker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/sgnl-ai/caep.dev/secevent/pkg/parser"
	"github.com/sgnl-ai/caep.dev/secevent/pkg/token"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// PubSubClient interface for pub/sub operations
type PubSubClient interface {
	PublishEvent(ctx context.Context, event *models.SecurityEvent) error
	CreateReceiverSubscription(ctx context.Context, receiver *models.Receiver) error
	DeleteReceiverSubscription(ctx context.Context, receiverID string, eventTypes []string) error
	Close() error
}

// Broker handles SSF event processing and distribution
type Broker struct {
	pubsubClient PubSubClient
	registry     registry.Registry
	logger       *slog.Logger
	parser       *parser.Parser
}

// New creates a new SSF broker
func New(pubsubClient PubSubClient, registry registry.Registry, logger *slog.Logger) *Broker {
	// Initialize SEC event parser with no verification for demo
	// In production, configure with JWKS URL and expected issuer
	secEventParser := parser.NewParser()

	return &Broker{
		pubsubClient: pubsubClient,
		registry:     registry,
		logger:       logger,
		parser:       secEventParser,
	}
}

// ProcessSecurityEvent processes an incoming security event token
func (b *Broker) ProcessSecurityEvent(ctx context.Context, rawSET string, transmitterID string) error {
	if rawSET == "" {
		return fmt.Errorf("raw SET cannot be empty")
	}
	if transmitterID == "" {
		return fmt.Errorf("transmitter ID cannot be empty")
	}

	// Parse the Security Event Token using secevent library
	// Note: Using ParseSecEventNoVerify for demo. In production, use ParseSecEvent with proper verification
	secEvent, err := b.parser.ParseSecEventNoVerify(rawSET)
	if err != nil {
		return fmt.Errorf("failed to parse security event token: %w", err)
	}

	// Convert to internal event model
	event := b.convertToSecurityEvent(secEvent, rawSET, transmitterID)

	b.logger.Info("Processing security event",
		"event_id", event.ID,
		"event_type", event.Type,
		"transmitter_id", transmitterID,
		"subject_format", event.Subject.Format,
		"subject_identifier", event.Subject.Identifier)

	// Find receivers interested in this event type
	receivers, err := b.registry.FilterReceivers(event)
	if err != nil {
		return fmt.Errorf("failed to get receivers for event type %s: %w", event.Type, err)
	}

	if len(receivers) == 0 {
		b.logger.Warn("No receivers found for event type", "event_type", event.Type)
		return nil
	}

	b.logger.Info("Found receivers for event",
		"event_id", event.ID,
		"receiver_count", len(receivers))

	// Update statistics for interested receivers
	for _, receiver := range receivers {
		if err := b.registry.IncrementEventReceived(receiver.ID); err != nil {
			b.logger.Error("Failed to update receiver stats", "receiver_id", receiver.ID, "error", err)
		}
	}

	// Publish event to Pub/Sub for distribution
	if err := b.pubsubClient.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to publish event to pub/sub: %w", err)
	}

	b.logger.Info("Security event processed successfully",
		"event_id", event.ID,
		"receivers_notified", len(receivers))

	return nil
}

// RegisterReceiver registers a new event receiver
func (b *Broker) RegisterReceiver(ctx context.Context, receiverReq *models.ReceiverRequest) (*models.Receiver, error) {
	// Convert request to receiver
	receiver := receiverReq.ToReceiver()

	// Validate the receiver
	if err := receiver.Validate(); err != nil {
		return nil, fmt.Errorf("receiver validation failed: %w", err)
	}

	// Register with the registry
	if err := b.registry.Register(receiver); err != nil {
		return nil, fmt.Errorf("failed to register receiver: %w", err)
	}

	// Create Pub/Sub subscriptions for the receiver
	if err := b.pubsubClient.CreateReceiverSubscription(ctx, receiver); err != nil {
		// Try to unregister if subscription creation fails
		if unregErr := b.registry.Unregister(receiver.ID); unregErr != nil {
			b.logger.Error("Failed to unregister receiver after subscription creation failure",
				"receiver_id", receiver.ID, "error", unregErr)
		}
		return nil, fmt.Errorf("failed to create pub/sub subscriptions: %w", err)
	}

	b.logger.Info("Receiver registered successfully",
		"receiver_id", receiver.ID,
		"event_types", receiver.EventTypes,
		"delivery_method", receiver.Delivery.Method)

	return receiver, nil
}

// UnregisterReceiver removes a receiver and its subscriptions
func (b *Broker) UnregisterReceiver(ctx context.Context, receiverID string) error {
	// Get receiver details first
	receiver, err := b.registry.Get(receiverID)
	if err != nil {
		return fmt.Errorf("receiver not found: %w", err)
	}

	// Delete Pub/Sub subscriptions
	if err := b.pubsubClient.DeleteReceiverSubscription(ctx, receiverID, receiver.EventTypes); err != nil {
		b.logger.Error("Failed to delete pub/sub subscriptions", "receiver_id", receiverID, "error", err)
		// Continue with unregistration even if subscription deletion fails
	}

	// Unregister from registry
	if err := b.registry.Unregister(receiverID); err != nil {
		return fmt.Errorf("failed to unregister receiver: %w", err)
	}

	b.logger.Info("Receiver unregistered successfully", "receiver_id", receiverID)

	return nil
}

// UpdateReceiver updates an existing receiver configuration
func (b *Broker) UpdateReceiver(ctx context.Context, receiverReq *models.ReceiverRequest) (*models.Receiver, error) {
	// Get existing receiver
	existing, err := b.registry.Get(receiverReq.ID)
	if err != nil {
		return nil, fmt.Errorf("receiver not found: %w", err)
	}

	// Convert request to receiver
	receiver := receiverReq.ToReceiver()

	// Check if event types changed
	eventTypesChanged := !b.slicesEqual(existing.EventTypes, receiver.EventTypes)

	// Update in registry
	if err := b.registry.Update(receiver); err != nil {
		return nil, fmt.Errorf("failed to update receiver: %w", err)
	}

	// Update Pub/Sub subscriptions if event types changed
	if eventTypesChanged {
		// Delete old subscriptions
		if err := b.pubsubClient.DeleteReceiverSubscription(ctx, receiver.ID, existing.EventTypes); err != nil {
			b.logger.Error("Failed to delete old pub/sub subscriptions", "receiver_id", receiver.ID, "error", err)
		}

		// Create new subscriptions
		if err := b.pubsubClient.CreateReceiverSubscription(ctx, receiver); err != nil {
			b.logger.Error("Failed to create new pub/sub subscriptions", "receiver_id", receiver.ID, "error", err)
			return nil, fmt.Errorf("failed to update pub/sub subscriptions: %w", err)
		}
	}

	b.logger.Info("Receiver updated successfully",
		"receiver_id", receiver.ID,
		"event_types_changed", eventTypesChanged)

	return receiver, nil
}

// GetReceiver retrieves a receiver by ID
func (b *Broker) GetReceiver(receiverID string) (*models.Receiver, error) {
	return b.registry.Get(receiverID)
}

// ListReceivers returns all registered receivers
func (b *Broker) ListReceivers() ([]*models.Receiver, error) {
	return b.registry.List()
}

// GetReceiverSubscriptionInfo returns Pub/Sub subscription information for a receiver
func (b *Broker) GetReceiverSubscriptionInfo(ctx context.Context, receiverID string) (map[string]interface{}, error) {
	_, err := b.registry.Get(receiverID)
	if err != nil {
		return nil, fmt.Errorf("receiver not found: %w", err)
	}

	// Return empty subscription info for now - this would be implemented
	// based on specific Pub/Sub client capabilities
	return map[string]interface{}{}, nil
}

// GetBrokerStats returns broker statistics
func (b *Broker) GetBrokerStats() (*BrokerStats, error) {
	receivers, err := b.registry.List()
	if err != nil {
		return nil, fmt.Errorf("failed to get receivers: %w", err)
	}

	stats := &BrokerStats{
		TotalReceivers: len(receivers),
		ReceiversByStatus: b.registry.CountByStatus(),
		EventTypeStats: make(map[string]int),
	}

	// Calculate event type statistics
	for _, receiver := range receivers {
		for _, eventType := range receiver.EventTypes {
			stats.EventTypeStats[eventType]++
		}
	}

	return stats, nil
}

// BrokerStats contains broker statistics
type BrokerStats struct {
	TotalReceivers    int                            `json:"total_receivers"`
	ReceiversByStatus map[models.ReceiverStatus]int  `json:"receivers_by_status"`
	EventTypeStats    map[string]int                 `json:"event_type_stats"`
}


// convertToSecurityEvent converts a parsed SEC event to internal event model
func (b *Broker) convertToSecurityEvent(secEvent *token.SecEvent, rawSET string, transmitterID string) *models.SecurityEvent {
	// Generate a unique processing ID
	processingID := uuid.New().String()

	// Extract subject information
	subjectPayload, err := secEvent.Subject.Payload()
	var subjectFormat, subjectIdentifier string
	subjectClaims := make(map[string]interface{})

	if err == nil {
		// Try to extract format and identifier from subject payload
		if format, ok := subjectPayload["format"].(string); ok {
			subjectFormat = format
		}
		if identifier, ok := subjectPayload["identifier"].(string); ok {
			subjectIdentifier = identifier
		}
		// Store all claims
		subjectClaims = subjectPayload
	}

	event := &models.SecurityEvent{
		ID:          secEvent.ID,
		Type:        string(secEvent.Event.Type()),
		Source:      secEvent.Issuer,
		SpecVersion: "1.0",
		Time:        time.Unix(secEvent.IssuedAt.Unix(), 0),
		Subject: models.Subject{
			Format:     subjectFormat,
			Identifier: subjectIdentifier,
			Claims:     subjectClaims,
		},
		Data:       make(map[string]interface{}),
		Extensions: make(map[string]interface{}),
		Metadata: models.EventMetadata{
			ReceivedAt:    time.Now(),
			ProcessedAt:   time.Now(),
			TransmitterID: transmitterID,
			RawSET:        rawSET,
			ProcessingID:  processingID,
			Tags:          make(map[string]string),
		},
	}

	// Extract event-specific data from the event payload
	if eventPayload := secEvent.Event.Payload(); eventPayload != nil {
		if payloadMap, ok := eventPayload.(map[string]interface{}); ok {
			for key, value := range payloadMap {
				event.Data[key] = value
			}
		}
	}

	return event
}

// slicesEqual compares two string slices for equality
func (b *Broker) slicesEqual(a, sliceB []string) bool {
	if len(a) != len(sliceB) {
		return false
	}

	for i, v := range a {
		if v != sliceB[i] {
			return false
		}
	}

	return true
}