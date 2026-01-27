package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/billing"
	"github.com/platinummonkey/spoke/pkg/orgs"
	"github.com/stretchr/testify/assert"
)

// mockBillingService implements billing.Service for testing
type mockBillingService struct {
	createSubscriptionFunc    func(orgID int64, req *billing.CreateSubscriptionRequest) (*billing.Subscription, error)
	getSubscriptionFunc       func(orgID int64) (*billing.Subscription, error)
	updateSubscriptionFunc    func(orgID int64, req *billing.UpdateSubscriptionRequest) (*billing.Subscription, error)
	cancelSubscriptionFunc    func(orgID int64, immediately bool) error
	reactivateSubscriptionFunc func(orgID int64) (*billing.Subscription, error)
	getInvoiceFunc            func(id int64) (*billing.Invoice, error)
	listInvoicesFunc          func(orgID int64, limit int) ([]*billing.Invoice, error)
	generateInvoiceFunc       func(orgID int64) (*billing.Invoice, error)
	addPaymentMethodFunc      func(orgID int64, req *billing.CreatePaymentMethodRequest) (*billing.PaymentMethod, error)
	listPaymentMethodsFunc    func(orgID int64) ([]*billing.PaymentMethod, error)
	setDefaultPaymentMethodFunc func(orgID int64, paymentMethodID int64) error
	removePaymentMethodFunc   func(orgID int64, paymentMethodID int64) error
	createStripeCustomerFunc  func(orgID int64) (string, error)
	getStripeCustomerFunc     func(orgID int64) (string, error)
	handleWebhookFunc         func(payload []byte, signature string) error
	recordUsageFunc           func(orgID int64, usage *orgs.OrgUsage) error
	calculateBillFunc         func(orgID int64, periodStart, periodEnd time.Time) (int64, error)
}

func (m *mockBillingService) CreateSubscription(orgID int64, req *billing.CreateSubscriptionRequest) (*billing.Subscription, error) {
	if m.createSubscriptionFunc != nil {
		return m.createSubscriptionFunc(orgID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) GetSubscription(orgID int64) (*billing.Subscription, error) {
	if m.getSubscriptionFunc != nil {
		return m.getSubscriptionFunc(orgID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) UpdateSubscription(orgID int64, req *billing.UpdateSubscriptionRequest) (*billing.Subscription, error) {
	if m.updateSubscriptionFunc != nil {
		return m.updateSubscriptionFunc(orgID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) CancelSubscription(orgID int64, immediately bool) error {
	if m.cancelSubscriptionFunc != nil {
		return m.cancelSubscriptionFunc(orgID, immediately)
	}
	return errors.New("not implemented")
}

func (m *mockBillingService) ReactivateSubscription(orgID int64) (*billing.Subscription, error) {
	if m.reactivateSubscriptionFunc != nil {
		return m.reactivateSubscriptionFunc(orgID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) GetInvoice(id int64) (*billing.Invoice, error) {
	if m.getInvoiceFunc != nil {
		return m.getInvoiceFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) ListInvoices(orgID int64, limit int) ([]*billing.Invoice, error) {
	if m.listInvoicesFunc != nil {
		return m.listInvoicesFunc(orgID, limit)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) GenerateInvoice(orgID int64) (*billing.Invoice, error) {
	if m.generateInvoiceFunc != nil {
		return m.generateInvoiceFunc(orgID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) AddPaymentMethod(orgID int64, req *billing.CreatePaymentMethodRequest) (*billing.PaymentMethod, error) {
	if m.addPaymentMethodFunc != nil {
		return m.addPaymentMethodFunc(orgID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) ListPaymentMethods(orgID int64) ([]*billing.PaymentMethod, error) {
	if m.listPaymentMethodsFunc != nil {
		return m.listPaymentMethodsFunc(orgID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBillingService) SetDefaultPaymentMethod(orgID int64, paymentMethodID int64) error {
	if m.setDefaultPaymentMethodFunc != nil {
		return m.setDefaultPaymentMethodFunc(orgID, paymentMethodID)
	}
	return errors.New("not implemented")
}

func (m *mockBillingService) RemovePaymentMethod(orgID int64, paymentMethodID int64) error {
	if m.removePaymentMethodFunc != nil {
		return m.removePaymentMethodFunc(orgID, paymentMethodID)
	}
	return errors.New("not implemented")
}

func (m *mockBillingService) CreateStripeCustomer(orgID int64) (string, error) {
	if m.createStripeCustomerFunc != nil {
		return m.createStripeCustomerFunc(orgID)
	}
	return "", errors.New("not implemented")
}

func (m *mockBillingService) GetStripeCustomer(orgID int64) (string, error) {
	if m.getStripeCustomerFunc != nil {
		return m.getStripeCustomerFunc(orgID)
	}
	return "", errors.New("not implemented")
}

func (m *mockBillingService) HandleWebhook(payload []byte, signature string) error {
	if m.handleWebhookFunc != nil {
		return m.handleWebhookFunc(payload, signature)
	}
	return errors.New("not implemented")
}

func (m *mockBillingService) RecordUsage(orgID int64, usage *orgs.OrgUsage) error {
	if m.recordUsageFunc != nil {
		return m.recordUsageFunc(orgID, usage)
	}
	return errors.New("not implemented")
}

func (m *mockBillingService) CalculateBill(orgID int64, periodStart, periodEnd time.Time) (int64, error) {
	if m.calculateBillFunc != nil {
		return m.calculateBillFunc(orgID, periodStart, periodEnd)
	}
	return 0, errors.New("not implemented")
}

// TestNewBillingHandlers verifies handler initialization
func TestNewBillingHandlers(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.billingService)
}

// TestBillingHandlers_RegisterRoutes verifies all routes are registered
func TestBillingHandlers_RegisterRoutes(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/orgs/1/subscription"},
		{"GET", "/orgs/1/subscription"},
		{"PUT", "/orgs/1/subscription"},
		{"POST", "/orgs/1/subscription/cancel"},
		{"POST", "/orgs/1/subscription/reactivate"},
		{"GET", "/orgs/1/invoices"},
		{"GET", "/invoices/1"},
		{"POST", "/orgs/1/invoices/generate"},
		{"POST", "/orgs/1/payment-methods"},
		{"GET", "/orgs/1/payment-methods"},
		{"PUT", "/orgs/1/payment-methods/1/default"},
		{"DELETE", "/orgs/1/payment-methods/1"},
		{"POST", "/billing/webhook"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			var match mux.RouteMatch
			matched := router.Match(req, &match)
			assert.True(t, matched, "Route %s %s should be registered", tt.method, tt.path)
		})
	}
}

// TestCreateSubscription_InvalidOrgID tests with invalid organization ID
func TestCreateSubscription_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/invalid/subscription", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.CreateSubscription(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateSubscription_InvalidJSON tests with invalid JSON body
func TestCreateSubscription_InvalidJSON(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/1/subscription", bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CreateSubscription(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateSubscription_ServiceError tests service error handling
func TestCreateSubscription_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		createSubscriptionFunc: func(orgID int64, req *billing.CreateSubscriptionRequest) (*billing.Subscription, error) {
			return nil, errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	reqBody, _ := json.Marshal(billing.CreateSubscriptionRequest{
		Plan: "pro",
	})
	req := httptest.NewRequest("POST", "/orgs/1/subscription", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CreateSubscription(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCreateSubscription_Success tests successful subscription creation
func TestCreateSubscription_Success(t *testing.T) {
	mockService := &mockBillingService{
		createSubscriptionFunc: func(orgID int64, req *billing.CreateSubscriptionRequest) (*billing.Subscription, error) {
			return &billing.Subscription{
				ID:     1,
				OrgID:  orgID,
				Status: "active",
			}, nil
		},
	}
	handlers := NewBillingHandlers(mockService)

	reqBody, _ := json.Marshal(billing.CreateSubscriptionRequest{
		Plan: "pro",
	})
	req := httptest.NewRequest("POST", "/orgs/1/subscription", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CreateSubscription(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// TestGetSubscription_InvalidOrgID tests with invalid organization ID
func TestGetSubscription_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/orgs/invalid/subscription", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.GetSubscription(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGetSubscription_NotFound tests when subscription not found
func TestGetSubscription_NotFound(t *testing.T) {
	mockService := &mockBillingService{
		getSubscriptionFunc: func(orgID int64) (*billing.Subscription, error) {
			return nil, errors.New("not found")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/orgs/1/subscription", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.GetSubscription(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetSubscription_Success tests successful retrieval
func TestGetSubscription_Success(t *testing.T) {
	mockService := &mockBillingService{
		getSubscriptionFunc: func(orgID int64) (*billing.Subscription, error) {
			return &billing.Subscription{
				ID:     1,
				OrgID:  orgID,
				Status: "active",
			}, nil
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/orgs/1/subscription", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.GetSubscription(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestUpdateSubscription_InvalidOrgID tests with invalid organization ID
func TestUpdateSubscription_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("PUT", "/orgs/invalid/subscription", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.UpdateSubscription(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateSubscription_InvalidJSON tests with invalid JSON body
func TestUpdateSubscription_InvalidJSON(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("PUT", "/orgs/1/subscription", bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.UpdateSubscription(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateSubscription_ServiceError tests service error handling
func TestUpdateSubscription_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		updateSubscriptionFunc: func(orgID int64, req *billing.UpdateSubscriptionRequest) (*billing.Subscription, error) {
			return nil, errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	tier := orgs.QuotaTierLarge
	reqBody, _ := json.Marshal(billing.UpdateSubscriptionRequest{
		Plan: &tier,
	})
	req := httptest.NewRequest("PUT", "/orgs/1/subscription", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.UpdateSubscription(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCancelSubscription_InvalidOrgID tests with invalid organization ID
func TestCancelSubscription_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/invalid/subscription/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.CancelSubscription(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCancelSubscription_ServiceError tests service error handling
func TestCancelSubscription_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		cancelSubscriptionFunc: func(orgID int64, immediately bool) error {
			return errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/1/subscription/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CancelSubscription(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCancelSubscription_Success tests successful cancellation
func TestCancelSubscription_Success(t *testing.T) {
	mockService := &mockBillingService{
		cancelSubscriptionFunc: func(orgID int64, immediately bool) error {
			return nil
		},
	}
	handlers := NewBillingHandlers(mockService)

	reqBody, _ := json.Marshal(map[string]bool{"immediately": true})
	req := httptest.NewRequest("POST", "/orgs/1/subscription/cancel", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CancelSubscription(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestReactivateSubscription_InvalidOrgID tests with invalid organization ID
func TestReactivateSubscription_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/invalid/subscription/reactivate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.ReactivateSubscription(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestReactivateSubscription_ServiceError tests service error handling
func TestReactivateSubscription_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		reactivateSubscriptionFunc: func(orgID int64) (*billing.Subscription, error) {
			return nil, errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/1/subscription/reactivate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.ReactivateSubscription(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGetInvoice_InvalidInvoiceID tests with invalid invoice ID
func TestGetInvoice_InvalidInvoiceID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/invoices/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"invoice_id": "invalid"})
	w := httptest.NewRecorder()

	handlers.GetInvoice(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGetInvoice_NotFound tests when invoice not found
func TestGetInvoice_NotFound(t *testing.T) {
	mockService := &mockBillingService{
		getInvoiceFunc: func(id int64) (*billing.Invoice, error) {
			return nil, errors.New("not found")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/invoices/1", nil)
	req = mux.SetURLVars(req, map[string]string{"invoice_id": "1"})
	w := httptest.NewRecorder()

	handlers.GetInvoice(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestListInvoices_InvalidOrgID tests with invalid organization ID
func TestListInvoices_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/orgs/invalid/invoices", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.ListInvoices(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListInvoices_InvalidLimit tests with invalid limit parameter
func TestListInvoices_InvalidLimit(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/orgs/1/invoices?limit=invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.ListInvoices(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListInvoices_ServiceError tests service error handling
func TestListInvoices_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		listInvoicesFunc: func(orgID int64, limit int) ([]*billing.Invoice, error) {
			return nil, errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/orgs/1/invoices", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.ListInvoices(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGenerateInvoice_InvalidOrgID tests with invalid organization ID
func TestGenerateInvoice_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/invalid/invoices/generate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.GenerateInvoice(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGenerateInvoice_ServiceError tests service error handling
func TestGenerateInvoice_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		generateInvoiceFunc: func(orgID int64) (*billing.Invoice, error) {
			return nil, errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/1/invoices/generate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.GenerateInvoice(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAddPaymentMethod_InvalidOrgID tests with invalid organization ID
func TestAddPaymentMethod_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/invalid/payment-methods", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.AddPaymentMethod(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAddPaymentMethod_InvalidJSON tests with invalid JSON body
func TestAddPaymentMethod_InvalidJSON(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/orgs/1/payment-methods", bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.AddPaymentMethod(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAddPaymentMethod_ServiceError tests service error handling
func TestAddPaymentMethod_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		addPaymentMethodFunc: func(orgID int64, req *billing.CreatePaymentMethodRequest) (*billing.PaymentMethod, error) {
			return nil, errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	reqBody, _ := json.Marshal(billing.CreatePaymentMethodRequest{
		StripePaymentMethodID: "pm_visa",
	})
	req := httptest.NewRequest("POST", "/orgs/1/payment-methods", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.AddPaymentMethod(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestListPaymentMethods_InvalidOrgID tests with invalid organization ID
func TestListPaymentMethods_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/orgs/invalid/payment-methods", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.ListPaymentMethods(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListPaymentMethods_ServiceError tests service error handling
func TestListPaymentMethods_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		listPaymentMethodsFunc: func(orgID int64) ([]*billing.PaymentMethod, error) {
			return nil, errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("GET", "/orgs/1/payment-methods", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.ListPaymentMethods(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestSetDefaultPaymentMethod_InvalidOrgID tests with invalid organization ID
func TestSetDefaultPaymentMethod_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("PUT", "/orgs/invalid/payment-methods/1/default", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid", "pm_id": "1"})
	w := httptest.NewRecorder()

	handlers.SetDefaultPaymentMethod(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSetDefaultPaymentMethod_InvalidPaymentMethodID tests with invalid payment method ID
func TestSetDefaultPaymentMethod_InvalidPaymentMethodID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("PUT", "/orgs/1/payment-methods/invalid/default", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1", "pm_id": "invalid"})
	w := httptest.NewRecorder()

	handlers.SetDefaultPaymentMethod(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSetDefaultPaymentMethod_ServiceError tests service error handling
func TestSetDefaultPaymentMethod_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		setDefaultPaymentMethodFunc: func(orgID int64, paymentMethodID int64) error {
			return errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("PUT", "/orgs/1/payment-methods/1/default", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1", "pm_id": "1"})
	w := httptest.NewRecorder()

	handlers.SetDefaultPaymentMethod(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestRemovePaymentMethod_InvalidOrgID tests with invalid organization ID
func TestRemovePaymentMethod_InvalidOrgID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("DELETE", "/orgs/invalid/payment-methods/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid", "pm_id": "1"})
	w := httptest.NewRecorder()

	handlers.RemovePaymentMethod(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRemovePaymentMethod_InvalidPaymentMethodID tests with invalid payment method ID
func TestRemovePaymentMethod_InvalidPaymentMethodID(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("DELETE", "/orgs/1/payment-methods/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1", "pm_id": "invalid"})
	w := httptest.NewRecorder()

	handlers.RemovePaymentMethod(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRemovePaymentMethod_ServiceError tests service error handling
func TestRemovePaymentMethod_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		removePaymentMethodFunc: func(orgID int64, paymentMethodID int64) error {
			return errors.New("service error")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("DELETE", "/orgs/1/payment-methods/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1", "pm_id": "1"})
	w := httptest.NewRecorder()

	handlers.RemovePaymentMethod(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestHandleWebhook_EmptyBody tests with empty request body
func TestHandleWebhook_EmptyBody(t *testing.T) {
	mockService := &mockBillingService{}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/billing/webhook", nil)
	w := httptest.NewRecorder()

	handlers.HandleWebhook(w, req)

	// Empty body is valid - should pass to service
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}

// TestHandleWebhook_ServiceError tests service error handling
func TestHandleWebhook_ServiceError(t *testing.T) {
	mockService := &mockBillingService{
		handleWebhookFunc: func(payload []byte, signature string) error {
			return errors.New("invalid signature")
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/billing/webhook", bytes.NewBufferString(`{"type":"payment.success"}`))
	req.Header.Set("Stripe-Signature", "invalid-sig")
	w := httptest.NewRecorder()

	handlers.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleWebhook_Success tests successful webhook handling
func TestHandleWebhook_Success(t *testing.T) {
	mockService := &mockBillingService{
		handleWebhookFunc: func(payload []byte, signature string) error {
			return nil
		},
	}
	handlers := NewBillingHandlers(mockService)

	req := httptest.NewRequest("POST", "/billing/webhook", bytes.NewBufferString(`{"type":"payment.success"}`))
	req.Header.Set("Stripe-Signature", "valid-sig")
	w := httptest.NewRecorder()

	handlers.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Benchmark tests

func BenchmarkCreateSubscription(b *testing.B) {
	mockService := &mockBillingService{
		createSubscriptionFunc: func(orgID int64, req *billing.CreateSubscriptionRequest) (*billing.Subscription, error) {
			return &billing.Subscription{ID: 1, OrgID: orgID, Status: "active"}, nil
		},
	}
	handlers := NewBillingHandlers(mockService)

	reqBody, _ := json.Marshal(billing.CreateSubscriptionRequest{Plan: "pro"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/orgs/1/subscription", bytes.NewBuffer(reqBody))
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		w := httptest.NewRecorder()
		handlers.CreateSubscription(w, req)
	}
}

func BenchmarkListInvoices(b *testing.B) {
	mockService := &mockBillingService{
		listInvoicesFunc: func(orgID int64, limit int) ([]*billing.Invoice, error) {
			return []*billing.Invoice{{ID: 1, OrgID: orgID}}, nil
		},
	}
	handlers := NewBillingHandlers(mockService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/orgs/1/invoices", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		w := httptest.NewRecorder()
		handlers.ListInvoices(w, req)
	}
}
