package billing

import (
	"time"

	"github.com/platinummonkey/spoke/pkg/orgs"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive     SubscriptionStatus = "active"
	SubscriptionStatusCanceled   SubscriptionStatus = "canceled"
	SubscriptionStatusPastDue    SubscriptionStatus = "past_due"
	SubscriptionStatusIncomplete SubscriptionStatus = "incomplete"
	SubscriptionStatusTrialing   SubscriptionStatus = "trialing"
)

// Subscription represents a billing subscription
type Subscription struct {
	ID                   int64              `json:"id"`
	OrgID                int64              `json:"org_id"`
	Plan                 orgs.PlanTier      `json:"plan"`
	StripeCustomerID     string             `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty"`
	Status               SubscriptionStatus `json:"status"`
	CurrentPeriodStart   *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end,omitempty"`
	CancelAt             *time.Time         `json:"cancel_at,omitempty"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
	TrialStart           *time.Time         `json:"trial_start,omitempty"`
	TrialEnd             *time.Time         `json:"trial_end,omitempty"`
	Metadata             map[string]any     `json:"metadata,omitempty"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// InvoiceStatus represents the status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusDraft          InvoiceStatus = "draft"
	InvoiceStatusOpen           InvoiceStatus = "open"
	InvoiceStatusPaid           InvoiceStatus = "paid"
	InvoiceStatusVoid           InvoiceStatus = "void"
	InvoiceStatusUncollectible  InvoiceStatus = "uncollectible"
)

// Invoice represents a billing invoice
type Invoice struct {
	ID              int64          `json:"id"`
	OrgID           int64          `json:"org_id"`
	InvoiceNumber   string         `json:"invoice_number,omitempty"`
	StripeInvoiceID string         `json:"stripe_invoice_id,omitempty"`
	AmountCents     int64          `json:"amount_cents"`
	Currency        string         `json:"currency"`
	PeriodStart     time.Time      `json:"period_start"`
	PeriodEnd       time.Time      `json:"period_end"`
	Status          InvoiceStatus  `json:"status"`
	PaidAt          *time.Time     `json:"paid_at,omitempty"`
	DueDate         *time.Time     `json:"due_date,omitempty"`
	InvoiceURL      string         `json:"invoice_url,omitempty"`
	InvoicePDFURL   string         `json:"invoice_pdf_url,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// PaymentMethodType represents the type of payment method
type PaymentMethodType string

const (
	PaymentMethodTypeCard        PaymentMethodType = "card"
	PaymentMethodTypeBankAccount PaymentMethodType = "bank_account"
)

// PaymentMethod represents a payment method
type PaymentMethod struct {
	ID                      int64             `json:"id"`
	OrgID                   int64             `json:"org_id"`
	StripePaymentMethodID   string            `json:"stripe_payment_method_id"`
	Type                    PaymentMethodType `json:"type"`
	IsDefault               bool              `json:"is_default"`
	CardBrand               string            `json:"card_brand,omitempty"`
	CardLast4               string            `json:"card_last4,omitempty"`
	CardExpMonth            int               `json:"card_exp_month,omitempty"`
	CardExpYear             int               `json:"card_exp_year,omitempty"`
	BankName                string            `json:"bank_name,omitempty"`
	BankLast4               string            `json:"bank_last4,omitempty"`
	Metadata                map[string]any    `json:"metadata,omitempty"`
	CreatedAt               time.Time         `json:"created_at"`
	UpdatedAt               time.Time         `json:"updated_at"`
}

// CreateSubscriptionRequest represents request to create a subscription
type CreateSubscriptionRequest struct {
	Plan              orgs.PlanTier `json:"plan"`
	PaymentMethodID   string        `json:"payment_method_id,omitempty"`
	TrialPeriodDays   int           `json:"trial_period_days,omitempty"`
}

// UpdateSubscriptionRequest represents request to update a subscription
type UpdateSubscriptionRequest struct {
	Plan   *orgs.PlanTier `json:"plan,omitempty"`
	CancelAtPeriodEnd bool `json:"cancel_at_period_end,omitempty"`
}

// CreatePaymentMethodRequest represents request to add a payment method
type CreatePaymentMethodRequest struct {
	StripePaymentMethodID string `json:"stripe_payment_method_id"`
	SetAsDefault          bool   `json:"set_as_default"`
}

// StripeWebhookEvent represents a Stripe webhook event
type StripeWebhookEvent struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Data    map[string]any `json:"data"`
	Created int64          `json:"created"`
}

// Service defines the interface for billing operations
type Service interface {
	// Subscription management
	CreateSubscription(orgID int64, req *CreateSubscriptionRequest) (*Subscription, error)
	GetSubscription(orgID int64) (*Subscription, error)
	UpdateSubscription(orgID int64, req *UpdateSubscriptionRequest) (*Subscription, error)
	CancelSubscription(orgID int64, immediately bool) error
	ReactivateSubscription(orgID int64) (*Subscription, error)

	// Invoice management
	GetInvoice(id int64) (*Invoice, error)
	ListInvoices(orgID int64, limit int) ([]*Invoice, error)
	GenerateInvoice(orgID int64) (*Invoice, error)

	// Payment method management
	AddPaymentMethod(orgID int64, req *CreatePaymentMethodRequest) (*PaymentMethod, error)
	ListPaymentMethods(orgID int64) ([]*PaymentMethod, error)
	SetDefaultPaymentMethod(orgID int64, paymentMethodID int64) error
	RemovePaymentMethod(orgID int64, paymentMethodID int64) error

	// Stripe integration
	CreateStripeCustomer(orgID int64) (string, error)
	GetStripeCustomer(orgID int64) (string, error)
	HandleWebhook(payload []byte, signature string) error

	// Usage-based billing
	RecordUsage(orgID int64, usage *orgs.OrgUsage) error
	CalculateBill(orgID int64, periodStart, periodEnd time.Time) (int64, error)
}

// PlanPricing defines pricing for subscription plans
type PlanPricing struct {
	Plan                orgs.PlanTier
	BasePriceCents      int64   // Base monthly price
	StoragePricePerGB   int64   // Price per GB over quota
	CompileJobPrice     int64   // Price per compile job over quota
	APIRequestPrice     int64   // Price per 1000 API requests over quota
	IncludedStorage     int64   // Included storage in bytes
	IncludedCompileJobs int     // Included compile jobs per month
	IncludedAPIRequests int64   // Included API requests per month
}

// DefaultPlanPricing returns default pricing for each plan
func DefaultPlanPricing() map[orgs.PlanTier]PlanPricing {
	return map[orgs.PlanTier]PlanPricing{
		orgs.PlanFree: {
			Plan:                orgs.PlanFree,
			BasePriceCents:      0,
			StoragePricePerGB:   0,
			CompileJobPrice:     0,
			APIRequestPrice:     0,
			IncludedStorage:     1 * 1024 * 1024 * 1024, // 1GB
			IncludedCompileJobs: 100,
			IncludedAPIRequests: 1000 * 60, // 1000/hour * 60 hours
		},
		orgs.PlanPro: {
			Plan:                orgs.PlanPro,
			BasePriceCents:      4900, // $49/month
			StoragePricePerGB:   500,  // $5/GB over quota
			CompileJobPrice:     5,    // $0.05 per job over quota
			APIRequestPrice:     10,   // $0.10 per 1000 requests over quota
			IncludedStorage:     10 * 1024 * 1024 * 1024, // 10GB
			IncludedCompileJobs: 1000,
			IncludedAPIRequests: 10000 * 60, // 10000/hour * 60 hours
		},
		orgs.PlanEnterprise: {
			Plan:                orgs.PlanEnterprise,
			BasePriceCents:      49900, // $499/month
			StoragePricePerGB:   300,   // $3/GB over quota
			CompileJobPrice:     3,     // $0.03 per job over quota
			APIRequestPrice:     5,     // $0.05 per 1000 requests over quota
			IncludedStorage:     100 * 1024 * 1024 * 1024, // 100GB
			IncludedCompileJobs: 10000,
			IncludedAPIRequests: 100000 * 60, // 100000/hour * 60 hours
		},
	}
}
