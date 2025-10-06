package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-broker/pkg/models"
)

// Registry manages registered receivers
type Registry interface {
	// Register adds a new receiver
	Register(receiver *models.Receiver) error

	// Unregister removes a receiver
	Unregister(id string) error

	// Get retrieves a receiver by ID
	Get(id string) (*models.Receiver, error)

	// List returns all registered receivers
	List() ([]*models.Receiver, error)

	// Update updates an existing receiver
	Update(receiver *models.Receiver) error

	// GetByEventType returns all receivers that subscribe to a specific event type
	GetByEventType(eventType string) ([]*models.Receiver, error)

	// GetActiveReceivers returns all active receivers
	GetActiveReceivers() ([]*models.Receiver, error)

	// UpdateStats updates receiver statistics
	UpdateStats(id string, stats models.ReceiverMetadata) error

	// FilterReceivers returns receivers that match the given event
	FilterReceivers(event *models.SecurityEvent) ([]*models.Receiver, error)

	// IncrementEventReceived increments the event received count for a receiver
	IncrementEventReceived(id string) error

	// CountByStatus returns a count of receivers by their status
	CountByStatus() map[models.ReceiverStatus]int

	// Count returns the total number of registered receivers
	Count() int
}

// MemoryRegistry implements Registry using in-memory storage
type MemoryRegistry struct {
	receivers map[string]*models.Receiver
	mutex     sync.RWMutex
}

// NewMemoryRegistry creates a new in-memory registry
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		receivers: make(map[string]*models.Receiver),
	}
}

// Register adds a new receiver
func (r *MemoryRegistry) Register(receiver *models.Receiver) error {
	if receiver == nil {
		return fmt.Errorf("receiver cannot be nil")
	}

	if err := receiver.Validate(); err != nil {
		return fmt.Errorf("receiver validation failed: %w", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if receiver already exists
	if _, exists := r.receivers[receiver.ID]; exists {
		return fmt.Errorf("receiver with ID %s already exists", receiver.ID)
	}

	// Set defaults and metadata
	receiver.SetDefaults()
	receiver.Metadata.CreatedAt = time.Now()
	receiver.Metadata.UpdatedAt = time.Now()

	// Store the receiver
	r.receivers[receiver.ID] = receiver

	return nil
}

// Unregister removes a receiver
func (r *MemoryRegistry) Unregister(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.receivers[id]; !exists {
		return fmt.Errorf("receiver with ID %s not found", id)
	}

	delete(r.receivers, id)
	return nil
}

// Get retrieves a receiver by ID
func (r *MemoryRegistry) Get(id string) (*models.Receiver, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	receiver, exists := r.receivers[id]
	if !exists {
		return nil, fmt.Errorf("receiver with ID %s not found", id)
	}

	// Return a copy to prevent external modification
	return r.copyReceiver(receiver), nil
}

// List returns all registered receivers
func (r *MemoryRegistry) List() ([]*models.Receiver, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	receivers := make([]*models.Receiver, 0, len(r.receivers))
	for _, receiver := range r.receivers {
		receivers = append(receivers, r.copyReceiver(receiver))
	}

	return receivers, nil
}

// Update updates an existing receiver
func (r *MemoryRegistry) Update(receiver *models.Receiver) error {
	if receiver == nil {
		return fmt.Errorf("receiver cannot be nil")
	}

	if err := receiver.Validate(); err != nil {
		return fmt.Errorf("receiver validation failed: %w", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existing, exists := r.receivers[receiver.ID]
	if !exists {
		return fmt.Errorf("receiver with ID %s not found", receiver.ID)
	}

	// Preserve creation time and statistics
	receiver.Metadata.CreatedAt = existing.Metadata.CreatedAt
	receiver.Metadata.EventsReceived = existing.Metadata.EventsReceived
	receiver.Metadata.EventsDelivered = existing.Metadata.EventsDelivered
	receiver.Metadata.EventsFailed = existing.Metadata.EventsFailed
	receiver.Metadata.LastEventAt = existing.Metadata.LastEventAt
	receiver.Metadata.UpdatedAt = time.Now()

	// Update the receiver
	r.receivers[receiver.ID] = receiver

	return nil
}

// GetByEventType returns all receivers that subscribe to a specific event type
func (r *MemoryRegistry) GetByEventType(eventType string) ([]*models.Receiver, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var matchingReceivers []*models.Receiver

	for _, receiver := range r.receivers {
		// Only include active receivers
		if receiver.Status != models.ReceiverStatusActive {
			continue
		}

		// Check if receiver subscribes to this event type
		for _, subscribedType := range receiver.EventTypes {
			if subscribedType == eventType {
				matchingReceivers = append(matchingReceivers, r.copyReceiver(receiver))
				break
			}
		}
	}

	return matchingReceivers, nil
}

// GetActiveReceivers returns all active receivers
func (r *MemoryRegistry) GetActiveReceivers() ([]*models.Receiver, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var activeReceivers []*models.Receiver

	for _, receiver := range r.receivers {
		if receiver.Status == models.ReceiverStatusActive {
			activeReceivers = append(activeReceivers, r.copyReceiver(receiver))
		}
	}

	return activeReceivers, nil
}

// UpdateStats updates receiver statistics
func (r *MemoryRegistry) UpdateStats(id string, stats models.ReceiverMetadata) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	receiver, exists := r.receivers[id]
	if !exists {
		return fmt.Errorf("receiver with ID %s not found", id)
	}

	// Update statistics
	receiver.Metadata.EventsReceived = stats.EventsReceived
	receiver.Metadata.EventsDelivered = stats.EventsDelivered
	receiver.Metadata.EventsFailed = stats.EventsFailed
	receiver.Metadata.LastEventAt = stats.LastEventAt
	receiver.Metadata.LastDeliveryError = stats.LastDeliveryError
	receiver.Metadata.UpdatedAt = time.Now()

	return nil
}

// IncrementEventReceived increments the events received counter
func (r *MemoryRegistry) IncrementEventReceived(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	receiver, exists := r.receivers[id]
	if !exists {
		return fmt.Errorf("receiver with ID %s not found", id)
	}

	receiver.Metadata.EventsReceived++
	receiver.Metadata.LastEventAt = time.Now()
	receiver.Metadata.UpdatedAt = time.Now()

	return nil
}

// IncrementEventDelivered increments the events delivered counter
func (r *MemoryRegistry) IncrementEventDelivered(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	receiver, exists := r.receivers[id]
	if !exists {
		return fmt.Errorf("receiver with ID %s not found", id)
	}

	receiver.Metadata.EventsDelivered++
	receiver.Metadata.UpdatedAt = time.Now()

	return nil
}

// IncrementEventFailed increments the events failed counter and sets error message
func (r *MemoryRegistry) IncrementEventFailed(id string, errorMessage string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	receiver, exists := r.receivers[id]
	if !exists {
		return fmt.Errorf("receiver with ID %s not found", id)
	}

	receiver.Metadata.EventsFailed++
	receiver.Metadata.LastDeliveryError = errorMessage
	receiver.Metadata.UpdatedAt = time.Now()

	return nil
}

// SetReceiverStatus updates the status of a receiver
func (r *MemoryRegistry) SetReceiverStatus(id string, status models.ReceiverStatus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	receiver, exists := r.receivers[id]
	if !exists {
		return fmt.Errorf("receiver with ID %s not found", id)
	}

	receiver.Status = status
	receiver.Metadata.UpdatedAt = time.Now()

	return nil
}

// GetReceiverStats returns statistics for a receiver
func (r *MemoryRegistry) GetReceiverStats(id string) (*models.ReceiverMetadata, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	receiver, exists := r.receivers[id]
	if !exists {
		return nil, fmt.Errorf("receiver with ID %s not found", id)
	}

	// Return a copy of the metadata
	stats := receiver.Metadata
	return &stats, nil
}

// FilterReceivers returns receivers that match the provided event and filters
func (r *MemoryRegistry) FilterReceivers(event *models.SecurityEvent) ([]*models.Receiver, error) {
	receivers, err := r.GetByEventType(event.Type)
	if err != nil {
		return nil, err
	}

	var filteredReceivers []*models.Receiver

	for _, receiver := range receivers {
		// Apply event filters
		if r.matchesFilters(event, receiver.Filters) {
			filteredReceivers = append(filteredReceivers, receiver)
		}
	}

	return filteredReceivers, nil
}

// matchesFilters checks if an event matches all receiver filters
func (r *MemoryRegistry) matchesFilters(event *models.SecurityEvent, filters []models.EventFilter) bool {
	// If no filters, match all events
	if len(filters) == 0 {
		return true
	}

	// All filters must match
	for _, filter := range filters {
		if !filter.Matches(event) {
			return false
		}
	}

	return true
}

// copyReceiver creates a deep copy of a receiver
func (r *MemoryRegistry) copyReceiver(original *models.Receiver) *models.Receiver {
	if original == nil {
		return nil
	}

	receiverCopy := *original

	// Copy slices
	if original.EventTypes != nil {
		receiverCopy.EventTypes = make([]string, len(original.EventTypes))
		copy(receiverCopy.EventTypes, original.EventTypes)
	}

	if original.Filters != nil {
		receiverCopy.Filters = make([]models.EventFilter, len(original.Filters))
		copy(receiverCopy.Filters, original.Filters)
	}

	if original.Auth.Scopes != nil {
		receiverCopy.Auth.Scopes = make([]string, len(original.Auth.Scopes))
		copy(receiverCopy.Auth.Scopes, original.Auth.Scopes)
	}

	// Copy maps
	if original.Metadata.Tags != nil {
		receiverCopy.Metadata.Tags = make(map[string]string)
		for k, v := range original.Metadata.Tags {
			receiverCopy.Metadata.Tags[k] = v
		}
	}

	return &receiverCopy
}

// Count returns the total number of registered receivers
func (r *MemoryRegistry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.receivers)
}

// CountByStatus returns the number of receivers by status
func (r *MemoryRegistry) CountByStatus() map[models.ReceiverStatus]int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	counts := make(map[models.ReceiverStatus]int)

	for _, receiver := range r.receivers {
		counts[receiver.Status]++
	}

	return counts
}