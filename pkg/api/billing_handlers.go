package api

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/billing"
	"github.com/platinummonkey/spoke/pkg/httputil"
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
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req billing.CreateSubscriptionRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	subscription, err := h.billingService.CreateSubscription(orgID, &req)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w, subscription)
}

// GetSubscription retrieves a subscription
func (h *BillingHandlers) GetSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	subscription, err := h.billingService.GetSubscription(orgID)
	if err != nil {
		httputil.WriteNotFoundError(w, err.Error())
		return
	}

	httputil.WriteSuccess(w, subscription)
}

// UpdateSubscription updates a subscription
func (h *BillingHandlers) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req billing.UpdateSubscriptionRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	subscription, err := h.billingService.UpdateSubscription(orgID, &req)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, subscription)
}

// CancelSubscription cancels a subscription
func (h *BillingHandlers) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req struct {
		Immediately bool `json:"immediately"`
	}
	// Default to false if no body or parsing fails
	httputil.ParseJSON(r, &req)

	if err := h.billingService.CancelSubscription(orgID, req.Immediately); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// ReactivateSubscription reactivates a canceled subscription
func (h *BillingHandlers) ReactivateSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	subscription, err := h.billingService.ReactivateSubscription(orgID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, subscription)
}

// GetInvoice retrieves an invoice
func (h *BillingHandlers) GetInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := httputil.ParsePathInt64OrError(w, r, "invoice_id")
	if !ok {
		return
	}

	invoice, err := h.billingService.GetInvoice(invoiceID)
	if err != nil {
		httputil.WriteNotFoundError(w, err.Error())
		return
	}

	httputil.WriteSuccess(w, invoice)
}

// ListInvoices lists invoices for an organization
func (h *BillingHandlers) ListInvoices(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	limit, err := httputil.ParseQueryInt(r, "limit", 100)
	if err != nil {
		httputil.WriteBadRequest(w, err.Error())
		return
	}

	invoices, err := h.billingService.ListInvoices(orgID, limit)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, invoices)
}

// GenerateInvoice generates an invoice for an organization
func (h *BillingHandlers) GenerateInvoice(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	invoice, err := h.billingService.GenerateInvoice(orgID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w, invoice)
}

// AddPaymentMethod adds a payment method
func (h *BillingHandlers) AddPaymentMethod(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req billing.CreatePaymentMethodRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	paymentMethod, err := h.billingService.AddPaymentMethod(orgID, &req)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w, paymentMethod)
}

// ListPaymentMethods lists payment methods
func (h *BillingHandlers) ListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	methods, err := h.billingService.ListPaymentMethods(orgID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, methods)
}

// SetDefaultPaymentMethod sets a payment method as default
func (h *BillingHandlers) SetDefaultPaymentMethod(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	pmID, ok := httputil.ParsePathInt64OrError(w, r, "pm_id")
	if !ok {
		return
	}

	if err := h.billingService.SetDefaultPaymentMethod(orgID, pmID); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// RemovePaymentMethod removes a payment method
func (h *BillingHandlers) RemovePaymentMethod(w http.ResponseWriter, r *http.Request) {
	orgID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	pmID, ok := httputil.ParsePathInt64OrError(w, r, "pm_id")
	if !ok {
		return
	}

	if err := h.billingService.RemovePaymentMethod(orgID, pmID); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// HandleWebhook handles Stripe webhook events
func (h *BillingHandlers) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.WriteBadRequest(w, "Failed to read request body")
		return
	}

	// Get the signature header
	signature := r.Header.Get("Stripe-Signature")

	// Handle the webhook
	if err := h.billingService.HandleWebhook(payload, signature); err != nil {
		httputil.WriteBadRequest(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}
