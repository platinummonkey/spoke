package webhooks

import (
	"testing"
	"time"
)

func TestNewDeliveryLogStore(t *testing.T) {
	tests := []struct {
		name     string
		maxLogs  int
		expected int
	}{
		{
			name:     "positive max logs",
			maxLogs:  500,
			expected: 500,
		},
		{
			name:     "zero max logs defaults to 1000",
			maxLogs:  0,
			expected: 1000,
		},
		{
			name:     "negative max logs defaults to 1000",
			maxLogs:  -10,
			expected: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewDeliveryLogStore(tt.maxLogs)
			if store == nil {
				t.Fatal("expected non-nil store")
			}
			if store.maxLogs != tt.expected {
				t.Errorf("expected maxLogs=%d, got %d", tt.expected, store.maxLogs)
			}
			if store.logs == nil {
				t.Error("expected logs map to be initialized")
			}
		})
	}
}

func TestDeliveryLogStore_Add(t *testing.T) {
	store := NewDeliveryLogStore(10)

	log := &DeliveryLog{
		ID:        "log1",
		WebhookID: "webhook1",
		EventID:   "event1",
		EventType: EventModuleCreated,
		URL:       "http://example.com/webhook",
		Status:    DeliveryStatusPending,
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	store.Add(log)

	retrieved, exists := store.Get("log1")
	if !exists {
		t.Error("expected log to exist after adding")
	}
	if retrieved.ID != "log1" {
		t.Errorf("expected ID=log1, got %s", retrieved.ID)
	}
}

func TestDeliveryLogStore_Get(t *testing.T) {
	store := NewDeliveryLogStore(10)

	log := &DeliveryLog{
		ID:        "log1",
		WebhookID: "webhook1",
		CreatedAt: time.Now(),
	}
	store.Add(log)

	t.Run("existing log", func(t *testing.T) {
		retrieved, exists := store.Get("log1")
		if !exists {
			t.Error("expected log to exist")
		}
		if retrieved.ID != "log1" {
			t.Errorf("expected ID=log1, got %s", retrieved.ID)
		}
	})

	t.Run("non-existing log", func(t *testing.T) {
		_, exists := store.Get("nonexistent")
		if exists {
			t.Error("expected log to not exist")
		}
	})
}

func TestDeliveryLogStore_Update(t *testing.T) {
	store := NewDeliveryLogStore(10)

	log := &DeliveryLog{
		ID:        "log1",
		WebhookID: "webhook1",
		Status:    DeliveryStatusPending,
		CreatedAt: time.Now(),
	}
	store.Add(log)

	// Update the log
	log.Status = DeliveryStatusSuccess
	log.StatusCode = 200
	store.Update(log)

	retrieved, _ := store.Get("log1")
	if retrieved.Status != DeliveryStatusSuccess {
		t.Errorf("expected status=success, got %s", retrieved.Status)
	}
	if retrieved.StatusCode != 200 {
		t.Errorf("expected status code=200, got %d", retrieved.StatusCode)
	}
}

func TestDeliveryLogStore_GetByWebhook(t *testing.T) {
	store := NewDeliveryLogStore(100)

	now := time.Now()

	// Add logs for different webhooks with different timestamps
	logs := []*DeliveryLog{
		{
			ID:        "log1",
			WebhookID: "webhook1",
			CreatedAt: now.Add(-3 * time.Hour),
		},
		{
			ID:        "log2",
			WebhookID: "webhook1",
			CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "log3",
			WebhookID: "webhook2",
			CreatedAt: now,
		},
		{
			ID:        "log4",
			WebhookID: "webhook1",
			CreatedAt: now.Add(-2 * time.Hour),
		},
	}

	for _, log := range logs {
		store.Add(log)
	}

	t.Run("get all logs for webhook1", func(t *testing.T) {
		results := store.GetByWebhook("webhook1", 0)
		if len(results) != 3 {
			t.Errorf("expected 3 logs, got %d", len(results))
		}
		// Check if sorted by created_at descending
		if len(results) > 1 && results[0].CreatedAt.Before(results[1].CreatedAt) {
			t.Error("expected logs to be sorted by created_at descending")
		}
	})

	t.Run("get limited logs", func(t *testing.T) {
		results := store.GetByWebhook("webhook1", 2)
		if len(results) != 2 {
			t.Errorf("expected 2 logs, got %d", len(results))
		}
	})

	t.Run("webhook with no logs", func(t *testing.T) {
		results := store.GetByWebhook("nonexistent", 0)
		if len(results) != 0 {
			t.Errorf("expected 0 logs, got %d", len(results))
		}
	})
}

func TestDeliveryLogStore_GetByEvent(t *testing.T) {
	store := NewDeliveryLogStore(100)

	logs := []*DeliveryLog{
		{
			ID:        "log1",
			EventID:   "event1",
			WebhookID: "webhook1",
			CreatedAt: time.Now(),
		},
		{
			ID:        "log2",
			EventID:   "event1",
			WebhookID: "webhook2",
			CreatedAt: time.Now(),
		},
		{
			ID:        "log3",
			EventID:   "event2",
			WebhookID: "webhook1",
			CreatedAt: time.Now(),
		},
	}

	for _, log := range logs {
		store.Add(log)
	}

	t.Run("get logs for event1", func(t *testing.T) {
		results := store.GetByEvent("event1")
		if len(results) != 2 {
			t.Errorf("expected 2 logs, got %d", len(results))
		}
	})

	t.Run("get logs for event2", func(t *testing.T) {
		results := store.GetByEvent("event2")
		if len(results) != 1 {
			t.Errorf("expected 1 log, got %d", len(results))
		}
	})

	t.Run("event with no logs", func(t *testing.T) {
		results := store.GetByEvent("nonexistent")
		if len(results) != 0 {
			t.Errorf("expected 0 logs, got %d", len(results))
		}
	})
}

func TestDeliveryLogStore_GetPendingRetries(t *testing.T) {
	store := NewDeliveryLogStore(100)

	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(1 * time.Hour)

	logs := []*DeliveryLog{
		{
			ID:          "log1",
			Status:      DeliveryStatusRetrying,
			NextRetryAt: &pastTime, // Should be returned
			CreatedAt:   time.Now(),
		},
		{
			ID:          "log2",
			Status:      DeliveryStatusRetrying,
			NextRetryAt: &futureTime, // Should not be returned (future)
			CreatedAt:   time.Now(),
		},
		{
			ID:        "log3",
			Status:    DeliveryStatusSuccess, // Should not be returned (not retrying)
			CreatedAt: time.Now(),
		},
		{
			ID:        "log4",
			Status:    DeliveryStatusRetrying, // Should not be returned (no NextRetryAt)
			CreatedAt: time.Now(),
		},
	}

	for _, log := range logs {
		store.Add(log)
	}

	results := store.GetPendingRetries()
	if len(results) != 1 {
		t.Errorf("expected 1 pending retry, got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != "log1" {
		t.Errorf("expected log1, got %s", results[0].ID)
	}
}

func TestDeliveryLogStore_GetStats(t *testing.T) {
	store := NewDeliveryLogStore(100)

	completedTime := time.Now()

	logs := []*DeliveryLog{
		{
			ID:          "log1",
			WebhookID:   "webhook1",
			Status:      DeliveryStatusSuccess,
			Duration:    100 * time.Millisecond,
			CompletedAt: &completedTime,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "log2",
			WebhookID:   "webhook1",
			Status:      DeliveryStatusSuccess,
			Duration:    200 * time.Millisecond,
			CompletedAt: &completedTime,
			CreatedAt:   time.Now(),
		},
		{
			ID:        "log3",
			WebhookID: "webhook1",
			Status:    DeliveryStatusFailed,
			CreatedAt: time.Now(),
		},
		{
			ID:        "log4",
			WebhookID: "webhook1",
			Status:    DeliveryStatusRetrying,
			CreatedAt: time.Now(),
		},
		{
			ID:        "log5",
			WebhookID: "webhook2",
			Status:    DeliveryStatusSuccess,
			CreatedAt: time.Now(),
		},
	}

	for _, log := range logs {
		store.Add(log)
	}

	stats := store.GetStats("webhook1")

	if stats.Total != 4 {
		t.Errorf("expected total=4, got %d", stats.Total)
	}
	if stats.Successful != 2 {
		t.Errorf("expected successful=2, got %d", stats.Successful)
	}
	if stats.Failed != 1 {
		t.Errorf("expected failed=1, got %d", stats.Failed)
	}
	if stats.Retrying != 1 {
		t.Errorf("expected retrying=1, got %d", stats.Retrying)
	}
	if stats.SuccessRate != 0.5 {
		t.Errorf("expected success rate=0.5, got %f", stats.SuccessRate)
	}
	// Average duration should be (100 + 200) / 2 = 150ms
	expectedAvg := 150 * time.Millisecond
	if stats.AverageDuration != expectedAvg {
		t.Errorf("expected average duration=%v, got %v", expectedAvg, stats.AverageDuration)
	}
}

func TestDeliveryLogStore_GetStats_NoLogs(t *testing.T) {
	store := NewDeliveryLogStore(100)

	stats := store.GetStats("webhook1")

	if stats.Total != 0 {
		t.Errorf("expected total=0, got %d", stats.Total)
	}
	if stats.SuccessRate != 0 {
		t.Errorf("expected success rate=0, got %f", stats.SuccessRate)
	}
}

func TestDeliveryLogStore_EvictOldest(t *testing.T) {
	store := NewDeliveryLogStore(10)

	now := time.Now()

	// Add enough logs to trigger eviction
	for i := 0; i < 12; i++ {
		log := &DeliveryLog{
			ID:        string(rune('a' + i)),
			WebhookID: "webhook1",
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		store.Add(log)
	}

	// After adding 12 logs with maxLogs=10, eviction should have occurred
	// The oldest log should have been removed
	_, exists := store.Get("a")
	if exists {
		t.Error("expected oldest log to be evicted")
	}

	// Newer logs should still exist
	_, exists = store.Get("k") // 11th log (index 10)
	if !exists {
		t.Error("expected newer log to still exist")
	}
}

func TestDeliveryLogStore_ConcurrentAccess(t *testing.T) {
	store := NewDeliveryLogStore(100)

	// Test concurrent writes and reads
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 10; i++ {
			log := &DeliveryLog{
				ID:        string(rune('a' + i)),
				WebhookID: "webhook1",
				CreatedAt: time.Now(),
			}
			store.Add(log)
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 10; i++ {
			store.Get("a")
			store.GetByWebhook("webhook1", 0)
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// If we get here without a race condition, the test passes
}

func TestDeliveryStatus_Constants(t *testing.T) {
	// Test that all status constants are defined correctly
	statuses := []DeliveryStatus{
		DeliveryStatusPending,
		DeliveryStatusSuccess,
		DeliveryStatusFailed,
		DeliveryStatusRetrying,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("expected non-empty status constant")
		}
	}
}
