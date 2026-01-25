package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// EventType represents the type of webhook event
type EventType string

const (
	EventModuleCreated       EventType = "module.created"
	EventModuleUpdated       EventType = "module.updated"
	EventModuleDeleted       EventType = "module.deleted"
	EventVersionCreated      EventType = "version.created"
	EventVersionDeleted      EventType = "version.deleted"
	EventCompilationStarted  EventType = "compilation.started"
	EventCompilationComplete EventType = "compilation.complete"
	EventCompilationFailed   EventType = "compilation.failed"
	EventBreakingChange      EventType = "breaking_change.detected"
	EventValidationFailed    EventType = "validation.failed"
)

// Event represents a webhook event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Webhook represents a registered webhook
type Webhook struct {
	ID          string      `json:"id"`
	URL         string      `json:"url"`
	Events      []EventType `json:"events"`
	Secret      string      `json:"secret,omitempty"`
	Active      bool        `json:"active"`
	Description string      `json:"description,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// WebhookManager manages webhooks
type WebhookManager struct {
	webhooks      map[string]*Webhook
	client        *http.Client
	deliveryStore *DeliveryLogStore
	retryWorker   *RetryWorker
	rateLimiter   *RateLimiter
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager() *WebhookManager {
	deliveryStore := NewDeliveryLogStore(1000)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	manager := &WebhookManager{
		webhooks:      make(map[string]*Webhook),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		deliveryStore: deliveryStore,
		rateLimiter:   NewRateLimiter(100, time.Minute), // 100 requests per minute per webhook
	}

	manager.retryWorker = NewRetryWorker(manager, deliveryStore, retryPolicy)

	return manager
}

// StartRetryWorker starts the retry worker
func (wm *WebhookManager) StartRetryWorker(ctx context.Context) {
	wm.retryWorker.Start(ctx, 30*time.Second) // Check for retries every 30 seconds
}

// StopRetryWorker stops the retry worker
func (wm *WebhookManager) StopRetryWorker() {
	wm.retryWorker.Stop()
}

// GetDeliveryLogs retrieves delivery logs for a webhook
func (wm *WebhookManager) GetDeliveryLogs(webhookID string, limit int) []*DeliveryLog {
	return wm.deliveryStore.GetByWebhook(webhookID, limit)
}

// GetDeliveryStats retrieves delivery statistics for a webhook
func (wm *WebhookManager) GetDeliveryStats(webhookID string) DeliveryStats {
	return wm.deliveryStore.GetStats(webhookID)
}

// RegisterWebhook registers a new webhook
func (wm *WebhookManager) RegisterWebhook(webhook *Webhook) error {
	if webhook.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}
	if len(webhook.Events) == 0 {
		return fmt.Errorf("at least one event type is required")
	}

	webhook.ID = generateID()
	webhook.Active = true
	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()

	wm.webhooks[webhook.ID] = webhook
	return nil
}

// UnregisterWebhook removes a webhook
func (wm *WebhookManager) UnregisterWebhook(id string) error {
	if _, exists := wm.webhooks[id]; !exists {
		return fmt.Errorf("webhook not found")
	}
	delete(wm.webhooks, id)
	return nil
}

// UpdateWebhook updates a webhook
func (wm *WebhookManager) UpdateWebhook(id string, updates *Webhook) error {
	webhook, exists := wm.webhooks[id]
	if !exists {
		return fmt.Errorf("webhook not found")
	}

	if updates.URL != "" {
		webhook.URL = updates.URL
	}
	if len(updates.Events) > 0 {
		webhook.Events = updates.Events
	}
	if updates.Secret != "" {
		webhook.Secret = updates.Secret
	}
	webhook.UpdatedAt = time.Now()

	return nil
}

// Dispatch sends an event to all registered webhooks
func (wm *WebhookManager) Dispatch(ctx context.Context, event *Event) error {
	event.ID = generateID()
	event.Timestamp = time.Now()

	for _, webhook := range wm.webhooks {
		if !webhook.Active {
			continue
		}

		// Check if webhook is interested in this event type
		interested := false
		for _, eventType := range webhook.Events {
			if eventType == event.Type {
				interested = true
				break
			}
		}

		if !interested {
			continue
		}

		// Create delivery log
		deliveryLog := &DeliveryLog{
			ID:        generateID(),
			WebhookID: webhook.ID,
			EventID:   event.ID,
			EventType: event.Type,
			URL:       webhook.URL,
			Status:    DeliveryStatusPending,
			Attempts:  0,
			CreatedAt: time.Now(),
		}
		wm.deliveryStore.Add(deliveryLog)

		// Send webhook asynchronously
		go wm.sendWebhookWithDeliveryLog(ctx, webhook, event, deliveryLog)
	}

	return nil
}

// sendWebhookWithDeliveryLog sends an event to a specific webhook with delivery logging
func (wm *WebhookManager) sendWebhookWithDeliveryLog(ctx context.Context, webhook *Webhook, event *Event, deliveryLog *DeliveryLog) {
	deliveryLog.Attempts++
	startTime := time.Now()

	err := wm.sendWebhook(ctx, webhook, event, deliveryLog)
	duration := time.Since(startTime)
	deliveryLog.Duration = duration

	if err != nil {
		// Check if we should retry
		retryPolicy := NewRetryPolicy(DefaultRetryConfig())
		if retryPolicy.ShouldRetry(deliveryLog.Attempts, err) {
			deliveryLog.Status = DeliveryStatusRetrying
			nextRetry := retryPolicy.NextRetryTime(deliveryLog.Attempts)
			deliveryLog.NextRetryAt = &nextRetry
			deliveryLog.ErrorMessage = err.Error()
		} else {
			deliveryLog.Status = DeliveryStatusFailed
			deliveryLog.ErrorMessage = err.Error()
			now := time.Now()
			deliveryLog.CompletedAt = &now
		}
	} else {
		deliveryLog.Status = DeliveryStatusSuccess
		now := time.Now()
		deliveryLog.CompletedAt = &now
	}

	wm.deliveryStore.Update(deliveryLog)
}

// sendWebhook sends an event to a specific webhook
func (wm *WebhookManager) sendWebhook(ctx context.Context, webhook *Webhook, event *Event, deliveryLog *DeliveryLog) error {
	// Check rate limit
	if !wm.rateLimiter.Allow(webhook.ID) {
		return fmt.Errorf("rate limit exceeded for webhook %s", webhook.ID)
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Spoke-Event", string(event.Type))
	req.Header.Set("X-Spoke-Event-ID", event.ID)
	req.Header.Set("X-Spoke-Delivery", time.Now().Format(time.RFC3339))

	// Add signature if secret is configured
	if webhook.Secret != "" {
		signature := generateSignature(payload, webhook.Secret)
		req.Header.Set("X-Spoke-Signature", signature)
	}

	// Record request headers if delivery log provided
	if deliveryLog != nil {
		deliveryLog.RequestHeaders = make(map[string]string)
		for key, values := range req.Header {
			if len(values) > 0 {
				deliveryLog.RequestHeaders[key] = values[0]
			}
		}
	}

	resp, err := wm.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Record status code
	if deliveryLog != nil {
		deliveryLog.StatusCode = resp.StatusCode
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

// sendWebhookWithLog is used by retry worker
func (wm *WebhookManager) sendWebhookWithLog(ctx context.Context, webhook *Webhook, event *Event, deliveryLog *DeliveryLog) error {
	return wm.sendWebhook(ctx, webhook, event, deliveryLog)
}

// VerifySignature verifies the webhook signature
func VerifySignature(payload []byte, signature, secret string) bool {
	expected := generateSignature(payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// generateSignature generates HMAC-SHA256 signature
func generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// generateID generates a unique ID
var idCounter int64

func generateID() string {
	// Use combination of timestamp and counter for uniqueness
	idCounter++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), idCounter)
}

// ListWebhooks returns all registered webhooks
func (wm *WebhookManager) ListWebhooks() []*Webhook {
	webhooks := make([]*Webhook, 0, len(wm.webhooks))
	for _, webhook := range wm.webhooks {
		webhooks = append(webhooks, webhook)
	}
	return webhooks
}

// GetWebhook retrieves a webhook by ID
func (wm *WebhookManager) GetWebhook(id string) (*Webhook, error) {
	webhook, exists := wm.webhooks[id]
	if !exists {
		return nil, fmt.Errorf("webhook not found")
	}
	return webhook, nil
}

// DeactivateWebhook deactivates a webhook
func (wm *WebhookManager) DeactivateWebhook(id string) error {
	webhook, exists := wm.webhooks[id]
	if !exists {
		return fmt.Errorf("webhook not found")
	}
	webhook.Active = false
	webhook.UpdatedAt = time.Now()
	return nil
}

// ActivateWebhook activates a webhook
func (wm *WebhookManager) ActivateWebhook(id string) error {
	webhook, exists := wm.webhooks[id]
	if !exists {
		return fmt.Errorf("webhook not found")
	}
	webhook.Active = true
	webhook.UpdatedAt = time.Now()
	return nil
}
