package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-broker/pkg/models"
)

// Client wraps Google Cloud Pub/Sub client with SSF broker functionality
type Client struct {
	client      *pubsub.Client
	projectID   string
	topicPrefix string
	logger      *slog.Logger
	topics      map[string]*pubsub.Topic
}

// NewClient creates a new Pub/Sub client
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	return &Client{
		client:      client,
		projectID:   projectID,
		topicPrefix: "ssf-events",
		logger:      slog.Default(),
		topics:      make(map[string]*pubsub.Topic),
	}, nil
}

// SetLogger sets the logger for the client
func (c *Client) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

// SetTopicPrefix sets the prefix for topic names
func (c *Client) SetTopicPrefix(prefix string) {
	c.topicPrefix = prefix
}

// Close closes the Pub/Sub client
func (c *Client) Close() error {
	// Close all topics
	for _, topic := range c.topics {
		topic.Stop()
	}
	return c.client.Close()
}

// PublishEvent publishes a security event to Pub/Sub
func (c *Client) PublishEvent(ctx context.Context, event *models.SecurityEvent) error {
	// Get or create topic for this event type
	topicName := c.getTopicName(event.Type)
	topic, err := c.getOrCreateTopic(ctx, topicName)
	if err != nil {
		return fmt.Errorf("failed to get topic %s: %w", topicName, err)
	}

	// Convert event to Pub/Sub message
	data, attributes, err := event.ToPubSubMessage()
	if err != nil {
		return fmt.Errorf("failed to convert event to pubsub message: %w", err)
	}

	// Publish the message
	msg := &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	}

	result := topic.Publish(ctx, msg)
	messageID, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	c.logger.Info("Event published to Pub/Sub",
		"event_id", event.ID,
		"event_type", event.Type,
		"topic", topicName,
		"message_id", messageID)

	return nil
}

// CreateReceiverSubscription creates a subscription for a receiver
func (c *Client) CreateReceiverSubscription(ctx context.Context, receiver *models.Receiver) error {
	for _, eventType := range receiver.EventTypes {
		topicName := c.getTopicName(eventType)
		subscriptionName := c.getSubscriptionName(receiver.ID, eventType)

		// Get or create topic
		topic, err := c.getOrCreateTopic(ctx, topicName)
		if err != nil {
			return fmt.Errorf("failed to get topic %s: %w", topicName, err)
		}

		// Create subscription if it doesn't exist
		subscription := c.client.Subscription(subscriptionName)
		exists, err := subscription.Exists(ctx)
		if err != nil {
			return fmt.Errorf("failed to check subscription existence: %w", err)
		}

		if !exists {
			cfg := pubsub.SubscriptionConfig{
				Topic:             topic,
				AckDeadline:       time.Duration(receiver.Retry.MaxInterval),
				RetentionDuration: 7 * 24 * time.Hour,
				RetryPolicy: &pubsub.RetryPolicy{
					MinimumBackoff: receiver.Retry.InitialInterval,
					MaximumBackoff: receiver.Retry.MaxInterval,
				},
			}

			// Configure push delivery if webhook URL is provided
			if receiver.Delivery.Method == models.DeliveryMethodWebhook && receiver.WebhookURL != "" {
				cfg.PushConfig = pubsub.PushConfig{
					Endpoint: receiver.WebhookURL,
				}

				// Add authentication headers if configured
				if receiver.Auth.Type == models.AuthTypeBearer {
					cfg.PushConfig.AuthenticationMethod = &pubsub.OIDCToken{
						ServiceAccountEmail: "", // Will use default service account
					}
				}
			}

			_, err = c.client.CreateSubscription(ctx, subscriptionName, cfg)
			if err != nil {
				return fmt.Errorf("failed to create subscription %s: %w", subscriptionName, err)
			}

			c.logger.Info("Created subscription for receiver",
				"receiver_id", receiver.ID,
				"subscription", subscriptionName,
				"topic", topicName,
				"delivery_method", receiver.Delivery.Method)
		}
	}

	return nil
}

// DeleteReceiverSubscription deletes subscriptions for a receiver
func (c *Client) DeleteReceiverSubscription(ctx context.Context, receiverID string, eventTypes []string) error {
	for _, eventType := range eventTypes {
		subscriptionName := c.getSubscriptionName(receiverID, eventType)
		subscription := c.client.Subscription(subscriptionName)

		exists, err := subscription.Exists(ctx)
		if err != nil {
			return fmt.Errorf("failed to check subscription existence: %w", err)
		}

		if exists {
			if err := subscription.Delete(ctx); err != nil {
				return fmt.Errorf("failed to delete subscription %s: %w", subscriptionName, err)
			}

			c.logger.Info("Deleted subscription for receiver",
				"receiver_id", receiverID,
				"subscription", subscriptionName,
				"event_type", eventType)
		}
	}

	return nil
}

// PullEvents pulls events from a subscription for a receiver
func (c *Client) PullEvents(ctx context.Context, receiverID string, eventType string, maxMessages int) ([]*models.SecurityEvent, error) {
	subscriptionName := c.getSubscriptionName(receiverID, eventType)
	subscription := c.client.Subscription(subscriptionName)

	// Configure receive settings
	subscription.ReceiveSettings.MaxOutstandingMessages = maxMessages

	events := make([]*models.SecurityEvent, 0, maxMessages)
	received := 0

	// Create a context with timeout
	pullCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := subscription.Receive(pullCtx, func(ctx context.Context, msg *pubsub.Message) {
		if received >= maxMessages {
			return
		}

		event, err := models.FromJSON(msg.Data)
		if err != nil {
			c.logger.Error("Failed to deserialize event", "error", err, "message_id", msg.ID)
			msg.Nack()
			return
		}

		events = append(events, event)
		received++
		msg.Ack()
	})

	if err != nil && err != context.DeadlineExceeded {
		return nil, fmt.Errorf("failed to receive messages: %w", err)
	}

	return events, nil
}

// getOrCreateTopic gets an existing topic or creates a new one
func (c *Client) getOrCreateTopic(ctx context.Context, topicName string) (*pubsub.Topic, error) {
	// Check if we already have this topic
	if topic, exists := c.topics[topicName]; exists {
		return topic, nil
	}

	// Get topic reference
	topic := c.client.Topic(topicName)

	// Check if topic exists
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check topic existence: %w", err)
	}

	// Create topic if it doesn't exist
	if !exists {
		_, err = c.client.CreateTopic(ctx, topicName)
		if err != nil {
			return nil, fmt.Errorf("failed to create topic: %w", err)
		}

		c.logger.Info("Created Pub/Sub topic", "topic", topicName)
	}

	// Cache the topic
	c.topics[topicName] = topic

	return topic, nil
}

// getTopicName generates a topic name for an event type
func (c *Client) getTopicName(eventType string) string {
	// Convert event type URI to a valid topic name
	// e.g., "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	// becomes "ssf-events-session-revoked"

	// Extract the last part of the URI
	parts := strings.Split(eventType, "/")
	if len(parts) > 0 {
		eventName := parts[len(parts)-1]
		return fmt.Sprintf("%s-%s", c.topicPrefix, eventName)
	}

	// Fallback to a general topic
	return fmt.Sprintf("%s-general", c.topicPrefix)
}

// getSubscriptionName generates a subscription name for a receiver and event type
func (c *Client) getSubscriptionName(receiverID, eventType string) string {
	// Extract event name from URI
	parts := strings.Split(eventType, "/")
	eventName := "general"
	if len(parts) > 0 {
		eventName = parts[len(parts)-1]
	}

	return fmt.Sprintf("%s-%s-%s", c.topicPrefix, receiverID, eventName)
}

// GetSubscriptionInfo returns information about subscriptions for a receiver
func (c *Client) GetSubscriptionInfo(ctx context.Context, receiverID string, eventTypes []string) (map[string]*SubscriptionInfo, error) {
	info := make(map[string]*SubscriptionInfo)

	for _, eventType := range eventTypes {
		subscriptionName := c.getSubscriptionName(receiverID, eventType)
		subscription := c.client.Subscription(subscriptionName)

		exists, err := subscription.Exists(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check subscription existence: %w", err)
		}

		subInfo := &SubscriptionInfo{
			Name:      subscriptionName,
			EventType: eventType,
			Exists:    exists,
		}

		if exists {
			// Get subscription config
			config, err := subscription.Config(ctx)
			if err != nil {
				c.logger.Error("Failed to get subscription config", "error", err, "subscription", subscriptionName)
			} else {
				subInfo.Config = &config
			}
		}

		info[eventType] = subInfo
	}

	return info, nil
}

// SubscriptionInfo contains information about a subscription
type SubscriptionInfo struct {
	Name      string                      `json:"name"`
	EventType string                      `json:"event_type"`
	Exists    bool                        `json:"exists"`
	Config    *pubsub.SubscriptionConfig  `json:"config,omitempty"`
}