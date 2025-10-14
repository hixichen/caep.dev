package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// Client wraps Google Cloud Pub/Sub client with SSF hub functionality
type Client struct {
	client        *pubsub.Client
	projectID     string
	unifiedTopic  *pubsub.Topic
	topicName     string
	logger        *slog.Logger
	hubInstanceID string
}

// NewClient creates a new Pub/Sub client with unified topic
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	c := &Client{
		client:        client,
		projectID:     projectID,
		topicName:     "ssf-hub-events", // Single unified topic
		logger:        slog.Default(),
		hubInstanceID: generateHubInstanceID(),
	}

	// Initialize the unified topic
	if err := c.initUnifiedTopic(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize unified topic: %w", err)
	}

	return c, nil
}

// SetLogger sets the logger for the client
func (c *Client) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

// SetTopicName sets the unified topic name
func (c *Client) SetTopicName(name string) {
	c.topicName = name
}

// Close closes the Pub/Sub client
func (c *Client) Close() error {
	// Stop the unified topic
	if c.unifiedTopic != nil {
		c.unifiedTopic.Stop()
	}
	return c.client.Close()
}

// PublishEvent publishes a security event to the unified topic
func (c *Client) PublishEvent(ctx context.Context, event *models.SecurityEvent, targetReceivers []string) error {
	// Convert event to internal message format for unified topic
	internalMsg := event.ToInternalMessage(targetReceivers, c.hubInstanceID)

	// Convert internal message to Pub/Sub format
	data, attributes, err := internalMsg.ToUnifiedPubSubMessage()
	if err != nil {
		return fmt.Errorf("failed to convert internal message to pubsub format: %w", err)
	}

	// Publish to unified topic
	msg := &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	}

	result := c.unifiedTopic.Publish(ctx, msg)
	messageID, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish message to unified topic: %w", err)
	}

	c.logger.Info("Event published to unified topic",
		"event_id", event.ID,
		"event_type", event.Type,
		"message_id", internalMsg.MessageID,
		"pubsub_message_id", messageID,
		"target_receivers", len(targetReceivers),
		"unified_topic", c.topicName)

	return nil
}

// CreateHubSubscription creates the hub's internal subscription to the unified topic
// This subscription is used by the hub to receive all events and distribute them to receivers
func (c *Client) CreateHubSubscription(ctx context.Context, subscriptionName string) error {
	subscription := c.client.Subscription(subscriptionName)
	exists, err := subscription.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check hub subscription existence: %w", err)
	}

	if !exists {
		cfg := pubsub.SubscriptionConfig{
			Topic:             c.unifiedTopic,
			AckDeadline:       60 * time.Second, // Hub processes messages quickly
			RetentionDuration: 7 * 24 * time.Hour,
			// No push config - hub uses pull-based processing for better control
		}

		_, err = c.client.CreateSubscription(ctx, subscriptionName, cfg)
		if err != nil {
			return fmt.Errorf("failed to create hub subscription %s: %w", subscriptionName, err)
		}

		c.logger.Info("Created hub subscription for unified topic",
			"subscription", subscriptionName,
			"topic", c.topicName)
	}

	return nil
}

// DeleteHubSubscription deletes the hub's subscription (for cleanup)
func (c *Client) DeleteHubSubscription(ctx context.Context, subscriptionName string) error {
	subscription := c.client.Subscription(subscriptionName)

	exists, err := subscription.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check hub subscription existence: %w", err)
	}

	if exists {
		if err := subscription.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete hub subscription %s: %w", subscriptionName, err)
		}

		c.logger.Info("Deleted hub subscription",
			"subscription", subscriptionName)
	}

	return nil
}

// PullInternalMessages pulls internal messages from the hub's subscription
// This is used by the hub to receive all events from the unified topic for processing
func (c *Client) PullInternalMessages(ctx context.Context, subscriptionName string, maxMessages int, handler func(*models.InternalMessage) error) error {
	subscription := c.client.Subscription(subscriptionName)

	// Configure receive settings
	subscription.ReceiveSettings.MaxOutstandingMessages = maxMessages

	// Create a context with timeout
	pullCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := subscription.Receive(pullCtx, func(ctx context.Context, msg *pubsub.Message) {
		// Deserialize internal message
		internalMsg, err := models.FromInternalMessageJSON(msg.Data)
		if err != nil {
			c.logger.Error("Failed to deserialize internal message", "error", err, "message_id", msg.ID)
			msg.Nack()
			return
		}

		// Process message through handler
		if err := handler(internalMsg); err != nil {
			c.logger.Error("Failed to process internal message", "error", err, "message_id", internalMsg.MessageID)
			msg.Nack()
			return
		}

		// Acknowledge successful processing
		msg.Ack()
		c.logger.Debug("Processed internal message", "message_id", internalMsg.MessageID, "event_id", internalMsg.Event.ID)
	})

	if err != nil && err != context.DeadlineExceeded {
		return fmt.Errorf("failed to receive internal messages: %w", err)
	}

	return nil
}

// initUnifiedTopic initializes the unified topic for the hub
func (c *Client) initUnifiedTopic(ctx context.Context) error {
	// Get topic reference
	c.unifiedTopic = c.client.Topic(c.topicName)

	// Check if topic exists
	exists, err := c.unifiedTopic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check unified topic existence: %w", err)
	}

	// Create topic if it doesn't exist
	if !exists {
		_, err = c.client.CreateTopic(ctx, c.topicName)
		if err != nil {
			return fmt.Errorf("failed to create unified topic: %w", err)
		}

		c.logger.Info("Created unified Pub/Sub topic", "topic", c.topicName)
	}

	return nil
}

// generateHubInstanceID generates a unique ID for this hub instance
func generateHubInstanceID() string {
	return fmt.Sprintf("hub_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
}

// GetHubInstanceID returns the hub instance ID
func (c *Client) GetHubInstanceID() string {
	return c.hubInstanceID
}

// GetUnifiedTopicInfo returns information about the unified topic
func (c *Client) GetUnifiedTopicInfo(ctx context.Context) (*TopicInfo, error) {
	exists, err := c.unifiedTopic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check unified topic existence: %w", err)
	}

	topicInfo := &TopicInfo{
		Name:   c.topicName,
		Exists: exists,
	}

	if exists {
		// Get topic config
		config, err := c.unifiedTopic.Config(ctx)
		if err != nil {
			c.logger.Error("Failed to get topic config", "error", err, "topic", c.topicName)
		} else {
			topicInfo.Config = &config
		}
	}

	return topicInfo, nil
}

// TopicInfo contains information about the unified topic
type TopicInfo struct {
	Name   string              `json:"name"`
	Exists bool                `json:"exists"`
	Config *pubsub.TopicConfig `json:"config,omitempty"`
}
