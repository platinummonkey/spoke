package billing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/platinummonkey/spoke/pkg/orgs"
)

// CreateStripeCustomer creates a Stripe customer for an organization
func (s *PostgresService) CreateStripeCustomer(orgID int64) (string, error) {
	// In a real implementation, this would call the Stripe API
	// For now, we'll generate a mock customer ID
	customerID := fmt.Sprintf("cus_mock_%d", orgID)

	query := `UPDATE subscriptions SET stripe_customer_id = $1 WHERE org_id = $2`
	_, err := s.db.Exec(query, customerID, orgID)
	if err != nil {
		return "", fmt.Errorf("failed to update customer ID: %w", err)
	}

	return customerID, nil
}

// GetStripeCustomer retrieves the Stripe customer ID for an organization
func (s *PostgresService) GetStripeCustomer(orgID int64) (string, error) {
	query := `SELECT stripe_customer_id FROM subscriptions WHERE org_id = $1`
	var customerID sql.NullString
	err := s.db.QueryRow(query, orgID).Scan(&customerID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("subscription not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get customer ID: %w", err)
	}
	if !customerID.Valid {
		return "", fmt.Errorf("customer ID not set")
	}

	return customerID.String, nil
}

// HandleWebhook handles a Stripe webhook event
func (s *PostgresService) HandleWebhook(payload []byte, signature string) error {
	// In a real implementation, this would:
	// 1. Verify the webhook signature using Stripe SDK
	// 2. Parse the event
	// 3. Handle different event types

	var event StripeWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	switch event.Type {
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(event)
	case "invoice.paid":
		return s.handleInvoicePaid(event)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(event)
	default:
		// Unknown event type, ignore
		return nil
	}
}

func (s *PostgresService) handleSubscriptionCreated(event StripeWebhookEvent) error {
	// Extract subscription data from event
	// Update database accordingly
	return nil
}

func (s *PostgresService) handleSubscriptionUpdated(event StripeWebhookEvent) error {
	// Update subscription status in database
	return nil
}

func (s *PostgresService) handleSubscriptionDeleted(event StripeWebhookEvent) error {
	// Mark subscription as canceled
	return nil
}

func (s *PostgresService) handleInvoicePaid(event StripeWebhookEvent) error {
	// Mark invoice as paid
	return nil
}

func (s *PostgresService) handleInvoicePaymentFailed(event StripeWebhookEvent) error {
	// Handle payment failure
	return nil
}

// AddPaymentMethod adds a payment method for an organization
func (s *PostgresService) AddPaymentMethod(orgID int64, req *CreatePaymentMethodRequest) (*PaymentMethod, error) {
	// In a real implementation, this would attach the payment method to the Stripe customer
	query := `
		INSERT INTO payment_methods (org_id, stripe_payment_method_id, type, is_default)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	pm := &PaymentMethod{
		OrgID:                 orgID,
		StripePaymentMethodID: req.StripePaymentMethodID,
		Type:                  PaymentMethodTypeCard, // Default to card
		IsDefault:             req.SetAsDefault,
	}

	err := s.db.QueryRow(query, pm.OrgID, pm.StripePaymentMethodID, pm.Type, pm.IsDefault).
		Scan(&pm.ID, &pm.CreatedAt, &pm.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to add payment method: %w", err)
	}

	return pm, nil
}

// ListPaymentMethods lists payment methods for an organization
func (s *PostgresService) ListPaymentMethods(orgID int64) ([]*PaymentMethod, error) {
	query := `
		SELECT id, org_id, stripe_payment_method_id, type, is_default,
		       card_brand, card_last4, card_exp_month, card_exp_year,
		       bank_name, bank_last4, metadata, created_at, updated_at
		FROM payment_methods
		WHERE org_id = $1
		ORDER BY is_default DESC, created_at DESC
	`
	rows, err := s.db.Query(query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}
	defer rows.Close()

	var methods []*PaymentMethod
	for rows.Next() {
		pm := &PaymentMethod{}
		var metadataJSON []byte
		var cardBrand, cardLast4, bankName, bankLast4 sql.NullString
		var cardExpMonth, cardExpYear sql.NullInt64

		if err := rows.Scan(
			&pm.ID, &pm.OrgID, &pm.StripePaymentMethodID, &pm.Type, &pm.IsDefault,
			&cardBrand, &cardLast4, &cardExpMonth, &cardExpYear,
			&bankName, &bankLast4, &metadataJSON, &pm.CreatedAt, &pm.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan payment method: %w", err)
		}

		if cardBrand.Valid {
			pm.CardBrand = cardBrand.String
		}
		if cardLast4.Valid {
			pm.CardLast4 = cardLast4.String
		}
		if cardExpMonth.Valid {
			pm.CardExpMonth = int(cardExpMonth.Int64)
		}
		if cardExpYear.Valid {
			pm.CardExpYear = int(cardExpYear.Int64)
		}
		if bankName.Valid {
			pm.BankName = bankName.String
		}
		if bankLast4.Valid {
			pm.BankLast4 = bankLast4.String
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &pm.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		methods = append(methods, pm)
	}

	return methods, nil
}

// SetDefaultPaymentMethod sets a payment method as default
func (s *PostgresService) SetDefaultPaymentMethod(orgID int64, paymentMethodID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Unset all other default payment methods
	query := `UPDATE payment_methods SET is_default = false WHERE org_id = $1`
	if _, err := tx.Exec(query, orgID); err != nil {
		return fmt.Errorf("failed to unset default: %w", err)
	}

	// Set the new default
	query = `UPDATE payment_methods SET is_default = true WHERE id = $1 AND org_id = $2`
	result, err := tx.Exec(query, paymentMethodID, orgID)
	if err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("payment method not found")
	}

	return tx.Commit()
}

// RemovePaymentMethod removes a payment method
func (s *PostgresService) RemovePaymentMethod(orgID int64, paymentMethodID int64) error {
	query := `DELETE FROM payment_methods WHERE id = $1 AND org_id = $2`
	result, err := s.db.Exec(query, paymentMethodID, orgID)
	if err != nil {
		return fmt.Errorf("failed to remove payment method: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("payment method not found")
	}

	return nil
}

// RecordUsage records usage for billing purposes
func (s *PostgresService) RecordUsage(orgID int64, usage *orgs.OrgUsage) error {
	// This is already handled by the orgs service
	// This method is here for completeness
	return nil
}

// CalculateBill calculates the bill for an organization for a given period
func (s *PostgresService) CalculateBill(orgID int64, periodStart, periodEnd time.Time) (int64, error) {
	// Get subscription
	sub, err := s.GetSubscription(orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get usage
	usage, err := s.orgService.GetUsage(orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to get usage: %w", err)
	}

	// Get quotas
	quotas, err := s.orgService.GetQuotas(orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to get quotas: %w", err)
	}

	// Get pricing
	pricing := DefaultPlanPricing()[sub.Plan]

	// Calculate base price
	totalCents := pricing.BasePriceCents

	// Calculate overage charges
	// Storage overage
	if usage.StorageBytes > quotas.MaxStorageBytes {
		overageGB := (usage.StorageBytes - quotas.MaxStorageBytes) / (1024 * 1024 * 1024)
		totalCents += overageGB * pricing.StoragePricePerGB
	}

	// Compile jobs overage
	if usage.CompileJobsCount > quotas.MaxCompileJobsPerMonth {
		overage := usage.CompileJobsCount - quotas.MaxCompileJobsPerMonth
		totalCents += int64(overage) * pricing.CompileJobPrice
	}

	// API requests overage
	overageRequests := int64(0)
	if usage.APIRequestsCount > int64(quotas.APIRateLimitPerHour)*24*30 {
		overageRequests = usage.APIRequestsCount - int64(quotas.APIRateLimitPerHour)*24*30
	}
	if overageRequests > 0 {
		overage1000s := overageRequests / 1000
		totalCents += overage1000s * pricing.APIRequestPrice
	}

	return totalCents, nil
}
