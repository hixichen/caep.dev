package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// MockClient provides an in-memory implementation of PubSubClient for local testing
// This eliminates the need for GCP service accounts during development
type MockClient struct {
	projectID     string
	topicName     string
	logger        *slog.Logger
	hubInstanceID string

	// In-memory storage
	topics       map[string]*MockTopic
	subscriptions map[string]*MockSubscription
	mu           sync.RWMutex
}

// MockTopic represents an in-memory topic
type MockTopic struct {
	name      string
	messages  []*models.InternalMessage
	mu        sync.RWMutex
	created   time.Time
}

// MockSubscription represents an in-memory subscription
type MockSubscription struct {
	name       string
	topicName  string
	messages   []*models.InternalMessage
	mu         sync.RWMutex
	created    time.Time
	ackDeadline time.Duration
}

// NewMockClient creates a new in-memory Pub/Sub client for testing
func NewMockClient(ctx context.Context, projectID string) (*MockClient, error) {
	client := &MockClient{
		projectID:     projectID,
		topicName:     "ssf-hub-events", // Default unified topic
		logger:        slog.Default(),
		hubInstanceID: generateHubInstanceID(),
		topics:        make(map[string]*MockTopic),
		subscriptions: make(map[string]*MockSubscription),
	}

	// Auto-create the unified topic
	client.createTopic(client.topicName)

	client.logger.Info("Mock Pub/Sub client created",
		"project_id", projectID,
		"unified_topic", client.topicName,
		"hub_instance_id", client.hubInstanceID)

	return client, nil
}

// SetLogger sets the logger for the mock client
func (m *MockClient) SetLogger(logger *slog.Logger) {
	m.logger = logger
}

// SetTopicName sets the unified topic name
func (m *MockClient) SetTopicName(name string) {
	m.topicName = name
}

// Close closes the mock client (no-op for in-memory)
func (m *MockClient) Close() error {
	m.logger.Info("Mock Pub/Sub client closed")
	return nil
}

// PublishEvent publishes a security event to the unified topic
func (m *MockClient) PublishEvent(ctx context.Context, event *models.SecurityEvent, targetReceivers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert event to internal message format
	internalMsg := event.ToInternalMessage(targetReceivers, m.hubInstanceID)

	// Get or create topic
	topic := m.getOrCreateTopic(m.topicName)

	// Add message to topic
	topic.mu.Lock()
	topic.messages = append(topic.messages, internalMsg)
	topic.mu.Unlock()

	m.logger.Info("Event published to mock unified topic",
		"event_id", event.ID,
		"event_type", event.Type,
		"target_receivers", targetReceivers,
		"unified_topic", m.topicName,
		"message_count", len(topic.messages))

	return nil
}

// CreateHubSubscription creates the hub's internal subscription to the unified topic
func (m *MockClient) CreateHubSubscription(ctx context.Context, subscriptionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if subscription already exists
	if _, exists := m.subscriptions[subscriptionName]; exists {
		m.logger.Debug("Mock hub subscription already exists", "subscription", subscriptionName)
		return nil
	}

	// Create subscription
	subscription := &MockSubscription{
		name:        subscriptionName,
		topicName:   m.topicName,
		messages:    make([]*models.InternalMessage, 0),
		created:     time.Now(),
		ackDeadline: 60 * time.Second,
	}

	m.subscriptions[subscriptionName] = subscription

	m.logger.Info("Created mock hub subscription",
		"subscription", subscriptionName,
		"topic", m.topicName)

	return nil
}

// DeleteHubSubscription deletes the hub's subscription
func (m *MockClient) DeleteHubSubscription(ctx context.Context, subscriptionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.subscriptions, subscriptionName)

	m.logger.Info("Deleted mock hub subscription", "subscription", subscriptionName)
	return nil
}

// PullInternalMessages pulls messages from a subscription
func (m *MockClient) PullInternalMessages(ctx context.Context, subscriptionName string, maxMessages int, handler func(*models.InternalMessage) error) error {
	m.mu.RLock()
	subscription, exists := m.subscriptions[subscriptionName]
	if !exists {
		m.mu.RUnlock()
		return fmt.Errorf("subscription %s does not exist", subscriptionName)
	}
	m.mu.RUnlock()

	// Get topic messages
	m.mu.RLock()
	topic, exists := m.topics[subscription.topicName]
	if !exists {
		m.mu.RUnlock()
		return fmt.Errorf("topic %s does not exist", subscription.topicName)
	}
	m.mu.RUnlock()

	// Copy unprocessed messages to subscription
	topic.mu.RLock()
	var messagesToProcess []*models.InternalMessage
	for _, msg := range topic.messages {
		// Simple logic: if message not in subscription, it's new
		if !m.messageInSubscription(subscription, msg) {
			messagesToProcess = append(messagesToProcess, msg)
			if len(messagesToProcess) >= maxMessages {
				break
			}
		}
	}
	topic.mu.RUnlock()

	// Process messages
	for _, msg := range messagesToProcess {
		m.logger.Debug("Processing mock message",
			"subscription", subscriptionName,
			"message_id", msg.MessageID,
			"event_type", msg.Event.Type)

		// Add to subscription (mark as processed)
		subscription.mu.Lock()
		subscription.messages = append(subscription.messages, msg)
		subscription.mu.Unlock()

		// Call handler
		if err := handler(msg); err != nil {
			m.logger.Error("Mock message handler failed",
				"error", err,
				"message_id", msg.MessageID)
			return err
		}
	}

	if len(messagesToProcess) > 0 {
		m.logger.Info("Processed mock messages",
			"subscription", subscriptionName,
			"processed_count", len(messagesToProcess))
	}

	return nil
}

// GetHubInstanceID returns the hub instance ID
func (m *MockClient) GetHubInstanceID() string {
	return m.hubInstanceID
}

// Helper methods

func (m *MockClient) createTopic(name string) *MockTopic {
	topic := &MockTopic{
		name:     name,
		messages: make([]*models.InternalMessage, 0),
		created:  time.Now(),
	}
	m.topics[name] = topic
	return topic
}

func (m *MockClient) getOrCreateTopic(name string) *MockTopic {
	if topic, exists := m.topics[name]; exists {
		return topic
	}
	return m.createTopic(name)
}

func (m *MockClient) messageInSubscription(subscription *MockSubscription, msg *models.InternalMessage) bool {
	subscription.mu.RLock()
	defer subscription.mu.RUnlock()

	for _, subMsg := range subscription.messages {
		if subMsg.MessageID == msg.MessageID {
			return true
		}
	}
	return false
}

// GetMockStats returns statistics about the mock client for debugging
func (m *MockClient) GetMockStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"project_id":       m.projectID,
		"unified_topic":    m.topicName,
		"hub_instance_id":  m.hubInstanceID,
		"topics_count":     len(m.topics),
		"subscriptions_count": len(m.subscriptions),
		"topics":          make(map[string]interface{}),
		"subscriptions":   make(map[string]interface{}),
	}

	// Topic stats
	for name, topic := range m.topics {
		topic.mu.RLock()
		stats["topics"].(map[string]interface{})[name] = map[string]interface{}{
			"message_count": len(topic.messages),
			"created_at":    topic.created,
		}
		topic.mu.RUnlock()
	}

	// Subscription stats
	for name, sub := range m.subscriptions {
		sub.mu.RLock()
		stats["subscriptions"].(map[string]interface{})[name] = map[string]interface{}{
			"topic_name":     sub.topicName,
			"message_count":  len(sub.messages),
			"created_at":     sub.created,
			"ack_deadline":   sub.ackDeadline,
		}
		sub.mu.RUnlock()
	}

	return stats
}

// ClearAllMessages clears all messages from topics and subscriptions (for testing)
func (m *MockClient) ClearAllMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, topic := range m.topics {
		topic.mu.Lock()
		topic.messages = topic.messages[:0]
		topic.mu.Unlock()
	}

	for _, sub := range m.subscriptions {
		sub.mu.Lock()
		sub.messages = sub.messages[:0]
		sub.mu.Unlock()
	}

	m.logger.Info("Cleared all mock messages")
}