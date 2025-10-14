package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/sgnl-ai/caep.dev/secevent/pkg/parser"
	"github.com/sgnl-ai/caep.dev/secevent/pkg/token"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/distributor"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/hubreceiver"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// PubSubClient interface for unified hub pub/sub operations
type PubSubClient interface {
	PublishEvent(ctx context.Context, event *models.SecurityEvent, targetReceivers []string) error
	CreateHubSubscription(ctx context.Context, subscriptionName string) error
	DeleteHubSubscription(ctx context.Context, subscriptionName string) error
	PullInternalMessages(ctx context.Context, subscriptionName string, maxMessages int, handler func(*models.InternalMessage) error) error
	GetHubInstanceID() string
	Close() error
}

// Broker handles SSF event processing and distribution
type Broker struct {
	pubsubClient  PubSubClient
	registry      registry.Registry
	logger        *slog.Logger
	parser        *parser.Parser
	hubReceiver   *hubreceiver.HubReceiver
	distributor   *distributor.EventDistributor
	hubInstanceID string
}

// New creates a new SSF broker
func New(pubsubClient PubSubClient, registry registry.Registry, logger *slog.Logger) *Broker {
	// Initialize SEC event parser with no verification for demo
	// In production, configure with JWKS URL and expected issuer
	secEventParser := parser.NewParser()

	// Get hub instance ID
	hubInstanceID := pubsubClient.GetHubInstanceID()

	// Create event distributor
	distributorConfig := &distributor.Config{
		Registry:    registry,
		Logger:      logger,
		HTTPTimeout: 30 * time.Second,
	}
	eventDistributor := distributor.NewEventDistributor(distributorConfig)

	// Create hub receiver
	hubReceiverConfig := &hubreceiver.Config{
		HubInstanceID:    hubInstanceID,
		PubSubClient:     pubsubClient,
		EventDistributor: eventDistributor,
		Logger:           logger,
		MaxMessages:      100,
		PullTimeout:      30 * time.Second,
	}
	hubRec := hubreceiver.NewHubReceiver(hubReceiverConfig)

	return &Broker{
		pubsubClient:  pubsubClient,
		registry:      registry,
		logger:        logger,
		parser:        secEventParser,
		hubReceiver:   hubRec,
		distributor:   eventDistributor,
		hubInstanceID: hubInstanceID,
	}
}

// ProcessSecurityEvent processes an incoming security event token
func (b *Broker) ProcessSecurityEvent(ctx context.Context, rawSET string, transmitterID string) error {
	b.logger.Debug("Starting security event processing",
		"transmitter_id", transmitterID,
		"raw_set_length", len(rawSET))

	if rawSET == "" {
		b.logger.Error("Security event processing failed: empty SET")
		return fmt.Errorf("raw SET cannot be empty")
	}
	if transmitterID == "" {
		b.logger.Error("Security event processing failed: empty transmitter ID")
		return fmt.Errorf("transmitter ID cannot be empty")
	}

	// Parse the Security Event Token using secevent library
	// Note: Using ParseSecEventNoVerify for demo. In production, use ParseSecEvent with proper verification
	b.logger.Debug("Parsing security event token", "parser_type", "no_verify")
	secEvent, err := b.parser.ParseSecEventNoVerify(rawSET)
	if err != nil {
		previewLen := 100
		if len(rawSET) < previewLen {
			previewLen = len(rawSET)
		}
		b.logger.Error("Failed to parse security event token",
			"error", err,
			"transmitter_id", transmitterID,
			"raw_set_preview", rawSET[:previewLen])
		return fmt.Errorf("failed to parse security event token: %w", err)
	}

	b.logger.Debug("Security event token parsed successfully",
		"event_id", secEvent.ID,
		"issuer", secEvent.Issuer,
		"issued_at", secEvent.IssuedAt)

	// Convert to internal event model
	b.logger.Debug("Converting to internal event model")
	event := b.convertToSecurityEvent(secEvent, rawSET, transmitterID)
	b.logger.Debug("Event conversion completed",
		"event_id", event.ID,
		"event_type", event.Type,
		"processing_id", event.Metadata.ProcessingID)

	b.logger.Info("Processing security event",
		"event_id", event.ID,
		"event_type", event.Type,
		"transmitter_id", transmitterID,
		"subject_format", event.Subject.Format,
		"subject_identifier", event.Subject.Identifier)

	// Find receivers interested in this event type
	b.logger.Debug("Filtering receivers for event", "event_type", event.Type)
	receivers, err := b.registry.FilterReceivers(event)
	if err != nil {
		b.logger.Error("Failed to filter receivers",
			"error", err,
			"event_type", event.Type,
			"event_id", event.ID)
		return fmt.Errorf("failed to get receivers for event type %s: %w", event.Type, err)
	}

	b.logger.Debug("Receiver filtering completed",
		"matching_receivers", len(receivers),
		"event_type", event.Type)

	if len(receivers) == 0 {
		b.logger.Warn("No receivers found for event type",
			"event_type", event.Type,
			"event_id", event.ID,
			"transmitter_id", transmitterID)
		b.logger.Debug("Ending processing due to no receivers")
		return nil
	}

	b.logger.Info("Found receivers for event",
		"event_id", event.ID,
		"receiver_count", len(receivers))

	// Extract receiver IDs for the unified topic message
	b.logger.Debug("Preparing target receivers list")
	targetReceivers := make([]string, len(receivers))
	for i, receiver := range receivers {
		targetReceivers[i] = receiver.ID
		b.logger.Debug("Processing receiver",
			"receiver_id", receiver.ID,
			"receiver_status", receiver.Status,
			"webhook_url", receiver.WebhookURL)

		// Update statistics for interested receivers
		if err := b.registry.IncrementEventReceived(receiver.ID); err != nil {
			b.logger.Error("Failed to update receiver stats",
				"receiver_id", receiver.ID,
				"error", err,
				"event_id", event.ID)
		} else {
			b.logger.Debug("Updated receiver statistics", "receiver_id", receiver.ID)
		}
	}

	// Publish event to unified Pub/Sub topic for hub-managed distribution
	b.logger.Debug("Publishing event to unified topic",
		"target_receiver_count", len(targetReceivers),
		"target_receivers", targetReceivers)

	if err := b.pubsubClient.PublishEvent(ctx, event, targetReceivers); err != nil {
		b.logger.Error("Failed to publish event to unified topic",
			"error", err,
			"event_id", event.ID,
			"target_receivers", targetReceivers)
		return fmt.Errorf("failed to publish event to unified topic: %w", err)
	}

	b.logger.Debug("Event published to unified topic successfully",
		"event_id", event.ID,
		"message_size_approx", len(rawSET))

	b.logger.Info("Security event processed successfully",
		"event_id", event.ID,
		"receivers_notified", len(receivers))

	return nil
}

// RegisterReceiver registers a new event receiver
func (b *Broker) RegisterReceiver(ctx context.Context, receiverReq *models.ReceiverRequest) (*models.Receiver, error) {
	b.logger.Debug("Starting receiver registration",
		"receiver_id", receiverReq.ID,
		"name", receiverReq.Name,
		"event_types", receiverReq.EventTypes,
		"delivery_method", receiverReq.Delivery.Method,
		"webhook_url", receiverReq.WebhookURL)

	// Convert request to receiver
	receiver := receiverReq.ToReceiver()
	b.logger.Debug("Receiver request converted to receiver model",
		"receiver_id", receiver.ID,
		"status", receiver.Status)

	// Validate the receiver
	b.logger.Debug("Validating receiver configuration", "receiver_id", receiver.ID)
	if err := receiver.Validate(); err != nil {
		b.logger.Error("Receiver validation failed",
			"receiver_id", receiver.ID,
			"error", err,
			"webhook_url", receiver.WebhookURL,
			"event_types", receiver.EventTypes)
		return nil, fmt.Errorf("receiver validation failed: %w", err)
	}
	b.logger.Debug("Receiver validation passed", "receiver_id", receiver.ID)

	// Register with the registry
	b.logger.Debug("Registering receiver in registry", "receiver_id", receiver.ID)
	if err := b.registry.Register(receiver); err != nil {
		b.logger.Error("Failed to register receiver in registry",
			"receiver_id", receiver.ID,
			"error", err)
		return nil, fmt.Errorf("failed to register receiver: %w", err)
	}
	b.logger.Debug("Receiver registered in registry successfully", "receiver_id", receiver.ID)

	// Note: In the new architecture, receivers only receive webhooks from the hub
	// No direct Pub/Sub subscriptions are created for receivers

	b.logger.Info("Receiver registered successfully",
		"receiver_id", receiver.ID,
		"event_types", receiver.EventTypes,
		"delivery_method", receiver.Delivery.Method)
	b.logger.Debug("Registration process completed",
		"receiver_id", receiver.ID,
		"created_at", receiver.Metadata.CreatedAt,
		"tags", receiver.Metadata.Tags)

	return receiver, nil
}

// UnregisterReceiver removes a receiver
func (b *Broker) UnregisterReceiver(ctx context.Context, receiverID string) error {
	b.logger.Debug("Starting receiver unregistration", "receiver_id", receiverID)

	// Check if receiver exists
	receiver, err := b.registry.Get(receiverID)
	if err != nil {
		b.logger.Error("Cannot unregister receiver: not found",
			"receiver_id", receiverID,
			"error", err)
		return fmt.Errorf("receiver not found: %w", err)
	}

	b.logger.Debug("Found receiver for unregistration",
		"receiver_id", receiverID,
		"event_types", receiver.EventTypes,
		"status", receiver.Status)

	// Note: In the new architecture, receivers don't have direct Pub/Sub subscriptions
	// The hub manages all Pub/Sub operations internally

	// Unregister from registry
	b.logger.Debug("Removing receiver from registry", "receiver_id", receiverID)
	if err := b.registry.Unregister(receiverID); err != nil {
		b.logger.Error("Failed to remove receiver from registry",
			"receiver_id", receiverID,
			"error", err)
		return fmt.Errorf("failed to unregister receiver: %w", err)
	}
	b.logger.Debug("Receiver removed from registry", "receiver_id", receiverID)

	b.logger.Info("Receiver unregistered successfully", "receiver_id", receiverID)

	return nil
}

// UpdateReceiver updates an existing receiver configuration
func (b *Broker) UpdateReceiver(ctx context.Context, receiverReq *models.ReceiverRequest) (*models.Receiver, error) {
	b.logger.Debug("Starting receiver update",
		"receiver_id", receiverReq.ID,
		"new_event_types", receiverReq.EventTypes,
		"new_webhook_url", receiverReq.WebhookURL)

	// Get existing receiver
	existing, err := b.registry.Get(receiverReq.ID)
	if err != nil {
		b.logger.Error("Cannot update receiver: not found",
			"receiver_id", receiverReq.ID,
			"error", err)
		return nil, fmt.Errorf("receiver not found: %w", err)
	}

	b.logger.Debug("Found existing receiver for update",
		"receiver_id", receiverReq.ID,
		"current_event_types", existing.EventTypes,
		"current_status", existing.Status)

	// Convert request to receiver
	receiver := receiverReq.ToReceiver()
	b.logger.Debug("Converted update request to receiver model",
		"receiver_id", receiver.ID)

	// Check if event types changed
	eventTypesChanged := !b.slicesEqual(existing.EventTypes, receiver.EventTypes)
	b.logger.Debug("Analyzed configuration changes",
		"receiver_id", receiver.ID,
		"event_types_changed", eventTypesChanged,
		"old_event_types", existing.EventTypes,
		"new_event_types", receiver.EventTypes)

	// Update in registry
	b.logger.Debug("Updating receiver in registry", "receiver_id", receiver.ID)
	if err := b.registry.Update(receiver); err != nil {
		b.logger.Error("Failed to update receiver in registry",
			"receiver_id", receiver.ID,
			"error", err)
		return nil, fmt.Errorf("failed to update receiver: %w", err)
	}
	b.logger.Debug("Receiver updated in registry successfully", "receiver_id", receiver.ID)

	// Update Pub/Sub subscriptions if event types changed
	if eventTypesChanged {
		// Note: In the new architecture, no direct Pub/Sub subscription management needed
		// The hub handles all event distribution via webhooks
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
	b.logger.Debug("Calculating broker statistics")

	receivers, err := b.registry.List()
	if err != nil {
		b.logger.Error("Failed to get receivers for statistics", "error", err)
		return nil, fmt.Errorf("failed to get receivers: %w", err)
	}

	b.logger.Debug("Retrieved receivers for statistics", "receiver_count", len(receivers))

	stats := &BrokerStats{
		TotalReceivers:    len(receivers),
		ReceiversByStatus: b.registry.CountByStatus(),
		EventTypeStats:    make(map[string]int),
	}

	// Calculate event type statistics
	b.logger.Debug("Calculating event type statistics")
	for _, receiver := range receivers {
		b.logger.Debug("Processing receiver for stats",
			"receiver_id", receiver.ID,
			"status", receiver.Status,
			"event_type_count", len(receiver.EventTypes))
		for _, eventType := range receiver.EventTypes {
			stats.EventTypeStats[eventType]++
		}
	}

	b.logger.Debug("Statistics calculation completed",
		"total_receivers", stats.TotalReceivers,
		"unique_event_types", len(stats.EventTypeStats))

	return stats, nil
}

// BrokerStats contains broker statistics
type BrokerStats struct {
	TotalReceivers    int                           `json:"total_receivers"`
	ReceiversByStatus map[models.ReceiverStatus]int `json:"receivers_by_status"`
	EventTypeStats    map[string]int                `json:"event_type_stats"`
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

// Start initializes and starts the broker, including the hub receiver
func (b *Broker) Start(ctx context.Context) error {
	b.logger.Info("Starting SSF Hub controller", "hub_instance_id", b.hubInstanceID)
	b.logger.Debug("Controller start sequence initiated",
		"components", []string{"hub_receiver", "event_distributor"},
		"registry_type", "memory")

	// Start the hub receiver to consume from unified topic
	b.logger.Debug("Starting hub receiver component")
	if err := b.hubReceiver.Start(ctx); err != nil {
		b.logger.Error("Failed to start hub receiver",
			"error", err,
			"hub_instance_id", b.hubInstanceID)
		return fmt.Errorf("failed to start hub receiver: %w", err)
	}
	b.logger.Debug("Hub receiver started successfully")

	b.logger.Info("SSF Hub controller started successfully")
	b.logger.Debug("All controller components initialized and running")
	return nil
}

// Stop gracefully stops the broker and hub receiver
func (b *Broker) Stop(ctx context.Context) error {
	b.logger.Info("Stopping SSF Hub controller")
	b.logger.Debug("Controller shutdown sequence initiated")

	// Stop the hub receiver
	b.logger.Debug("Stopping hub receiver component")
	if err := b.hubReceiver.Stop(ctx); err != nil {
		b.logger.Error("Failed to stop hub receiver",
			"error", err,
			"hub_instance_id", b.hubInstanceID)
		return fmt.Errorf("failed to stop hub receiver: %w", err)
	}
	b.logger.Debug("Hub receiver stopped successfully")

	b.logger.Info("SSF Hub controller stopped successfully")
	b.logger.Debug("All controller components shut down cleanly")
	return nil
}

// GetHubReceiver returns the hub receiver for monitoring/management
func (b *Broker) GetHubReceiver() *hubreceiver.HubReceiver {
	return b.hubReceiver
}

// GetDistributor returns the event distributor for monitoring/management
func (b *Broker) GetDistributor() *distributor.EventDistributor {
	return b.distributor
}

// GetHubInstanceID returns the hub instance ID
func (b *Broker) GetHubInstanceID() string {
	return b.hubInstanceID
}

// AsReceiver returns receiver information for this hub (for federation scenarios)
func (b *Broker) AsReceiver() *models.Receiver {
	return b.hubReceiver.AsReceiver()
}
