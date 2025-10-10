package distributor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

// EventDistributor handles distribution of security events to registered receivers
type EventDistributor struct {
	registry    registry.Registry
	httpClient  *http.Client
	logger      *slog.Logger
	retryPolicy *RetryPolicy
	metrics     *DistributorMetrics
}

// RetryPolicy defines retry behavior for failed deliveries
type RetryPolicy struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// DistributorMetrics tracks delivery metrics
type DistributorMetrics struct {
	DeliveriesAttempted int64
	DeliveriesSucceeded int64
	DeliveriesFailed    int64
	DeliveryDuration    time.Duration
}

// Config contains configuration for the event distributor
type Config struct {
	Registry    registry.Registry
	Logger      *slog.Logger
	HTTPTimeout time.Duration
	RetryPolicy *RetryPolicy
}

// NewEventDistributor creates a new event distributor
func NewEventDistributor(config *Config) *EventDistributor {
	if config.RetryPolicy == nil {
		config.RetryPolicy = &RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     60 * time.Second,
			Multiplier:      2.0,
		}
	}

	httpTimeout := config.HTTPTimeout
	if httpTimeout == 0 {
		httpTimeout = 30 * time.Second
	}

	return &EventDistributor{
		registry: config.Registry,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger:      config.Logger,
		retryPolicy: config.RetryPolicy,
		metrics:     &DistributorMetrics{},
	}
}

// DistributeToReceivers distributes an internal message to target receivers
func (d *EventDistributor) DistributeToReceivers(ctx context.Context, msg *models.InternalMessage) error {
	d.logger.Info("Distributing event to receivers",
		"message_id", msg.MessageID,
		"event_id", msg.Event.ID,
		"target_receivers", len(msg.Routing.TargetReceivers))

	for _, receiverID := range msg.Routing.TargetReceivers {
		// Get receiver details
		receiver, err := d.registry.Get(receiverID)
		if err != nil {
			d.logger.Error("Receiver not found",
				"receiver_id", receiverID,
				"event_id", msg.Event.ID,
				"error", err)
			continue
		}

		// Apply receiver-specific filtering
		if !d.shouldDeliverToReceiver(receiver, msg.Event) {
			d.logger.Debug("Event filtered out for receiver",
				"receiver_id", receiverID,
				"event_id", msg.Event.ID)
			continue
		}

		// Deliver based on delivery method
		switch receiver.Delivery.Method {
		case models.DeliveryMethodWebhook:
			go d.deliverWebhook(ctx, receiver, msg.Event)
		default:
			d.logger.Warn("Unsupported delivery method",
				"receiver_id", receiverID,
				"method", receiver.Delivery.Method)
		}
	}

	return nil
}

// shouldDeliverToReceiver applies receiver-specific filters
func (d *EventDistributor) shouldDeliverToReceiver(receiver *models.Receiver, event *models.SecurityEvent) bool {
	// Check event type subscription
	eventTypeMatched := false
	for _, eventType := range receiver.EventTypes {
		if eventType == "*" || eventType == event.Type {
			eventTypeMatched = true
			break
		}
	}

	if !eventTypeMatched {
		return false
	}

	// Apply custom filters
	for _, filter := range receiver.Filters {
		if !filter.Matches(event) {
			return false
		}
	}

	return true
}

// deliverWebhook delivers an event via HTTP webhook
func (d *EventDistributor) deliverWebhook(ctx context.Context, receiver *models.Receiver, event *models.SecurityEvent) {
	delivery := &models.EventDelivery{
		DeliveryID: d.generateDeliveryID(),
		ReceiverID: receiver.ID,
		EventID:    event.ID,
		Attempt:    0,
		Status:     models.DeliveryStatusPending,
	}

	d.performDeliveryWithRetries(ctx, receiver, event, delivery)
}

// performDeliveryWithRetries performs webhook delivery with retry logic
func (d *EventDistributor) performDeliveryWithRetries(ctx context.Context, receiver *models.Receiver, event *models.SecurityEvent, delivery *models.EventDelivery) {
	backoff := d.retryPolicy.InitialInterval

	for attempt := 1; attempt <= d.retryPolicy.MaxRetries+1; attempt++ {
		delivery.Attempt = attempt
		start := time.Now()

		success, statusCode, responseBody, err := d.performWebhookDelivery(ctx, receiver, event)
		duration := time.Since(start)

		// Update delivery record
		delivery.Duration = duration
		delivery.ResponseCode = statusCode
		delivery.ResponseBody = responseBody

		if err != nil {
			delivery.ErrorMessage = err.Error()
		}

		// Update metrics
		d.metrics.DeliveriesAttempted++
		d.metrics.DeliveryDuration += duration

		if success {
			// Successful delivery
			delivery.Status = models.DeliveryStatusDelivered
			delivery.DeliveredAt = time.Now()
			d.metrics.DeliveriesSucceeded++

			d.logger.Info("Event delivered successfully",
				"receiver_id", receiver.ID,
				"event_id", event.ID,
				"delivery_id", delivery.DeliveryID,
				"attempt", attempt,
				"duration", duration,
				"status_code", statusCode)

			// Update receiver stats
			if updateErr := d.registry.IncrementEventDelivered(receiver.ID); updateErr != nil {
				d.logger.Error("Failed to update receiver stats",
					"receiver_id", receiver.ID,
					"error", updateErr)
			}

			return
		}

		// Failed delivery
		d.metrics.DeliveriesFailed++
		d.logger.Warn("Event delivery failed",
			"receiver_id", receiver.ID,
			"event_id", event.ID,
			"delivery_id", delivery.DeliveryID,
			"attempt", attempt,
			"error", err,
			"status_code", statusCode,
			"response", responseBody)

		// Update receiver failure stats
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		if updateErr := d.registry.IncrementEventFailed(receiver.ID, errorMsg); updateErr != nil {
			d.logger.Error("Failed to update receiver failure stats",
				"receiver_id", receiver.ID,
				"error", updateErr)
		}

		// Check if we should retry
		if attempt <= d.retryPolicy.MaxRetries {
			delivery.NextRetryAt = time.Now().Add(backoff)
			d.logger.Info("Scheduling delivery retry",
				"receiver_id", receiver.ID,
				"event_id", event.ID,
				"delivery_id", delivery.DeliveryID,
				"next_attempt", attempt+1,
				"backoff", backoff)

			// Wait before retry
			select {
			case <-ctx.Done():
				delivery.Status = models.DeliveryStatusAbandoned
				return
			case <-time.After(backoff):
				// Continue to next retry
			}

			// Exponential backoff
			backoff = time.Duration(float64(backoff) * d.retryPolicy.Multiplier)
			if backoff > d.retryPolicy.MaxInterval {
				backoff = d.retryPolicy.MaxInterval
			}
		} else {
			// Max retries exceeded
			delivery.Status = models.DeliveryStatusAbandoned
			d.logger.Error("Event delivery abandoned after max retries",
				"receiver_id", receiver.ID,
				"event_id", event.ID,
				"delivery_id", delivery.DeliveryID,
				"attempts", attempt-1)
		}
	}
}

// performWebhookDelivery performs a single webhook delivery attempt
func (d *EventDistributor) performWebhookDelivery(ctx context.Context, receiver *models.Receiver, event *models.SecurityEvent) (success bool, statusCode int, responseBody string, err error) {
	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return false, 0, "", fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", receiver.WebhookURL, bytes.NewBuffer(eventData))
	if err != nil {
		return false, 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SSF-Hub/1.0")
	req.Header.Set("X-SSF-Event-ID", event.ID)
	req.Header.Set("X-SSF-Event-Type", event.Type)

	// Add authentication
	if err := d.addAuthentication(req, receiver); err != nil {
		return false, 0, "", fmt.Errorf("failed to add authentication: %w", err)
	}

	// Perform request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return false, 0, "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody := make([]byte, 1024) // Limit response body size
	n, _ := resp.Body.Read(respBody)
	responseBody = string(respBody[:n])

	// Check if delivery was successful (2xx status codes)
	success = resp.StatusCode >= 200 && resp.StatusCode < 300
	statusCode = resp.StatusCode

	return success, statusCode, responseBody, nil
}

// addAuthentication adds authentication headers to the request
func (d *EventDistributor) addAuthentication(req *http.Request, receiver *models.Receiver) error {
	switch receiver.Auth.Type {
	case models.AuthTypeNone:
		// No authentication required
		return nil

	case models.AuthTypeBearer:
		if receiver.Auth.Token == "" {
			return fmt.Errorf("bearer token is required but not provided")
		}
		req.Header.Set("Authorization", "Bearer "+receiver.Auth.Token)
		return nil

	case models.AuthTypeHMAC:
		if receiver.Auth.Secret == "" {
			return fmt.Errorf("HMAC secret is required but not provided")
		}
		// TODO: Implement HMAC signature
		return fmt.Errorf("HMAC authentication not yet implemented")

	case models.AuthTypeOAuth2:
		// TODO: Implement OAuth2 token acquisition
		return fmt.Errorf("OAuth2 authentication not yet implemented")

	default:
		return fmt.Errorf("unsupported authentication type: %s", receiver.Auth.Type)
	}
}

// generateDeliveryID generates a unique delivery ID
func (d *EventDistributor) generateDeliveryID() string {
	return "delivery_" + uuid.New().String()
}

// GetMetrics returns distributor metrics
func (d *EventDistributor) GetMetrics() *DistributorMetrics {
	return d.metrics
}

// HealthCheck checks the distributor's health
func (d *EventDistributor) HealthCheck() error {
	// Check if we can reach the registry
	_, err := d.registry.List()
	if err != nil {
		return fmt.Errorf("registry unreachable: %w", err)
	}

	return nil
}