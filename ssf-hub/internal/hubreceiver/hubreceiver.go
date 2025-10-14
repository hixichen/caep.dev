package hubreceiver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/distributor"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// HubReceiver implements the hub as a receiver of its own unified topic
// This aligns the hub with the receiver model while handling internal distribution
type HubReceiver struct {
	hubInstanceID    string
	subscriptionName string
	pubsubClient     PubSubClient
	distributor      *distributor.EventDistributor
	logger           *slog.Logger
	running          bool
	stopChan         chan struct{}
	metrics          *HubReceiverMetrics
}

// PubSubClient interface for hub receiver operations
type PubSubClient interface {
	CreateHubSubscription(ctx context.Context, subscriptionName string) error
	DeleteHubSubscription(ctx context.Context, subscriptionName string) error
	PullInternalMessages(ctx context.Context, subscriptionName string, maxMessages int, handler func(*models.InternalMessage) error) error
	GetHubInstanceID() string
}

// HubReceiverMetrics tracks hub receiver performance
type HubReceiverMetrics struct {
	MessagesReceived   int64
	MessagesProcessed  int64
	MessagesFailed     int64
	ProcessingDuration time.Duration
	LastProcessedAt    time.Time
}

// Config contains configuration for the hub receiver
type Config struct {
	HubInstanceID    string
	PubSubClient     PubSubClient
	EventDistributor *distributor.EventDistributor
	Logger           *slog.Logger
	MaxMessages      int
	PullTimeout      time.Duration
}

// NewHubReceiver creates a new hub receiver
func NewHubReceiver(config *Config) *HubReceiver {
	hubInstanceID := config.HubInstanceID
	if hubInstanceID == "" {
		hubInstanceID = config.PubSubClient.GetHubInstanceID()
	}

	maxMessages := config.MaxMessages
	if maxMessages == 0 {
		maxMessages = 100
	}

	return &HubReceiver{
		hubInstanceID:    hubInstanceID,
		subscriptionName: generateHubSubscriptionName(hubInstanceID),
		pubsubClient:     config.PubSubClient,
		distributor:      config.EventDistributor,
		logger:           config.Logger,
		stopChan:         make(chan struct{}),
		metrics:          &HubReceiverMetrics{},
	}
}

// Start initializes and starts the hub receiver
func (h *HubReceiver) Start(ctx context.Context) error {
	h.logger.Info("Starting hub receiver",
		"hub_instance_id", h.hubInstanceID,
		"subscription", h.subscriptionName)

	// Create hub subscription if it doesn't exist
	if err := h.pubsubClient.CreateHubSubscription(ctx, h.subscriptionName); err != nil {
		return fmt.Errorf("failed to create hub subscription: %w", err)
	}

	h.running = true

	// Start message consumption loop
	go h.consumeMessages(ctx)

	h.logger.Info("Hub receiver started successfully",
		"subscription", h.subscriptionName)

	return nil
}

// SetPubSubClient updates the PubSub client (used for testing)
func (h *HubReceiver) SetPubSubClient(client PubSubClient) {
	h.pubsubClient = client
}

// Stop gracefully stops the hub receiver
func (h *HubReceiver) Stop(ctx context.Context) error {
	h.logger.Info("Stopping hub receiver", "subscription", h.subscriptionName)

	h.running = false
	close(h.stopChan)

	// Optionally delete the subscription (for cleanup)
	// In production, you might want to keep it for resilience
	if err := h.pubsubClient.DeleteHubSubscription(ctx, h.subscriptionName); err != nil {
		h.logger.Warn("Failed to delete hub subscription", "error", err)
	}

	h.logger.Info("Hub receiver stopped", "subscription", h.subscriptionName)
	return nil
}

// consumeMessages is the main message consumption loop
func (h *HubReceiver) consumeMessages(ctx context.Context) {
	h.logger.Info("Starting message consumption loop")

	for h.running {
		select {
		case <-h.stopChan:
			h.logger.Info("Received stop signal, exiting consumption loop")
			return
		case <-ctx.Done():
			h.logger.Info("Context cancelled, exiting consumption loop")
			return
		default:
			// Pull and process messages
			if err := h.pullAndProcessMessages(ctx); err != nil {
				h.logger.Error("Failed to pull messages", "error", err)

				// Back off on error to avoid tight error loops
				select {
				case <-time.After(5 * time.Second):
					continue
				case <-h.stopChan:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// pullAndProcessMessages pulls messages from the subscription and processes them
func (h *HubReceiver) pullAndProcessMessages(ctx context.Context) error {
	// Create a timeout context for this pull operation
	pullCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return h.pubsubClient.PullInternalMessages(pullCtx, h.subscriptionName, 100, h.processInternalMessage)
}

// processInternalMessage processes a single internal message
func (h *HubReceiver) processInternalMessage(msg *models.InternalMessage) error {
	start := time.Now()
	h.metrics.MessagesReceived++

	h.logger.Debug("Processing internal message",
		"message_id", msg.MessageID,
		"event_id", msg.Event.ID,
		"target_receivers", len(msg.Routing.TargetReceivers),
		"hub_instance", msg.Metadata.HubInstanceID)

	// Skip messages from this hub instance to avoid loops
	// (though this shouldn't happen with proper routing)
	if msg.Metadata.HubInstanceID == h.hubInstanceID {
		h.logger.Debug("Skipping message from same hub instance",
			"message_id", msg.MessageID,
			"hub_instance", msg.Metadata.HubInstanceID)
		return nil
	}

	// Check if message has expired
	if h.isMessageExpired(msg) {
		h.logger.Warn("Message expired, skipping",
			"message_id", msg.MessageID,
			"event_id", msg.Event.ID,
			"ttl", msg.Routing.TTL,
			"created_at", msg.Metadata.CreatedAt)
		return nil
	}

	// Distribute to receivers
	ctx := context.Background() // Use background context for distribution
	if err := h.distributor.DistributeToReceivers(ctx, msg); err != nil {
		h.metrics.MessagesFailed++
		return fmt.Errorf("failed to distribute message: %w", err)
	}

	// Update metrics
	duration := time.Since(start)
	h.metrics.MessagesProcessed++
	h.metrics.ProcessingDuration += duration
	h.metrics.LastProcessedAt = time.Now()

	h.logger.Debug("Successfully processed internal message",
		"message_id", msg.MessageID,
		"event_id", msg.Event.ID,
		"processing_duration", duration)

	return nil
}

// isMessageExpired checks if a message has exceeded its TTL
func (h *HubReceiver) isMessageExpired(msg *models.InternalMessage) bool {
	if msg.Routing.TTL <= 0 {
		return false // No TTL set
	}

	expiry := msg.Metadata.CreatedAt.Add(msg.Routing.TTL)
	return time.Now().After(expiry)
}

// GetMetrics returns hub receiver metrics
func (h *HubReceiver) GetMetrics() *HubReceiverMetrics {
	return h.metrics
}

// GetStatus returns the current status of the hub receiver
func (h *HubReceiver) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":           h.running,
		"hub_instance_id":   h.hubInstanceID,
		"subscription_name": h.subscriptionName,
		"metrics":           h.metrics,
	}
}

// HealthCheck performs a health check of the hub receiver
func (h *HubReceiver) HealthCheck() error {
	if !h.running {
		return fmt.Errorf("hub receiver is not running")
	}

	// Check if we've processed messages recently (within last 5 minutes)
	if !h.metrics.LastProcessedAt.IsZero() {
		timeSinceLastProcessed := time.Since(h.metrics.LastProcessedAt)
		if timeSinceLastProcessed > 5*time.Minute {
			return fmt.Errorf("no messages processed in last %v", timeSinceLastProcessed)
		}
	}

	// Check distributor health
	if err := h.distributor.HealthCheck(); err != nil {
		return fmt.Errorf("distributor unhealthy: %w", err)
	}

	return nil
}

// generateHubSubscriptionName generates a unique subscription name for this hub instance
func generateHubSubscriptionName(hubInstanceID string) string {
	return fmt.Sprintf("ssf-hub-subscription-%s", hubInstanceID)
}

// AsReceiver returns receiver information for this hub (for federation scenarios)
func (h *HubReceiver) AsReceiver() *models.Receiver {
	return &models.Receiver{
		ID:          h.hubInstanceID,
		Name:        fmt.Sprintf("SSF Hub Instance %s", h.hubInstanceID),
		Description: "Internal SSF Hub receiver for event distribution",
		EventTypes:  []string{"*"}, // Hub processes all event types
		Delivery: models.DeliveryConfig{
			Method: models.DeliveryMethodWebhook, // For federation scenarios
		},
		Status: models.ReceiverStatusActive,
		Metadata: models.ReceiverMetadata{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags: map[string]string{
				"type":     "hub",
				"instance": h.hubInstanceID,
				"role":     "internal-receiver",
			},
		},
	}
}
