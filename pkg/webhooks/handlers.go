package webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// WebhookHandlers provides HTTP handlers for webhook management
type WebhookHandlers struct {
	manager *WebhookManager
}

// NewWebhookHandlers creates new webhook handlers
func NewWebhookHandlers(manager *WebhookManager) *WebhookHandlers {
	return &WebhookHandlers{
		manager: manager,
	}
}

// RegisterRoutes registers webhook routes
func (h *WebhookHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/webhooks", h.createWebhook).Methods("POST")
	router.HandleFunc("/webhooks", h.listWebhooks).Methods("GET")
	router.HandleFunc("/webhooks/{id}", h.getWebhook).Methods("GET")
	router.HandleFunc("/webhooks/{id}", h.updateWebhook).Methods("PUT")
	router.HandleFunc("/webhooks/{id}", h.deleteWebhook).Methods("DELETE")
	router.HandleFunc("/webhooks/{id}/activate", h.activateWebhook).Methods("POST")
	router.HandleFunc("/webhooks/{id}/deactivate", h.deactivateWebhook).Methods("POST")
	router.HandleFunc("/webhooks/{id}/test", h.testWebhook).Methods("POST")
	router.HandleFunc("/webhooks/{id}/deliveries", h.getDeliveryLogs).Methods("GET")
	router.HandleFunc("/webhooks/{id}/stats", h.getDeliveryStats).Methods("GET")
}

// createWebhook handles POST /webhooks
func (h *WebhookHandlers) createWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.manager.RegisterWebhook(&webhook); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(webhook)
}

// listWebhooks handles GET /webhooks
func (h *WebhookHandlers) listWebhooks(w http.ResponseWriter, r *http.Request) {
	webhooks := h.manager.ListWebhooks()
	json.NewEncoder(w).Encode(webhooks)
}

// getWebhook handles GET /webhooks/{id}
func (h *WebhookHandlers) getWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	webhook, err := h.manager.GetWebhook(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(webhook)
}

// updateWebhook handles PUT /webhooks/{id}
func (h *WebhookHandlers) updateWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updates Webhook
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.manager.UpdateWebhook(id, &updates); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	webhook, _ := h.manager.GetWebhook(id)
	json.NewEncoder(w).Encode(webhook)
}

// deleteWebhook handles DELETE /webhooks/{id}
func (h *WebhookHandlers) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.manager.UnregisterWebhook(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// activateWebhook handles POST /webhooks/{id}/activate
func (h *WebhookHandlers) activateWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.manager.ActivateWebhook(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	webhook, _ := h.manager.GetWebhook(id)
	json.NewEncoder(w).Encode(webhook)
}

// deactivateWebhook handles POST /webhooks/{id}/deactivate
func (h *WebhookHandlers) deactivateWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.manager.DeactivateWebhook(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	webhook, _ := h.manager.GetWebhook(id)
	json.NewEncoder(w).Encode(webhook)
}

// testWebhook handles POST /webhooks/{id}/test
func (h *WebhookHandlers) testWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := h.manager.GetWebhook(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Create a test event
	testEvent := &Event{
		Type: "webhook.test",
		Data: map[string]interface{}{
			"message": "This is a test webhook delivery from Spoke Schema Registry",
			"webhook_id": id,
		},
	}

	// Send the test event
	err = h.manager.Dispatch(r.Context(), testEvent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Test webhook sent successfully",
		"event_id": testEvent.ID,
	})
}

// getDeliveryLogs handles GET /webhooks/{id}/deliveries
func (h *WebhookHandlers) getDeliveryLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := h.manager.GetWebhook(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get limit from query params (default: 50)
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	logs := h.manager.GetDeliveryLogs(id, limit)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"webhook_id": id,
		"deliveries": logs,
		"count":      len(logs),
	})
}

// getDeliveryStats handles GET /webhooks/{id}/stats
func (h *WebhookHandlers) getDeliveryStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := h.manager.GetWebhook(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	stats := h.manager.GetDeliveryStats(id)
	json.NewEncoder(w).Encode(stats)
}
