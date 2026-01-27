package webhooks

import (
	"context"
	"fmt"
	"math"
	"runtime/debug"
	"time"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffMultiplier float64      `json:"backoff_multiplier"`
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     5,
		InitialDelay:    1 * time.Second,
		MaxDelay:        5 * time.Minute,
		BackoffMultiplier: 2.0,
	}
}

// RetryPolicy implements exponential backoff retry logic
type RetryPolicy struct {
	config RetryConfig
}

// NewRetryPolicy creates a new retry policy
func NewRetryPolicy(config RetryConfig) *RetryPolicy {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 5
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 1 * time.Second
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 5 * time.Minute
	}
	if config.BackoffMultiplier <= 1.0 {
		config.BackoffMultiplier = 2.0
	}

	return &RetryPolicy{
		config: config,
	}
}

// ShouldRetry determines if a delivery should be retried
func (p *RetryPolicy) ShouldRetry(attempts int, err error) bool {
	if err == nil {
		return false
	}

	if attempts >= p.config.MaxAttempts {
		return false
	}

	return true
}

// NextRetryDelay calculates the delay before the next retry
func (p *RetryPolicy) NextRetryDelay(attempts int) time.Duration {
	if attempts <= 0 {
		return p.config.InitialDelay
	}

	// Exponential backoff: delay = initialDelay * (multiplier ^ (attempts - 1))
	delay := float64(p.config.InitialDelay) * math.Pow(p.config.BackoffMultiplier, float64(attempts-1))

	// Cap at max delay
	if delay > float64(p.config.MaxDelay) {
		return p.config.MaxDelay
	}

	return time.Duration(delay)
}

// NextRetryTime calculates when the next retry should occur
func (p *RetryPolicy) NextRetryTime(attempts int) time.Time {
	delay := p.NextRetryDelay(attempts)
	return time.Now().Add(delay)
}

// RetryWorker processes failed webhook deliveries
type RetryWorker struct {
	manager       *WebhookManager
	deliveryStore *DeliveryLogStore
	retryPolicy   *RetryPolicy
	stopCh        chan struct{}
	ticker        *time.Ticker
}

// NewRetryWorker creates a new retry worker
func NewRetryWorker(manager *WebhookManager, deliveryStore *DeliveryLogStore, retryPolicy *RetryPolicy) *RetryWorker {
	return &RetryWorker{
		manager:       manager,
		deliveryStore: deliveryStore,
		retryPolicy:   retryPolicy,
		stopCh:        make(chan struct{}),
	}
}

// Start starts the retry worker
func (w *RetryWorker) Start(ctx context.Context, checkInterval time.Duration) {
	w.ticker = time.NewTicker(checkInterval)

	go func() {
		// Recover from panics to prevent crashing the process
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[RetryWorker] PANIC: %v\n%s\n", r, debug.Stack())
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-w.ticker.C:
				w.processRetries(ctx)
			}
		}
	}()
}

// Stop stops the retry worker
func (w *RetryWorker) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
	close(w.stopCh)
}

// processRetries processes all pending retries
func (w *RetryWorker) processRetries(ctx context.Context) {
	logs := w.deliveryStore.GetPendingRetries()

	for _, log := range logs {
		// Get the webhook
		webhook, err := w.manager.GetWebhook(log.WebhookID)
		if err != nil {
			// Webhook no longer exists, mark as failed
			log.Status = DeliveryStatusFailed
			log.ErrorMessage = fmt.Sprintf("webhook not found: %v", err)
			now := time.Now()
			log.CompletedAt = &now
			w.deliveryStore.Update(log)
			continue
		}

		if !webhook.Active {
			// Webhook is inactive, mark as failed
			log.Status = DeliveryStatusFailed
			log.ErrorMessage = "webhook is inactive"
			now := time.Now()
			log.CompletedAt = &now
			w.deliveryStore.Update(log)
			continue
		}

		// Attempt retry
		w.retryDelivery(ctx, webhook, log)
	}
}

// retryDelivery attempts to deliver a webhook again
func (w *RetryWorker) retryDelivery(ctx context.Context, webhook *Webhook, log *DeliveryLog) {
	log.Attempts++

	// Recreate the event from the log
	event := &Event{
		ID:        log.EventID,
		Type:      log.EventType,
		Timestamp: log.CreatedAt,
		Data:      make(map[string]interface{}),
	}

	// Try to send
	startTime := time.Now()
	err := w.manager.sendWebhookWithLog(ctx, webhook, event, log)
	duration := time.Since(startTime)
	log.Duration = duration

	if err != nil {
		// Check if we should retry again
		if w.retryPolicy.ShouldRetry(log.Attempts, err) {
			log.Status = DeliveryStatusRetrying
			nextRetry := w.retryPolicy.NextRetryTime(log.Attempts)
			log.NextRetryAt = &nextRetry
			log.ErrorMessage = err.Error()
		} else {
			// Max retries exceeded
			log.Status = DeliveryStatusFailed
			log.ErrorMessage = fmt.Sprintf("max retries exceeded: %v", err)
			now := time.Now()
			log.CompletedAt = &now
		}
	} else {
		// Success
		log.Status = DeliveryStatusSuccess
		log.ErrorMessage = ""
		now := time.Now()
		log.CompletedAt = &now
	}

	w.deliveryStore.Update(log)
}
