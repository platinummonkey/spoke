package billing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/platinummonkey/spoke/pkg/orgs"
)

// PostgresService implements the billing Service interface using PostgreSQL
type PostgresService struct {
	db            *sql.DB
	stripeAPIKey  string
	stripeWebhookSecret string
	orgService    orgs.Service
}

// NewPostgresService creates a new PostgresService
func NewPostgresService(db *sql.DB, stripeAPIKey, stripeWebhookSecret string, orgService orgs.Service) *PostgresService {
	return &PostgresService{
		db:                  db,
		stripeAPIKey:        stripeAPIKey,
		stripeWebhookSecret: stripeWebhookSecret,
		orgService:          orgService,
	}
}

// CreateSubscription creates a new subscription for an organization
func (s *PostgresService) CreateSubscription(orgID int64, req *CreateSubscriptionRequest) (*Subscription, error) {
	// Get or create Stripe customer
	customerID, err := s.GetStripeCustomer(orgID)
	if err != nil {
		customerID, err = s.CreateStripeCustomer(orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to create stripe customer: %w", err)
		}
	}

	// For now, create a basic subscription record
	// Full Stripe integration would use the Stripe API here
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // 1 month from now

	query := `
		INSERT INTO subscriptions (org_id, plan, stripe_customer_id, status, current_period_start, current_period_end)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (org_id) DO UPDATE
		SET plan = EXCLUDED.plan, status = EXCLUDED.status,
		    current_period_start = EXCLUDED.current_period_start,
		    current_period_end = EXCLUDED.current_period_end
		RETURNING id, created_at, updated_at
	`
	sub := &Subscription{
		OrgID:              orgID,
		Plan:               req.Plan,
		StripeCustomerID:   customerID,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &periodEnd,
	}

	err = s.db.QueryRow(query, sub.OrgID, sub.Plan, sub.StripeCustomerID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd).
		Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Update organization plan tier
	org, err := s.orgService.GetOrganization(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	displayName := org.DisplayName
	s.orgService.UpdateOrganization(orgID, &orgs.UpdateOrgRequest{
		DisplayName: &displayName,
	})

	// Update quotas based on new plan
	quotas := s.orgService.GetDefaultQuotas(req.Plan)
	quotas.OrgID = orgID
	if err := s.orgService.UpdateQuotas(orgID, quotas); err != nil {
		return nil, fmt.Errorf("failed to update quotas: %w", err)
	}

	return sub, nil
}

// GetSubscription retrieves the subscription for an organization
func (s *PostgresService) GetSubscription(orgID int64) (*Subscription, error) {
	query := `
		SELECT id, org_id, plan, stripe_customer_id, stripe_subscription_id, status,
		       current_period_start, current_period_end, cancel_at, canceled_at,
		       trial_start, trial_end, metadata, created_at, updated_at
		FROM subscriptions
		WHERE org_id = $1
	`
	sub := &Subscription{}
	var metadataJSON []byte
	err := s.db.QueryRow(query, orgID).Scan(
		&sub.ID, &sub.OrgID, &sub.Plan, &sub.StripeCustomerID, &sub.StripeSubscriptionID,
		&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CancelAt,
		&sub.CanceledAt, &sub.TrialStart, &sub.TrialEnd, &metadataJSON,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return sub, nil
}

// UpdateSubscription updates a subscription
func (s *PostgresService) UpdateSubscription(orgID int64, req *UpdateSubscriptionRequest) (*Subscription, error) {
	if req.Plan != nil {
		query := `UPDATE subscriptions SET plan = $1 WHERE org_id = $2`
		if _, err := s.db.Exec(query, *req.Plan, orgID); err != nil {
			return nil, fmt.Errorf("failed to update subscription: %w", err)
		}

		// Update quotas based on new plan
		quotas := s.orgService.GetDefaultQuotas(*req.Plan)
		quotas.OrgID = orgID
		if err := s.orgService.UpdateQuotas(orgID, quotas); err != nil {
			return nil, fmt.Errorf("failed to update quotas: %w", err)
		}
	}

	if req.CancelAtPeriodEnd {
		query := `UPDATE subscriptions SET cancel_at = current_period_end WHERE org_id = $1`
		if _, err := s.db.Exec(query, orgID); err != nil {
			return nil, fmt.Errorf("failed to schedule cancellation: %w", err)
		}
	}

	return s.GetSubscription(orgID)
}

// CancelSubscription cancels a subscription
func (s *PostgresService) CancelSubscription(orgID int64, immediately bool) error {
	if immediately {
		query := `UPDATE subscriptions SET status = $1, canceled_at = NOW() WHERE org_id = $2`
		if _, err := s.db.Exec(query, SubscriptionStatusCanceled, orgID); err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}

		// Downgrade to small tier
		quotas := s.orgService.GetDefaultQuotas(orgs.QuotaTierSmall)
		quotas.OrgID = orgID
		if err := s.orgService.UpdateQuotas(orgID, quotas); err != nil {
			return fmt.Errorf("failed to update quotas: %w", err)
		}
	} else {
		query := `UPDATE subscriptions SET cancel_at = current_period_end WHERE org_id = $1`
		if _, err := s.db.Exec(query, orgID); err != nil {
			return fmt.Errorf("failed to schedule cancellation: %w", err)
		}
	}

	return nil
}

// ReactivateSubscription reactivates a canceled subscription
func (s *PostgresService) ReactivateSubscription(orgID int64) (*Subscription, error) {
	query := `UPDATE subscriptions SET status = $1, cancel_at = NULL, canceled_at = NULL WHERE org_id = $2`
	if _, err := s.db.Exec(query, SubscriptionStatusActive, orgID); err != nil {
		return nil, fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	return s.GetSubscription(orgID)
}

// GetInvoice retrieves an invoice by ID
func (s *PostgresService) GetInvoice(id int64) (*Invoice, error) {
	query := `
		SELECT id, org_id, invoice_number, stripe_invoice_id, amount_cents, currency,
		       period_start, period_end, status, paid_at, due_date,
		       invoice_url, invoice_pdf_url, metadata, created_at, updated_at
		FROM invoices
		WHERE id = $1
	`
	invoice := &Invoice{}
	var metadataJSON []byte
	err := s.db.QueryRow(query, id).Scan(
		&invoice.ID, &invoice.OrgID, &invoice.InvoiceNumber, &invoice.StripeInvoiceID,
		&invoice.AmountCents, &invoice.Currency, &invoice.PeriodStart, &invoice.PeriodEnd,
		&invoice.Status, &invoice.PaidAt, &invoice.DueDate, &invoice.InvoiceURL,
		&invoice.InvoicePDFURL, &metadataJSON, &invoice.CreatedAt, &invoice.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invoice not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &invoice.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return invoice, nil
}

// ListInvoices lists invoices for an organization
func (s *PostgresService) ListInvoices(orgID int64, limit int) ([]*Invoice, error) {
	query := `
		SELECT id, org_id, invoice_number, stripe_invoice_id, amount_cents, currency,
		       period_start, period_end, status, paid_at, due_date,
		       invoice_url, invoice_pdf_url, metadata, created_at, updated_at
		FROM invoices
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.Query(query, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*Invoice
	for rows.Next() {
		invoice := &Invoice{}
		var metadataJSON []byte
		if err := rows.Scan(
			&invoice.ID, &invoice.OrgID, &invoice.InvoiceNumber, &invoice.StripeInvoiceID,
			&invoice.AmountCents, &invoice.Currency, &invoice.PeriodStart, &invoice.PeriodEnd,
			&invoice.Status, &invoice.PaidAt, &invoice.DueDate, &invoice.InvoiceURL,
			&invoice.InvoicePDFURL, &metadataJSON, &invoice.CreatedAt, &invoice.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan invoice: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &invoice.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		invoices = append(invoices, invoice)
	}

	return invoices, nil
}

// GenerateInvoice generates an invoice for an organization
func (s *PostgresService) GenerateInvoice(orgID int64) (*Invoice, error) {
	// Get usage for the period
	usage, err := s.orgService.GetUsage(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	// Calculate bill
	amount, err := s.CalculateBill(orgID, usage.PeriodStart, usage.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate bill: %w", err)
	}

	// Create invoice
	dueDate := time.Now().AddDate(0, 0, 30) // 30 days from now
	query := `
		INSERT INTO invoices (org_id, amount_cents, currency, period_start, period_end, status, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	invoice := &Invoice{
		OrgID:       orgID,
		AmountCents: amount,
		Currency:    "usd",
		PeriodStart: usage.PeriodStart,
		PeriodEnd:   usage.PeriodEnd,
		Status:      InvoiceStatusOpen,
		DueDate:     &dueDate,
	}

	err = s.db.QueryRow(query, invoice.OrgID, invoice.AmountCents, invoice.Currency,
		invoice.PeriodStart, invoice.PeriodEnd, invoice.Status, invoice.DueDate).
		Scan(&invoice.ID, &invoice.CreatedAt, &invoice.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	return invoice, nil
}

// Continue in next file for payment methods and Stripe integration...
