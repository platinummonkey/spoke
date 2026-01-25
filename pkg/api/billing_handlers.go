package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/billing"
)

// BillingHandlers handles billing-related HTTP requests
type BillingHandlers struct {
	billingService billing.Service
}

// NewBillingHandlers creates a new BillingHandlers
func NewBillingHandlers(billingService billing.Service) *BillingHandlers {
	return &BillingHandlers{
		billingService: billingService,
	}
}

// RegisterRoutes registers billing routes
func (h *BillingHandlers) RegisterRoutes(router *mux.Router) {
	// Subscriptions
	router.HandleFunc("/orgs/{id}/subscription", h.CreateSubscription).Methods("POST")
	router.HandleFunc("/orgs/{id}/subscription", h.GetSubscription).Methods("GET")
	router.HandleFunc("/orgs/{id}/subscription", h.UpdateSubscription).Methods("PUT")
	router.HandleFunc("/orgs/{id}/subscription/cancel", h.CancelSubscription).Methods("POST")
	router.HandleFunc("/orgs/{id}/subscription/reactivate", h.ReactivateSubscription).Methods("POST")

	// Invoices
	router.HandleFunc("/orgs/{id}/invoices", h.ListInvoices).Methods("GET")
	router.HandleFunc("/invoices/{invoice_id}", h.GetInvoice).Methods("GET")
	router.HandleFunc("/orgs/{id}/invoices/generate", h.GenerateInvoice).Methods("POST")

	// Payment methods
	router.HandleFunc("/orgs/{id}/payment-methods", h.AddPaymentMethod).Methods("POST")
	router.HandleFunc("/orgs/{id}/payment-methods", h.ListPaymentMethods).Methods("GET")
	router.HandleFunc("/orgs/{id}/payment-methods/{pm_id}/default", h.SetDefaultPaymentMethod).Methods("PUT")
	router.HandleFunc("/orgs/{id}/payment-methods/{pm_id}", h.RemovePaymentMethod).Methods("DELETE")

	// Webhooks
	router.HandleFunc("/billing/webhook", h.HandleWebhook).Methods("POST")
}

// CreateSubscription creates a new subscription
func (h *BillingHandlers) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	var req billing.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	subscription, err := h.billingService.CreateSubscription(orgID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(subscription)
}

// GetSubscription retrieves a subscription
func (h *BillingHandlers) GetSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	subscription, err := h.billingService.GetSubscription(orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

// UpdateSubscription updates a subscription
func (h *BillingHandlers) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	var req billing.UpdateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	subscription, err := h.billingService.UpdateSubscription(orgID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

// CancelSubscription cancels a subscription
func (h *BillingHandlers) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Immediately bool `json:"immediately"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to false if no body
		req.Immediately = false
	}

	if err := h.billingService.CancelSubscription(orgID, req.Immediately); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReactivateSubscription reactivates a canceled subscription
func (h *BillingHandlers) ReactivateSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	subscription, err := h.billingService.ReactivateSubscription(orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

// GetInvoice retrieves an invoice
func (h *BillingHandlers) GetInvoice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invoiceID, err := strconv.ParseInt(vars["invoice_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	invoice, err := h.billingService.GetInvoice(invoiceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invoice)
}

// ListInvoices lists invoices for an organization
func (h *BillingHandlers) ListInvoices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	invoices, err := h.billingService.ListInvoices(orgID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invoices)
}

// GenerateInvoice generates an invoice for an organization
func (h *BillingHandlers) GenerateInvoice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	invoice, err := h.billingService.GenerateInvoice(orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(invoice)
}

// AddPaymentMethod adds a payment method
func (h *BillingHandlers) AddPaymentMethod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	var req billing.CreatePaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	paymentMethod, err := h.billingService.AddPaymentMethod(orgID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(paymentMethod)
}

// ListPaymentMethods lists payment methods
func (h *BillingHandlers) ListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	methods, err := h.billingService.ListPaymentMethods(orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(methods)
}

// SetDefaultPaymentMethod sets a payment method as default
func (h *BillingHandlers) SetDefaultPaymentMethod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	pmID, err := strconv.ParseInt(vars["pm_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid payment method ID", http.StatusBadRequest)
		return
	}

	if err := h.billingService.SetDefaultPaymentMethod(orgID, pmID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemovePaymentMethod removes a payment method
func (h *BillingHandlers) RemovePaymentMethod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	pmID, err := strconv.ParseInt(vars["pm_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid payment method ID", http.StatusBadRequest)
		return
	}

	if err := h.billingService.RemovePaymentMethod(orgID, pmID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleWebhook handles Stripe webhook events
func (h *BillingHandlers) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Get the signature header
	signature := r.Header.Get("Stripe-Signature")

	// Handle the webhook
	if err := h.billingService.HandleWebhook(payload, signature); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
