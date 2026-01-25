package webhooks

import (
	"encoding/json"
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
