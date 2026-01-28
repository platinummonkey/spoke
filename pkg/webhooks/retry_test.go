package webhooks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts to be 5, got %d", config.MaxAttempts)
	}
	if config.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay to be 1s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 5*time.Minute {
		t.Errorf("Expected MaxDelay to be 5m, got %v", config.MaxDelay)
	}
	if config.BackoffMultiplier != 2.0 {
		t.Errorf("Expected BackoffMultiplier to be 2.0, got %v", config.BackoffMultiplier)
	}
}

func TestNewRetryPolicy(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       3,
			InitialDelay:      2 * time.Second,
			MaxDelay:          10 * time.Minute,
			BackoffMultiplier: 1.5,
		}
		policy := NewRetryPolicy(config)

		if policy.config.MaxAttempts != 3 {
			t.Errorf("Expected MaxAttempts to be 3, got %d", policy.config.MaxAttempts)
		}
		if policy.config.InitialDelay != 2*time.Second {
			t.Errorf("Expected InitialDelay to be 2s, got %v", policy.config.InitialDelay)
		}
	})

	t.Run("zero max attempts uses default", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       0,
			InitialDelay:      1 * time.Second,
			MaxDelay:          5 * time.Minute,
			BackoffMultiplier: 2.0,
		}
		policy := NewRetryPolicy(config)

		if policy.config.MaxAttempts != 5 {
			t.Errorf("Expected MaxAttempts to default to 5, got %d", policy.config.MaxAttempts)
		}
	})

	t.Run("negative max attempts uses default", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       -1,
			InitialDelay:      1 * time.Second,
			MaxDelay:          5 * time.Minute,
			BackoffMultiplier: 2.0,
		}
		policy := NewRetryPolicy(config)

		if policy.config.MaxAttempts != 5 {
			t.Errorf("Expected MaxAttempts to default to 5, got %d", policy.config.MaxAttempts)
		}
	})

	t.Run("zero initial delay uses default", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       3,
			InitialDelay:      0,
			MaxDelay:          5 * time.Minute,
			BackoffMultiplier: 2.0,
		}
		policy := NewRetryPolicy(config)

		if policy.config.InitialDelay != 1*time.Second {
			t.Errorf("Expected InitialDelay to default to 1s, got %v", policy.config.InitialDelay)
		}
	})

	t.Run("negative initial delay uses default", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       3,
			InitialDelay:      -1 * time.Second,
			MaxDelay:          5 * time.Minute,
			BackoffMultiplier: 2.0,
		}
		policy := NewRetryPolicy(config)

		if policy.config.InitialDelay != 1*time.Second {
			t.Errorf("Expected InitialDelay to default to 1s, got %v", policy.config.InitialDelay)
		}
	})

	t.Run("zero max delay uses default", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       3,
			InitialDelay:      1 * time.Second,
			MaxDelay:          0,
			BackoffMultiplier: 2.0,
		}
		policy := NewRetryPolicy(config)

		if policy.config.MaxDelay != 5*time.Minute {
			t.Errorf("Expected MaxDelay to default to 5m, got %v", policy.config.MaxDelay)
		}
	})

	t.Run("negative max delay uses default", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       3,
			InitialDelay:      1 * time.Second,
			MaxDelay:          -1 * time.Minute,
			BackoffMultiplier: 2.0,
		}
		policy := NewRetryPolicy(config)

		if policy.config.MaxDelay != 5*time.Minute {
			t.Errorf("Expected MaxDelay to default to 5m, got %v", policy.config.MaxDelay)
		}
	})

	t.Run("backoff multiplier <= 1.0 uses default", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       3,
			InitialDelay:      1 * time.Second,
			MaxDelay:          5 * time.Minute,
			BackoffMultiplier: 1.0,
		}
		policy := NewRetryPolicy(config)

		if policy.config.BackoffMultiplier != 2.0 {
			t.Errorf("Expected BackoffMultiplier to default to 2.0, got %v", policy.config.BackoffMultiplier)
		}
	})

	t.Run("negative backoff multiplier uses default", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts:       3,
			InitialDelay:      1 * time.Second,
			MaxDelay:          5 * time.Minute,
			BackoffMultiplier: -1.0,
		}
		policy := NewRetryPolicy(config)

		if policy.config.BackoffMultiplier != 2.0 {
			t.Errorf("Expected BackoffMultiplier to default to 2.0, got %v", policy.config.BackoffMultiplier)
		}
	})
}

func TestRetryPolicy_ShouldRetry(t *testing.T) {
	policy := NewRetryPolicy(RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          5 * time.Minute,
		BackoffMultiplier: 2.0,
	})

	t.Run("no error should not retry", func(t *testing.T) {
		if policy.ShouldRetry(1, nil) {
			t.Error("Expected ShouldRetry to return false when err is nil")
		}
	})

	t.Run("within max attempts should retry", func(t *testing.T) {
		err := errors.New("test error")
		if !policy.ShouldRetry(1, err) {
			t.Error("Expected ShouldRetry to return true when attempts < max")
		}
		if !policy.ShouldRetry(2, err) {
			t.Error("Expected ShouldRetry to return true when attempts < max")
		}
	})

	t.Run("at max attempts should not retry", func(t *testing.T) {
		err := errors.New("test error")
		if policy.ShouldRetry(3, err) {
			t.Error("Expected ShouldRetry to return false when attempts >= max")
		}
	})

	t.Run("beyond max attempts should not retry", func(t *testing.T) {
		err := errors.New("test error")
		if policy.ShouldRetry(4, err) {
			t.Error("Expected ShouldRetry to return false when attempts > max")
		}
	})
}

func TestRetryPolicy_NextRetryDelay(t *testing.T) {
	policy := NewRetryPolicy(RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      1 * time.Second,
		MaxDelay:          1 * time.Minute,
		BackoffMultiplier: 2.0,
	})

	t.Run("zero attempts returns initial delay", func(t *testing.T) {
		delay := policy.NextRetryDelay(0)
		if delay != 1*time.Second {
			t.Errorf("Expected delay of 1s for 0 attempts, got %v", delay)
		}
	})

	t.Run("negative attempts returns initial delay", func(t *testing.T) {
		delay := policy.NextRetryDelay(-1)
		if delay != 1*time.Second {
			t.Errorf("Expected delay of 1s for negative attempts, got %v", delay)
		}
	})

	t.Run("exponential backoff for attempt 1", func(t *testing.T) {
		// delay = initialDelay * (multiplier ^ (attempts - 1))
		// For attempt 1: 1s * (2.0 ^ 0) = 1s
		delay := policy.NextRetryDelay(1)
		if delay != 1*time.Second {
			t.Errorf("Expected delay of 1s for attempt 1, got %v", delay)
		}
	})

	t.Run("exponential backoff for attempt 2", func(t *testing.T) {
		// For attempt 2: 1s * (2.0 ^ 1) = 2s
		delay := policy.NextRetryDelay(2)
		if delay != 2*time.Second {
			t.Errorf("Expected delay of 2s for attempt 2, got %v", delay)
		}
	})

	t.Run("exponential backoff for attempt 3", func(t *testing.T) {
		// For attempt 3: 1s * (2.0 ^ 2) = 4s
		delay := policy.NextRetryDelay(3)
		if delay != 4*time.Second {
			t.Errorf("Expected delay of 4s for attempt 3, got %v", delay)
		}
	})

	t.Run("delay capped at max delay", func(t *testing.T) {
		// For attempt 10: 1s * (2.0 ^ 9) = 512s > 60s max
		delay := policy.NextRetryDelay(10)
		if delay != 1*time.Minute {
			t.Errorf("Expected delay to be capped at 1m, got %v", delay)
		}
	})
}

func TestRetryPolicy_NextRetryTime(t *testing.T) {
	policy := NewRetryPolicy(RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      1 * time.Second,
		MaxDelay:          5 * time.Minute,
		BackoffMultiplier: 2.0,
	})

	before := time.Now()
	nextRetry := policy.NextRetryTime(1)
	after := time.Now()

	// Should be approximately 1 second in the future
	expectedMin := before.Add(1 * time.Second)
	expectedMax := after.Add(1 * time.Second)

	if nextRetry.Before(expectedMin) || nextRetry.After(expectedMax) {
		t.Errorf("Expected NextRetryTime to be around 1s in the future, got %v (now: %v)", nextRetry, before)
	}
}

func TestNewRetryWorker(t *testing.T) {
	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	if worker.manager != manager {
		t.Error("Expected worker manager to be set")
	}
	if worker.deliveryStore != deliveryStore {
		t.Error("Expected worker deliveryStore to be set")
	}
	if worker.retryPolicy != retryPolicy {
		t.Error("Expected worker retryPolicy to be set")
	}
	if worker.stopCh == nil {
		t.Error("Expected worker stopCh to be initialized")
	}
}

func TestRetryWorker_StartStop(t *testing.T) {
	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the worker
	worker.Start(ctx, 100*time.Millisecond)

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	if worker.ticker == nil {
		t.Error("Expected ticker to be initialized after Start")
	}

	// Stop the worker
	worker.Stop()

	// Give it a moment to stop
	time.Sleep(50 * time.Millisecond)
}

func TestRetryWorker_StopWithoutStart(t *testing.T) {
	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	// Should not panic when stopping without starting
	worker.Stop()
}

func TestRetryWorker_ContextCancellation(t *testing.T) {
	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	ctx, cancel := context.WithCancel(context.Background())

	// Start the worker
	worker.Start(ctx, 100*time.Millisecond)

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Give it a moment to stop
	time.Sleep(150 * time.Millisecond)

	// Worker should have stopped gracefully
}

func TestRetryWorker_ProcessRetries_NoRetries(t *testing.T) {
	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	ctx := context.Background()

	// Process retries when there are none
	worker.processRetries(ctx)

	// Should complete without error
}

func TestRetryWorker_ProcessRetries_WebhookNotFound(t *testing.T) {
	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	// Create a delivery log for a webhook that doesn't exist
	now := time.Now()
	nextRetry := now.Add(-1 * time.Second) // In the past so it's ready for retry
	deliveryLog := &DeliveryLog{
		ID:          "test-delivery-1",
		WebhookID:   "nonexistent-webhook",
		EventID:     "test-event-1",
		EventType:   EventModuleCreated,
		URL:         "https://example.com/webhook",
		Status:      DeliveryStatusRetrying,
		Attempts:    1,
		NextRetryAt: &nextRetry,
		CreatedAt:   now,
	}
	deliveryStore.Add(deliveryLog)

	ctx := context.Background()
	worker.processRetries(ctx)

	// Give it a moment to process
	time.Sleep(50 * time.Millisecond)

	// Check that the delivery log was marked as failed
	log, exists := deliveryStore.Get("test-delivery-1")
	if !exists {
		t.Fatal("Expected delivery log to exist")
	}
	if log.Status != DeliveryStatusFailed {
		t.Errorf("Expected status to be failed, got %v", log.Status)
	}
	if log.ErrorMessage == "" {
		t.Error("Expected error message to be set")
	}
	if log.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestRetryWorker_ProcessRetries_InactiveWebhook(t *testing.T) {
	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	// Create and register an inactive webhook
	webhook := &Webhook{
		URL:    "https://example.com/webhook",
		Events: []EventType{EventModuleCreated},
		Active: false,
	}
	manager.RegisterWebhook(webhook)
	webhook.Active = false // Deactivate after registration

	// Create a delivery log for this webhook
	now := time.Now()
	nextRetry := now.Add(-1 * time.Second) // In the past so it's ready for retry
	deliveryLog := &DeliveryLog{
		ID:          "test-delivery-2",
		WebhookID:   webhook.ID,
		EventID:     "test-event-2",
		EventType:   EventModuleCreated,
		URL:         webhook.URL,
		Status:      DeliveryStatusRetrying,
		Attempts:    1,
		NextRetryAt: &nextRetry,
		CreatedAt:   now,
	}
	deliveryStore.Add(deliveryLog)

	ctx := context.Background()
	worker.processRetries(ctx)

	// Give it a moment to process
	time.Sleep(50 * time.Millisecond)

	// Check that the delivery log was marked as failed
	log, exists := deliveryStore.Get("test-delivery-2")
	if !exists {
		t.Fatal("Expected delivery log to exist")
	}
	if log.Status != DeliveryStatusFailed {
		t.Errorf("Expected status to be failed, got %v", log.Status)
	}
	if log.ErrorMessage != "webhook is inactive" {
		t.Errorf("Expected error message 'webhook is inactive', got %v", log.ErrorMessage)
	}
	if log.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestRetryWorker_RetryDelivery_Success(t *testing.T) {
	// Create a test server that succeeds
	successCount := 0
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		successCount++
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(DefaultRetryConfig())

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	// Create and register a webhook
	webhook := &Webhook{
		URL:    server.URL,
		Events: []EventType{EventModuleCreated},
		Active: true,
	}
	manager.RegisterWebhook(webhook)

	// Create a delivery log
	now := time.Now()
	nextRetry := now.Add(-1 * time.Second)
	deliveryLog := &DeliveryLog{
		ID:          "test-delivery-success",
		WebhookID:   webhook.ID,
		EventID:     "test-event-success",
		EventType:   EventModuleCreated,
		URL:         webhook.URL,
		Status:      DeliveryStatusRetrying,
		Attempts:    1,
		NextRetryAt: &nextRetry,
		CreatedAt:   now,
	}
	deliveryStore.Add(deliveryLog)

	ctx := context.Background()
	worker.processRetries(ctx)

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// Check that the delivery succeeded
	log, exists := deliveryStore.Get("test-delivery-success")
	if !exists {
		t.Fatal("Expected delivery log to exist")
	}
	if log.Status != DeliveryStatusSuccess {
		t.Errorf("Expected status to be success, got %v (error: %v)", log.Status, log.ErrorMessage)
	}
	if log.Attempts != 2 {
		t.Errorf("Expected attempts to be 2, got %d", log.Attempts)
	}
	if log.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
	if log.ErrorMessage != "" {
		t.Errorf("Expected error message to be empty, got %v", log.ErrorMessage)
	}
	if successCount != 1 {
		t.Errorf("Expected 1 successful webhook call, got %d", successCount)
	}
}

func TestRetryWorker_RetryDelivery_FailureWithRetry(t *testing.T) {
	// Create a test server that fails
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      1 * time.Second,
		MaxDelay:          5 * time.Minute,
		BackoffMultiplier: 2.0,
	})

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	// Create and register a webhook
	webhook := &Webhook{
		URL:    server.URL,
		Events: []EventType{EventModuleCreated},
		Active: true,
	}
	manager.RegisterWebhook(webhook)

	// Create a delivery log with attempt count below max
	now := time.Now()
	nextRetry := now.Add(-1 * time.Second)
	deliveryLog := &DeliveryLog{
		ID:          "test-delivery-fail-retry",
		WebhookID:   webhook.ID,
		EventID:     "test-event-fail-retry",
		EventType:   EventModuleCreated,
		URL:         webhook.URL,
		Status:      DeliveryStatusRetrying,
		Attempts:    2, // Below max of 5
		NextRetryAt: &nextRetry,
		CreatedAt:   now,
	}
	deliveryStore.Add(deliveryLog)

	ctx := context.Background()
	worker.processRetries(ctx)

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// Check that it's still retrying
	log, exists := deliveryStore.Get("test-delivery-fail-retry")
	if !exists {
		t.Fatal("Expected delivery log to exist")
	}
	if log.Status != DeliveryStatusRetrying {
		t.Errorf("Expected status to be retrying, got %v", log.Status)
	}
	if log.Attempts != 3 {
		t.Errorf("Expected attempts to be 3, got %d", log.Attempts)
	}
	if log.NextRetryAt == nil {
		t.Error("Expected NextRetryAt to be set")
	}
	if log.ErrorMessage == "" {
		t.Error("Expected error message to be set")
	}
	if log.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil for retrying status")
	}
}

func TestRetryWorker_RetryDelivery_MaxRetriesExceeded(t *testing.T) {
	// Create a test server that fails
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          5 * time.Minute,
		BackoffMultiplier: 2.0,
	})

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	// Create and register a webhook
	webhook := &Webhook{
		URL:    server.URL,
		Events: []EventType{EventModuleCreated},
		Active: true,
	}
	manager.RegisterWebhook(webhook)

	// Create a delivery log at max attempts
	now := time.Now()
	nextRetry := now.Add(-1 * time.Second)
	deliveryLog := &DeliveryLog{
		ID:          "test-delivery-max-retry",
		WebhookID:   webhook.ID,
		EventID:     "test-event-max-retry",
		EventType:   EventModuleCreated,
		URL:         webhook.URL,
		Status:      DeliveryStatusRetrying,
		Attempts:    2, // Will become 3 after retry, which equals max
		NextRetryAt: &nextRetry,
		CreatedAt:   now,
	}
	deliveryStore.Add(deliveryLog)

	ctx := context.Background()
	worker.processRetries(ctx)

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// Check that it failed after max retries
	log, exists := deliveryStore.Get("test-delivery-max-retry")
	if !exists {
		t.Fatal("Expected delivery log to exist")
	}
	if log.Status != DeliveryStatusFailed {
		t.Errorf("Expected status to be failed, got %v", log.Status)
	}
	if log.Attempts != 3 {
		t.Errorf("Expected attempts to be 3, got %d", log.Attempts)
	}
	if log.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
	if log.ErrorMessage == "" {
		t.Error("Expected error message to be set")
	}
}

func TestRetryWorker_Integration(t *testing.T) {
	// Integration test for the full retry workflow
	callCount := 0
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Fail first 2 times, succeed on 3rd
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})
	defer server.Close()

	manager := NewWebhookManager()
	deliveryStore := NewDeliveryLogStore(100)
	retryPolicy := NewRetryPolicy(RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      10 * time.Millisecond, // Short delays for testing
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	})

	worker := NewRetryWorker(manager, deliveryStore, retryPolicy)

	// Create and register a webhook
	webhook := &Webhook{
		URL:    server.URL,
		Events: []EventType{EventModuleCreated},
		Active: true,
	}
	manager.RegisterWebhook(webhook)

	// Start the worker with a short check interval
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx, 50*time.Millisecond)

	// Create initial delivery log
	now := time.Now()
	nextRetry := now.Add(-1 * time.Second) // Ready immediately
	deliveryLog := &DeliveryLog{
		ID:          "test-delivery-integration",
		WebhookID:   webhook.ID,
		EventID:     "test-event-integration",
		EventType:   EventModuleCreated,
		URL:         webhook.URL,
		Status:      DeliveryStatusRetrying,
		Attempts:    1,
		NextRetryAt: &nextRetry,
		CreatedAt:   now,
	}
	deliveryStore.Add(deliveryLog)

	// Wait for retries to complete
	time.Sleep(500 * time.Millisecond)

	// Check final status
	log, exists := deliveryStore.Get("test-delivery-integration")
	if !exists {
		t.Fatal("Expected delivery log to exist")
	}

	// Should have succeeded after 2 retries (total 3 calls)
	if log.Status != DeliveryStatusSuccess {
		t.Errorf("Expected status to be success, got %v (attempts: %d, error: %v)", log.Status, log.Attempts, log.ErrorMessage)
	}
	if callCount < 2 {
		t.Errorf("Expected at least 2 webhook calls, got %d", callCount)
	}

	worker.Stop()
}

// Helper function to create a test HTTP server
func createTestServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handler))
}
