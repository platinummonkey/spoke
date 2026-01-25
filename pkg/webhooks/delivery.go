package webhooks

import (
	"sync"
	"time"
)

// DeliveryStatus represents the status of a webhook delivery
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusSuccess   DeliveryStatus = "success"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
)

// DeliveryLog represents a webhook delivery attempt
type DeliveryLog struct {
	ID             string         `json:"id"`
	WebhookID      string         `json:"webhook_id"`
	EventID        string         `json:"event_id"`
	EventType      EventType      `json:"event_type"`
	URL            string         `json:"url"`
	Status         DeliveryStatus `json:"status"`
	StatusCode     int            `json:"status_code,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	Attempts       int            `json:"attempts"`
	NextRetryAt    *time.Time     `json:"next_retry_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	Duration       time.Duration  `json:"duration,omitempty"`
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	ResponseBody   string         `json:"response_body,omitempty"`
}

// DeliveryLogStore manages delivery logs
type DeliveryLogStore struct {
	logs  map[string]*DeliveryLog
	mutex sync.RWMutex
	maxLogs int
}

// NewDeliveryLogStore creates a new delivery log store
func NewDeliveryLogStore(maxLogs int) *DeliveryLogStore {
	if maxLogs <= 0 {
		maxLogs = 1000 // Default to 1000 logs
	}
	return &DeliveryLogStore{
		logs:    make(map[string]*DeliveryLog),
		maxLogs: maxLogs,
	}
}

// Add adds a delivery log
func (s *DeliveryLogStore) Add(log *DeliveryLog) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Evict oldest logs if we exceed maxLogs
	if len(s.logs) >= s.maxLogs {
		s.evictOldest()
	}

	s.logs[log.ID] = log
}

// Get retrieves a delivery log by ID
func (s *DeliveryLogStore) Get(id string) (*DeliveryLog, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	log, exists := s.logs[id]
	return log, exists
}

// GetByWebhook retrieves all delivery logs for a webhook
func (s *DeliveryLogStore) GetByWebhook(webhookID string, limit int) []*DeliveryLog {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*DeliveryLog
	for _, log := range s.logs {
		if log.WebhookID == webhookID {
			result = append(result, log)
		}
	}

	// Sort by created_at descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].CreatedAt.Before(result[j].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result
}

// GetByEvent retrieves delivery logs for an event
func (s *DeliveryLogStore) GetByEvent(eventID string) []*DeliveryLog {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*DeliveryLog
	for _, log := range s.logs {
		if log.EventID == eventID {
			result = append(result, log)
		}
	}
	return result
}

// Update updates a delivery log
func (s *DeliveryLogStore) Update(log *DeliveryLog) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.logs[log.ID] = log
}

// GetPendingRetries returns delivery logs that need retry
func (s *DeliveryLogStore) GetPendingRetries() []*DeliveryLog {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	now := time.Now()
	var result []*DeliveryLog
	for _, log := range s.logs {
		if log.Status == DeliveryStatusRetrying &&
		   log.NextRetryAt != nil &&
		   log.NextRetryAt.Before(now) {
			result = append(result, log)
		}
	}
	return result
}

// evictOldest removes the oldest 10% of logs
func (s *DeliveryLogStore) evictOldest() {
	// Convert to slice for sorting
	logs := make([]*DeliveryLog, 0, len(s.logs))
	for _, log := range s.logs {
		logs = append(logs, log)
	}

	// Sort by created_at ascending
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[i].CreatedAt.After(logs[j].CreatedAt) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}

	// Remove oldest 10%
	evictCount := len(logs) / 10
	if evictCount == 0 {
		evictCount = 1
	}

	for i := 0; i < evictCount && i < len(logs); i++ {
		delete(s.logs, logs[i].ID)
	}
}

// GetStats returns delivery statistics for a webhook
func (s *DeliveryLogStore) GetStats(webhookID string) DeliveryStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := DeliveryStats{
		WebhookID: webhookID,
	}

	for _, log := range s.logs {
		if log.WebhookID != webhookID {
			continue
		}

		stats.Total++
		switch log.Status {
		case DeliveryStatusSuccess:
			stats.Successful++
		case DeliveryStatusFailed:
			stats.Failed++
		case DeliveryStatusRetrying:
			stats.Retrying++
		}

		if log.CompletedAt != nil {
			stats.TotalDuration += log.Duration
		}
	}

	if stats.Successful > 0 {
		stats.AverageDuration = stats.TotalDuration / time.Duration(stats.Successful)
	}

	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Successful) / float64(stats.Total)
	}

	return stats
}

// DeliveryStats represents delivery statistics
type DeliveryStats struct {
	WebhookID       string        `json:"webhook_id"`
	Total           int           `json:"total"`
	Successful      int           `json:"successful"`
	Failed          int           `json:"failed"`
	Retrying        int           `json:"retrying"`
	SuccessRate     float64       `json:"success_rate"`
	AverageDuration time.Duration `json:"average_duration"`
	TotalDuration   time.Duration `json:"total_duration"`
}
