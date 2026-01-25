package webhooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookManager_RegisterWebhook(t *testing.T) {
	manager := NewWebhookManager()

	webhook := &Webhook{
		URL:    "https://example.com/webhook",
		Events: []EventType{EventModuleCreated, EventVersionCreated},
	}

	err := manager.RegisterWebhook(webhook)
	if err != nil {
		t.Fatalf("Failed to register webhook: %v", err)
	}

	if webhook.ID == "" {
		t.Error("Expected webhook ID to be set")
	}

	if !webhook.Active {
		t.Error("Expected webhook to be active")
	}
}

func TestWebhookManager_RegisterWebhook_Validation(t *testing.T) {
	manager := NewWebhookManager()

	t.Run("empty URL", func(t *testing.T) {
		webhook := &Webhook{
			Events: []EventType{EventModuleCreated},
		}

		err := manager.RegisterWebhook(webhook)
		if err == nil {
			t.Error("Expected error for empty URL")
		}
	})

	t.Run("no events", func(t *testing.T) {
		webhook := &Webhook{
			URL: "https://example.com/webhook",
		}

		err := manager.RegisterWebhook(webhook)
		if err == nil {
			t.Error("Expected error for no events")
		}
	})
}

func TestWebhookManager_UnregisterWebhook(t *testing.T) {
	manager := NewWebhookManager()

	webhook := &Webhook{
		URL:    "https://example.com/webhook",
		Events: []EventType{EventModuleCreated},
	}

	manager.RegisterWebhook(webhook)
	err := manager.UnregisterWebhook(webhook.ID)
	if err != nil {
		t.Fatalf("Failed to unregister webhook: %v", err)
	}

	_, err = manager.GetWebhook(webhook.ID)
	if err == nil {
		t.Error("Expected error getting unregistered webhook")
	}
}

func TestWebhookManager_UpdateWebhook(t *testing.T) {
	manager := NewWebhookManager()

	webhook := &Webhook{
		URL:    "https://example.com/webhook",
		Events: []EventType{EventModuleCreated},
	}

	manager.RegisterWebhook(webhook)

	updates := &Webhook{
		URL: "https://example.com/new-webhook",
	}

	err := manager.UpdateWebhook(webhook.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update webhook: %v", err)
	}

	updated, _ := manager.GetWebhook(webhook.ID)
	if updated.URL != "https://example.com/new-webhook" {
		t.Errorf("Expected URL to be updated, got %s", updated.URL)
	}
}

func TestWebhookManager_Dispatch(t *testing.T) {
	// Create a test server to receive webhook
	received := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("X-Spoke-Event") != string(EventModuleCreated) {
			t.Errorf("Expected event type %s", EventModuleCreated)
		}
		if r.Header.Get("X-Spoke-Event-ID") == "" {
			t.Error("Expected event ID header")
		}

		// Verify payload
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode event: %v", err)
		}

		if event.Type != EventModuleCreated {
			t.Errorf("Expected event type %s, got %s", EventModuleCreated, event.Type)
		}

		w.WriteHeader(http.StatusOK)
		received <- true
	}))
	defer server.Close()

	manager := NewWebhookManager()

	webhook := &Webhook{
		URL:    server.URL,
		Events: []EventType{EventModuleCreated},
	}

	manager.RegisterWebhook(webhook)

	event := &Event{
		Type: EventModuleCreated,
		Data: map[string]interface{}{
			"module": "test.module",
		},
	}

	err := manager.Dispatch(context.Background(), event)
	if err != nil {
		t.Fatalf("Failed to dispatch event: %v", err)
	}

	// Wait for webhook to be received
	select {
	case <-received:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Webhook was not received")
	}
}

func TestWebhookManager_Dispatch_FilterEvents(t *testing.T) {
	received := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewWebhookManager()

	// Register webhook only interested in module events
	webhook := &Webhook{
		URL:    server.URL,
		Events: []EventType{EventModuleCreated},
	}

	manager.RegisterWebhook(webhook)

	// Dispatch version event (should not be sent)
	event := &Event{
		Type: EventVersionCreated,
		Data: map[string]interface{}{},
	}

	manager.Dispatch(context.Background(), event)

	// Should not receive webhook
	select {
	case <-received:
		t.Error("Webhook should not have been sent for unsubscribed event")
	case <-time.After(500 * time.Millisecond):
		// Expected - no webhook sent
	}
}

func TestWebhookManager_ActivateDeactivate(t *testing.T) {
	manager := NewWebhookManager()

	webhook := &Webhook{
		URL:    "https://example.com/webhook",
		Events: []EventType{EventModuleCreated},
	}

	manager.RegisterWebhook(webhook)

	err := manager.DeactivateWebhook(webhook.ID)
	if err != nil {
		t.Fatalf("Failed to deactivate webhook: %v", err)
	}

	deactivated, _ := manager.GetWebhook(webhook.ID)
	if deactivated.Active {
		t.Error("Expected webhook to be inactive")
	}

	err = manager.ActivateWebhook(webhook.ID)
	if err != nil {
		t.Fatalf("Failed to activate webhook: %v", err)
	}

	activated, _ := manager.GetWebhook(webhook.ID)
	if !activated.Active {
		t.Error("Expected webhook to be active")
	}
}

func TestGenerateSignature(t *testing.T) {
	payload := []byte(`{"type":"module.created"}`)
	secret := "test-secret"

	signature := generateSignature(payload, secret)

	if signature == "" {
		t.Error("Expected signature to be generated")
	}

	if !VerifySignature(payload, signature, secret) {
		t.Error("Expected signature verification to succeed")
	}

	// Wrong secret should fail
	if VerifySignature(payload, signature, "wrong-secret") {
		t.Error("Expected signature verification to fail with wrong secret")
	}
}

func TestListWebhooks(t *testing.T) {
	manager := NewWebhookManager()

	// Initially should be empty
	webhooks := manager.ListWebhooks()
	if len(webhooks) != 0 {
		t.Fatalf("Expected 0 webhooks initially, got %d", len(webhooks))
	}

	// Register multiple webhooks
	registered := 0
	for range 3 {
		webhook := &Webhook{
			URL:    "https://example.com/webhook",
			Events: []EventType{EventModuleCreated},
		}
		if err := manager.RegisterWebhook(webhook); err != nil {
			t.Fatalf("Failed to register webhook: %v", err)
		}
		registered++
	}

	webhooks = manager.ListWebhooks()
	if len(webhooks) != registered {
		t.Errorf("Expected %d webhooks, got %d", registered, len(webhooks))
		for i, wh := range webhooks {
			t.Logf("Webhook %d: ID=%s, URL=%s", i, wh.ID, wh.URL)
		}
	}
}
