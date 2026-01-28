package billing

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/orgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrgService is a mock implementation of orgs.Service
type mockOrgService struct {
	getOrgFunc          func(id int64) (*orgs.Organization, error)
	updateOrgFunc       func(id int64, req *orgs.UpdateOrgRequest) error
	getDefaultQuotaFunc func(tier orgs.QuotaTier) *orgs.OrgQuotas
	updateQuotasFunc    func(orgID int64, quotas *orgs.OrgQuotas) error
	getUsageFunc        func(orgID int64) (*orgs.OrgUsage, error)
	getQuotasFunc       func(orgID int64) (*orgs.OrgQuotas, error)
}

func (m *mockOrgService) GetOrganization(id int64) (*orgs.Organization, error) {
	if m.getOrgFunc != nil {
		return m.getOrgFunc(id)
	}
	return &orgs.Organization{ID: id, DisplayName: "Test Org"}, nil
}

func (m *mockOrgService) UpdateOrganization(id int64, req *orgs.UpdateOrgRequest) error {
	if m.updateOrgFunc != nil {
		return m.updateOrgFunc(id, req)
	}
	return nil
}

func (m *mockOrgService) GetDefaultQuotas(tier orgs.QuotaTier) *orgs.OrgQuotas {
	if m.getDefaultQuotaFunc != nil {
		return m.getDefaultQuotaFunc(tier)
	}
	return &orgs.OrgQuotas{
		MaxModules:             10,
		MaxVersionsPerModule:   100,
		MaxStorageBytes:        5 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth: 5000,
		APIRateLimitPerHour:    5000,
	}
}

func (m *mockOrgService) UpdateQuotas(orgID int64, quotas *orgs.OrgQuotas) error {
	if m.updateQuotasFunc != nil {
		return m.updateQuotasFunc(orgID, quotas)
	}
	return nil
}

func (m *mockOrgService) GetUsage(orgID int64) (*orgs.OrgUsage, error) {
	if m.getUsageFunc != nil {
		return m.getUsageFunc(orgID)
	}
	now := time.Now()
	return &orgs.OrgUsage{
		OrgID:            orgID,
		PeriodStart:      now.AddDate(0, -1, 0),
		PeriodEnd:        now,
		StorageBytes:     1024 * 1024 * 1024,
		CompileJobsCount: 100,
		APIRequestsCount: 10000,
	}, nil
}

func (m *mockOrgService) GetQuotas(orgID int64) (*orgs.OrgQuotas, error) {
	if m.getQuotasFunc != nil {
		return m.getQuotasFunc(orgID)
	}
	return &orgs.OrgQuotas{
		MaxStorageBytes:        5 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth: 5000,
		APIRateLimitPerHour:    5000,
	}, nil
}

// Implement other required methods as no-ops
func (m *mockOrgService) CreateOrganization(org *orgs.Organization) error { return nil }
func (m *mockOrgService) GetOrganizationBySlug(slug string) (*orgs.Organization, error) {
	return nil, nil
}
func (m *mockOrgService) ListOrganizations(userID int64) ([]*orgs.Organization, error) { return nil, nil }
func (m *mockOrgService) DeleteOrganization(id int64) error                             { return nil }
func (m *mockOrgService) GetUsageHistory(orgID int64, limit int) ([]*orgs.OrgUsage, error) {
	return nil, nil
}
func (m *mockOrgService) ResetUsagePeriod(orgID int64) error                         { return nil }
func (m *mockOrgService) ListMembers(orgID int64) ([]*orgs.OrgMember, error)         { return nil, nil }
func (m *mockOrgService) GetMember(orgID, userID int64) (*orgs.OrgMember, error)     { return nil, nil }
func (m *mockOrgService) AddMember(orgID, userID int64, role auth.Role, invitedBy *int64) error {
	return nil
}
func (m *mockOrgService) UpdateMemberRole(orgID, userID int64, role auth.Role) error { return nil }
func (m *mockOrgService) RemoveMember(orgID, userID int64) error                       { return nil }
func (m *mockOrgService) CreateInvitation(invitation *orgs.OrgInvitation) error        { return nil }
func (m *mockOrgService) GetInvitation(token string) (*orgs.OrgInvitation, error)      { return nil, nil }
func (m *mockOrgService) ListInvitations(orgID int64) ([]*orgs.OrgInvitation, error)   { return nil, nil }
func (m *mockOrgService) AcceptInvitation(token string, userID int64) error            { return nil }
func (m *mockOrgService) RevokeInvitation(id int64) error                              { return nil }
func (m *mockOrgService) CleanupExpiredInvitations() error                             { return nil }
func (m *mockOrgService) CheckModuleQuota(orgID int64) error                           { return nil }
func (m *mockOrgService) CheckVersionQuota(orgID int64, moduleName string) error       { return nil }
func (m *mockOrgService) CheckStorageQuota(orgID int64, additionalBytes int64) error   { return nil }
func (m *mockOrgService) CheckCompileJobQuota(orgID int64) error                       { return nil }
func (m *mockOrgService) CheckAPIRateLimit(orgID int64) error                          { return nil }
func (m *mockOrgService) IncrementModules(orgID int64) error                           { return nil }
func (m *mockOrgService) IncrementVersions(orgID int64) error                          { return nil }
func (m *mockOrgService) IncrementStorage(orgID int64, bytes int64) error              { return nil }
func (m *mockOrgService) IncrementCompileJobs(orgID int64) error                       { return nil }
func (m *mockOrgService) IncrementAPIRequests(orgID int64) error                       { return nil }
func (m *mockOrgService) DecrementModules(orgID int64) error                           { return nil }
func (m *mockOrgService) DecrementVersions(orgID int64) error                          { return nil }
func (m *mockOrgService) DecrementStorage(orgID int64, bytes int64) error              { return nil }

func TestNewPostgresService(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, "test_key", service.stripeAPIKey)
	assert.Equal(t, "test_secret", service.stripeWebhookSecret)
	assert.Equal(t, mockOrg, service.orgService)
}

func TestServiceCreateSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("success - new subscription", func(t *testing.T) {
		orgID := int64(1)
		req := &CreateSubscriptionRequest{
			Plan: orgs.QuotaTierMedium,
		}

		// Mock GetStripeCustomer - not found
		mock.ExpectQuery("SELECT stripe_customer_id FROM subscriptions").
			WithArgs(orgID).
			WillReturnError(sql.ErrNoRows)

		// Mock CreateStripeCustomer
		mock.ExpectExec("UPDATE subscriptions SET stripe_customer_id").
			WithArgs(sqlmock.AnyArg(), orgID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Mock subscription insert
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now)
		mock.ExpectQuery("INSERT INTO subscriptions").
			WithArgs(orgID, req.Plan, sqlmock.AnyArg(), SubscriptionStatusActive,
				sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		sub, err := service.CreateSubscription(orgID, req)
		require.NoError(t, err)
		assert.NotNil(t, sub)
		assert.Equal(t, orgID, sub.OrgID)
		assert.Equal(t, orgs.QuotaTierMedium, sub.Plan)
		assert.Equal(t, SubscriptionStatusActive, sub.Status)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("error - database failure", func(t *testing.T) {
		db2, mock2, err := sqlmock.New()
		require.NoError(t, err)
		defer db2.Close()

		service2 := NewPostgresService(db2, "test_key", "test_secret", mockOrg)

		orgID := int64(1)
		req := &CreateSubscriptionRequest{
			Plan: orgs.QuotaTierMedium,
		}

		// Mock GetStripeCustomer - not found
		mock2.ExpectQuery("SELECT stripe_customer_id FROM subscriptions").
			WithArgs(orgID).
			WillReturnError(sql.ErrNoRows)

		// Mock CreateStripeCustomer
		mock2.ExpectExec("UPDATE subscriptions SET stripe_customer_id").
			WithArgs(sqlmock.AnyArg(), orgID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Mock subscription insert failure
		mock2.ExpectQuery("INSERT INTO subscriptions").
			WithArgs(orgID, req.Plan, sqlmock.AnyArg(), SubscriptionStatusActive,
				sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("database error"))

		sub, err := service2.CreateSubscription(orgID, req)
		assert.Error(t, err)
		assert.Nil(t, sub)
		assert.Contains(t, err.Error(), "failed to create subscription")
	})
}

func TestServiceGetSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("success", func(t *testing.T) {
		orgID := int64(1)
		now := time.Now()
		metadata := map[string]any{"key": "value"}
		metadataJSON, _ := json.Marshal(metadata)

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			1, orgID, orgs.QuotaTierMedium, "cus_123", "sub_123",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
			nil, nil, nil, metadataJSON, now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions").
			WithArgs(orgID).
			WillReturnRows(rows)

		sub, err := service.GetSubscription(orgID)
		require.NoError(t, err)
		assert.NotNil(t, sub)
		assert.Equal(t, int64(1), sub.ID)
		assert.Equal(t, orgID, sub.OrgID)
		assert.Equal(t, orgs.QuotaTierMedium, sub.Plan)
		assert.Equal(t, "cus_123", sub.StripeCustomerID)
		assert.Equal(t, SubscriptionStatusActive, sub.Status)
		assert.NotNil(t, sub.Metadata)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		orgID := int64(999)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions").
			WithArgs(orgID).
			WillReturnError(sql.ErrNoRows)

		sub, err := service.GetSubscription(orgID)
		assert.Error(t, err)
		assert.Nil(t, sub)
		assert.Contains(t, err.Error(), "subscription not found")
	})
}

func TestServiceUpdateSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("update plan", func(t *testing.T) {
		orgID := int64(1)
		newPlan := orgs.QuotaTierLarge
		req := &UpdateSubscriptionRequest{
			Plan: &newPlan,
		}

		mock.ExpectExec("UPDATE subscriptions SET plan").
			WithArgs(newPlan, orgID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			1, orgID, newPlan, "cus_123", "sub_123",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
			nil, nil, nil, []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions").
			WithArgs(orgID).
			WillReturnRows(rows)

		sub, err := service.UpdateSubscription(orgID, req)
		require.NoError(t, err)
		assert.NotNil(t, sub)
		assert.Equal(t, newPlan, sub.Plan)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("cancel at period end", func(t *testing.T) {
		orgID := int64(2)
		req := &UpdateSubscriptionRequest{
			CancelAtPeriodEnd: true,
		}

		mock.ExpectExec("UPDATE subscriptions SET cancel_at = current_period_end").
			WithArgs(orgID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			2, orgID, orgs.QuotaTierMedium, "cus_123", "sub_123",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), now.AddDate(0, 1, 0),
			nil, nil, nil, []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions").
			WithArgs(orgID).
			WillReturnRows(rows)

		sub, err := service.UpdateSubscription(orgID, req)
		require.NoError(t, err)
		assert.NotNil(t, sub)
		assert.NotNil(t, sub.CancelAt)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestServiceCancelSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("cancel immediately", func(t *testing.T) {
		orgID := int64(1)

		mock.ExpectExec("UPDATE subscriptions SET status").
			WithArgs(SubscriptionStatusCanceled, orgID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.CancelSubscription(orgID, true)
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("cancel at period end", func(t *testing.T) {
		orgID := int64(2)

		mock.ExpectExec("UPDATE subscriptions SET cancel_at = current_period_end").
			WithArgs(orgID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.CancelSubscription(orgID, false)
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestServiceReactivateSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	orgID := int64(1)

	mock.ExpectExec("UPDATE subscriptions SET status").
		WithArgs(SubscriptionStatusActive, orgID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
		"status", "current_period_start", "current_period_end", "cancel_at",
		"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
	}).AddRow(
		1, orgID, orgs.QuotaTierMedium, "cus_123", "sub_123",
		SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
		nil, nil, nil, []byte("{}"), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM subscriptions").
		WithArgs(orgID).
		WillReturnRows(rows)

	sub, err := service.ReactivateSubscription(orgID)
	require.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, SubscriptionStatusActive, sub.Status)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestServiceGetInvoice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("success", func(t *testing.T) {
		invoiceID := int64(1)
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "invoice_number", "stripe_invoice_id", "amount_cents",
			"currency", "period_start", "period_end", "status", "paid_at", "due_date",
			"invoice_url", "invoice_pdf_url", "metadata", "created_at", "updated_at",
		}).AddRow(
			invoiceID, 1, "INV-001", "in_123", 4900,
			"usd", now, now.AddDate(0, 1, 0), InvoiceStatusPaid, now, now.AddDate(0, 0, 30),
			"http://example.com", "http://example.com/pdf", []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM invoices").
			WithArgs(invoiceID).
			WillReturnRows(rows)

		invoice, err := service.GetInvoice(invoiceID)
		require.NoError(t, err)
		assert.NotNil(t, invoice)
		assert.Equal(t, invoiceID, invoice.ID)
		assert.Equal(t, int64(4900), invoice.AmountCents)
		assert.Equal(t, InvoiceStatusPaid, invoice.Status)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		invoiceID := int64(999)

		mock.ExpectQuery("SELECT (.+) FROM invoices").
			WithArgs(invoiceID).
			WillReturnError(sql.ErrNoRows)

		invoice, err := service.GetInvoice(invoiceID)
		assert.Error(t, err)
		assert.Nil(t, invoice)
		assert.Contains(t, err.Error(), "invoice not found")
	})
}

func TestServiceListInvoices(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	orgID := int64(1)
	limit := 10
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "org_id", "invoice_number", "stripe_invoice_id", "amount_cents",
		"currency", "period_start", "period_end", "status", "paid_at", "due_date",
		"invoice_url", "invoice_pdf_url", "metadata", "created_at", "updated_at",
	}).
		AddRow(1, orgID, "INV-001", "in_123", 4900, "usd", now, now, InvoiceStatusPaid,
			now, now, "", "", []byte("{}"), now, now).
		AddRow(2, orgID, "INV-002", "in_124", 4900, "usd", now, now, InvoiceStatusOpen,
			nil, now, "", "", []byte("{}"), now, now)

	mock.ExpectQuery("SELECT (.+) FROM invoices").
		WithArgs(orgID, limit).
		WillReturnRows(rows)

	invoices, err := service.ListInvoices(orgID, limit)
	require.NoError(t, err)
	assert.NotNil(t, invoices)
	assert.Len(t, invoices, 2)
	assert.Equal(t, "INV-001", invoices[0].InvoiceNumber)
	assert.Equal(t, "INV-002", invoices[1].InvoiceNumber)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestServiceGenerateInvoice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	mockOrg := &mockOrgService{
		getUsageFunc: func(orgID int64) (*orgs.OrgUsage, error) {
			return &orgs.OrgUsage{
				OrgID:            orgID,
				PeriodStart:      now.AddDate(0, -1, 0),
				PeriodEnd:        now,
				StorageBytes:     6 * 1024 * 1024 * 1024,
				CompileJobsCount: 100,
				APIRequestsCount: 10000,
			}, nil
		},
		getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
			return &orgs.OrgQuotas{
				MaxStorageBytes:        5 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 5000,
				APIRateLimitPerHour:    5000,
			}, nil
		},
	}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	orgID := int64(1)

	// Mock GetSubscription
	subRows := sqlmock.NewRows([]string{
		"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
		"status", "current_period_start", "current_period_end", "cancel_at",
		"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
	}).AddRow(
		1, orgID, orgs.QuotaTierSmall, "cus_123", "sub_123",
		SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
		nil, nil, nil, []byte("{}"), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM subscriptions").
		WithArgs(orgID).
		WillReturnRows(subRows)

	// Mock invoice insert
	invoiceRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(1, now, now)

	mock.ExpectQuery("INSERT INTO invoices").
		WithArgs(orgID, sqlmock.AnyArg(), "usd", sqlmock.AnyArg(), sqlmock.AnyArg(),
			InvoiceStatusOpen, sqlmock.AnyArg()).
		WillReturnRows(invoiceRows)

	invoice, err := service.GenerateInvoice(orgID)
	require.NoError(t, err)
	assert.NotNil(t, invoice)
	assert.Equal(t, orgID, invoice.OrgID)
	assert.Equal(t, InvoiceStatusOpen, invoice.Status)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestServiceCalculateBill(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()

	t.Run("no overage", func(t *testing.T) {
		mockOrg := &mockOrgService{
			getUsageFunc: func(orgID int64) (*orgs.OrgUsage, error) {
				return &orgs.OrgUsage{
					OrgID:            orgID,
					StorageBytes:     1 * 1024 * 1024 * 1024,
					CompileJobsCount: 100,
					APIRequestsCount: 10000,
				}, nil
			},
			getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
				return &orgs.OrgQuotas{
					MaxStorageBytes:        5 * 1024 * 1024 * 1024,
					MaxCompileJobsPerMonth: 5000,
					APIRateLimitPerHour:    5000,
				}, nil
			},
		}
		service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

		orgID := int64(1)

		// Mock GetSubscription
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			1, orgID, orgs.QuotaTierSmall, "cus_123", "sub_123",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
			nil, nil, nil, []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions").
			WithArgs(orgID).
			WillReturnRows(rows)

		amount, err := service.CalculateBill(orgID, now.AddDate(0, -1, 0), now)
		require.NoError(t, err)
		assert.Equal(t, int64(0), amount)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("with storage overage", func(t *testing.T) {
		mockOrg := &mockOrgService{
			getUsageFunc: func(orgID int64) (*orgs.OrgUsage, error) {
				return &orgs.OrgUsage{
					OrgID:            orgID,
					StorageBytes:     15 * 1024 * 1024 * 1024,
					CompileJobsCount: 500,
					APIRequestsCount: 100000,
				}, nil
			},
			getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
				return &orgs.OrgQuotas{
					MaxStorageBytes:        10 * 1024 * 1024 * 1024,
					MaxCompileJobsPerMonth: 1000,
					APIRateLimitPerHour:    10000,
				}, nil
			},
		}
		service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

		orgID := int64(2)

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			2, orgID, orgs.QuotaTierMedium, "cus_124", "sub_124",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
			nil, nil, nil, []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions").
			WithArgs(orgID).
			WillReturnRows(rows)

		amount, err := service.CalculateBill(orgID, now.AddDate(0, -1, 0), now)
		require.NoError(t, err)
		assert.True(t, amount >= 4900)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestServiceCreateStripeCustomer(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	orgID := int64(1)

	mock.ExpectExec("UPDATE subscriptions SET stripe_customer_id").
		WithArgs(sqlmock.AnyArg(), orgID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	customerID, err := service.CreateStripeCustomer(orgID)
	require.NoError(t, err)
	assert.NotEmpty(t, customerID)
	assert.Contains(t, customerID, "cus_mock_")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestServiceGetStripeCustomer(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("customer exists", func(t *testing.T) {
		orgID := int64(1)
		customerID := "cus_123"

		rows := sqlmock.NewRows([]string{"stripe_customer_id"}).
			AddRow(customerID)

		mock.ExpectQuery("SELECT stripe_customer_id FROM subscriptions").
			WithArgs(orgID).
			WillReturnRows(rows)

		result, err := service.GetStripeCustomer(orgID)
		require.NoError(t, err)
		assert.Equal(t, customerID, result)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("customer not found", func(t *testing.T) {
		orgID := int64(999)

		mock.ExpectQuery("SELECT stripe_customer_id FROM subscriptions").
			WithArgs(orgID).
			WillReturnError(sql.ErrNoRows)

		result, err := service.GetStripeCustomer(orgID)
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "subscription not found")
	})
}

func TestServiceHandleWebhook(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("valid webhook", func(t *testing.T) {
		event := StripeWebhookEvent{
			ID:   "evt_123",
			Type: "customer.subscription.created",
			Data: map[string]any{"test": "data"},
		}
		payload, _ := json.Marshal(event)

		err := service.HandleWebhook(payload, "test_signature")
		require.NoError(t, err)
	})

	t.Run("unknown event type", func(t *testing.T) {
		event := StripeWebhookEvent{
			ID:   "evt_124",
			Type: "unknown.event",
			Data: map[string]any{},
		}
		payload, _ := json.Marshal(event)

		err := service.HandleWebhook(payload, "test_signature")
		require.NoError(t, err)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		payload := []byte("invalid json")

		err := service.HandleWebhook(payload, "test_signature")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse webhook")
	})
}

func TestServiceAddPaymentMethod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	orgID := int64(1)
	now := time.Now()
	req := &CreatePaymentMethodRequest{
		StripePaymentMethodID: "pm_123",
		SetAsDefault:          true,
	}

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(1, now, now)

	mock.ExpectQuery("INSERT INTO payment_methods").
		WithArgs(orgID, req.StripePaymentMethodID, PaymentMethodTypeCard, req.SetAsDefault).
		WillReturnRows(rows)

	pm, err := service.AddPaymentMethod(orgID, req)
	require.NoError(t, err)
	assert.NotNil(t, pm)
	assert.Equal(t, orgID, pm.OrgID)
	assert.Equal(t, req.StripePaymentMethodID, pm.StripePaymentMethodID)
	assert.True(t, pm.IsDefault)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestServiceListPaymentMethods(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	orgID := int64(1)
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "org_id", "stripe_payment_method_id", "type", "is_default",
		"card_brand", "card_last4", "card_exp_month", "card_exp_year",
		"bank_name", "bank_last4", "metadata", "created_at", "updated_at",
	}).
		AddRow(1, orgID, "pm_123", PaymentMethodTypeCard, true,
			"visa", "4242", 12, 2025, nil, nil, []byte("{}"), now, now).
		AddRow(2, orgID, "pm_124", PaymentMethodTypeCard, false,
			"mastercard", "5555", 6, 2026, nil, nil, []byte("{}"), now, now)

	mock.ExpectQuery("SELECT (.+) FROM payment_methods").
		WithArgs(orgID).
		WillReturnRows(rows)

	methods, err := service.ListPaymentMethods(orgID)
	require.NoError(t, err)
	assert.Len(t, methods, 2)
	assert.Equal(t, "pm_123", methods[0].StripePaymentMethodID)
	assert.True(t, methods[0].IsDefault)
	assert.Equal(t, "visa", methods[0].CardBrand)
	assert.Equal(t, "pm_124", methods[1].StripePaymentMethodID)
	assert.False(t, methods[1].IsDefault)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestServiceSetDefaultPaymentMethod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	orgID := int64(1)
	paymentMethodID := int64(2)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE payment_methods SET is_default = false").
		WithArgs(orgID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE payment_methods SET is_default = true").
		WithArgs(paymentMethodID, orgID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = service.SetDefaultPaymentMethod(orgID, paymentMethodID)
	require.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestServiceRemovePaymentMethod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("success", func(t *testing.T) {
		orgID := int64(1)
		paymentMethodID := int64(2)

		mock.ExpectExec("DELETE FROM payment_methods").
			WithArgs(paymentMethodID, orgID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.RemovePaymentMethod(orgID, paymentMethodID)
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		orgID := int64(1)
		paymentMethodID := int64(999)

		mock.ExpectExec("DELETE FROM payment_methods").
			WithArgs(paymentMethodID, orgID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := service.RemovePaymentMethod(orgID, paymentMethodID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payment method not found")
	})
}

func TestServiceRecordUsage(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	orgID := int64(1)
	usage := &orgs.OrgUsage{
		OrgID:            orgID,
		StorageBytes:     1024,
		CompileJobsCount: 10,
	}

	err = service.RecordUsage(orgID, usage)
	require.NoError(t, err)
}

func TestStripeWebhookEventHandlers(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("handleSubscriptionUpdated", func(t *testing.T) {
		event := StripeWebhookEvent{
			ID:   "evt_456",
			Type: "customer.subscription.updated",
			Data: map[string]any{"object": map[string]any{"id": "sub_456"}},
		}
		payload, _ := json.Marshal(event)
		err := service.HandleWebhook(payload, "sig")
		require.NoError(t, err)
	})

	t.Run("handleSubscriptionDeleted", func(t *testing.T) {
		event := StripeWebhookEvent{
			ID:   "evt_789",
			Type: "customer.subscription.deleted",
			Data: map[string]any{"object": map[string]any{"id": "sub_789"}},
		}
		payload, _ := json.Marshal(event)
		err := service.HandleWebhook(payload, "sig")
		require.NoError(t, err)
	})

	t.Run("handleInvoicePaid", func(t *testing.T) {
		event := StripeWebhookEvent{
			ID:   "evt_101",
			Type: "invoice.paid",
			Data: map[string]any{"object": map[string]any{"id": "in_101"}},
		}
		payload, _ := json.Marshal(event)
		err := service.HandleWebhook(payload, "sig")
		require.NoError(t, err)
	})

	t.Run("handleInvoicePaymentFailed", func(t *testing.T) {
		event := StripeWebhookEvent{
			ID:   "evt_202",
			Type: "invoice.payment_failed",
			Data: map[string]any{"object": map[string]any{"id": "in_202"}},
		}
		payload, _ := json.Marshal(event)
		err := service.HandleWebhook(payload, "sig")
		require.NoError(t, err)
	})
}

func TestStripeGetCustomerEdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("database error during query", func(t *testing.T) {
		orgID := int64(123)
		mock.ExpectQuery("SELECT stripe_customer_id FROM subscriptions WHERE org_id").
			WithArgs(orgID).
			WillReturnError(sql.ErrConnDone)

		customerID, err := service.GetStripeCustomer(orgID)
		require.Error(t, err)
		assert.Empty(t, customerID)
		assert.Contains(t, err.Error(), "failed to get customer ID")
	})

	t.Run("null customer ID in database", func(t *testing.T) {
		orgID := int64(456)
		rows := sqlmock.NewRows([]string{"stripe_customer_id"}).AddRow(nil)
		mock.ExpectQuery("SELECT stripe_customer_id FROM subscriptions WHERE org_id").
			WithArgs(orgID).
			WillReturnRows(rows)

		customerID, err := service.GetStripeCustomer(orgID)
		require.Error(t, err)
		assert.Empty(t, customerID)
		assert.Contains(t, err.Error(), "customer ID not set")
	})
}

func TestStripeAddPaymentMethodEdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("database error during insert", func(t *testing.T) {
		orgID := int64(123)
		req := &CreatePaymentMethodRequest{
			StripePaymentMethodID: "pm_test_error",
			SetAsDefault:          false,
		}

		mock.ExpectQuery("INSERT INTO payment_methods").
			WithArgs(orgID, req.StripePaymentMethodID, PaymentMethodTypeCard, req.SetAsDefault).
			WillReturnError(sql.ErrConnDone)

		pm, err := service.AddPaymentMethod(orgID, req)
		require.Error(t, err)
		assert.Nil(t, pm)
		assert.Contains(t, err.Error(), "failed to add payment method")
	})
}

func TestStripeListPaymentMethodsEdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("database error during query", func(t *testing.T) {
		orgID := int64(123)
		mock.ExpectQuery("SELECT (.+) FROM payment_methods WHERE org_id").
			WithArgs(orgID).
			WillReturnError(sql.ErrConnDone)

		methods, err := service.ListPaymentMethods(orgID)
		require.Error(t, err)
		assert.Nil(t, methods)
		assert.Contains(t, err.Error(), "failed to list payment methods")
	})

	t.Run("scan error during iteration", func(t *testing.T) {
		orgID := int64(456)
		rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT (.+) FROM payment_methods WHERE org_id").
			WithArgs(orgID).
			WillReturnRows(rows)

		methods, err := service.ListPaymentMethods(orgID)
		require.Error(t, err)
		assert.Nil(t, methods)
		assert.Contains(t, err.Error(), "failed to scan payment method")
	})

	t.Run("invalid metadata JSON", func(t *testing.T) {
		orgID := int64(789)
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "stripe_payment_method_id", "type", "is_default",
			"card_brand", "card_last4", "card_exp_month", "card_exp_year",
			"bank_name", "bank_last4", "metadata", "created_at", "updated_at",
		}).AddRow(1, orgID, "pm_test", PaymentMethodTypeCard, true,
			"visa", "4242", 12, 2025, nil, nil, []byte("invalid json"), now, now)

		mock.ExpectQuery("SELECT (.+) FROM payment_methods WHERE org_id").
			WithArgs(orgID).
			WillReturnRows(rows)

		methods, err := service.ListPaymentMethods(orgID)
		require.Error(t, err)
		assert.Nil(t, methods)
		assert.Contains(t, err.Error(), "failed to unmarshal metadata")
	})
}

func TestStripeSetDefaultPaymentMethodEdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("transaction begin error", func(t *testing.T) {
		orgID := int64(123)
		paymentMethodID := int64(1)

		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		err := service.SetDefaultPaymentMethod(orgID, paymentMethodID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
	})

	t.Run("error unsetting defaults", func(t *testing.T) {
		orgID := int64(456)
		paymentMethodID := int64(1)

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE payment_methods SET is_default = false WHERE org_id").
			WithArgs(orgID).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := service.SetDefaultPaymentMethod(orgID, paymentMethodID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unset default")
	})

	t.Run("error setting new default", func(t *testing.T) {
		orgID := int64(789)
		paymentMethodID := int64(1)

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE payment_methods SET is_default = false WHERE org_id").
			WithArgs(orgID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE payment_methods SET is_default = true WHERE id").
			WithArgs(paymentMethodID, orgID).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := service.SetDefaultPaymentMethod(orgID, paymentMethodID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set default")
	})

	t.Run("payment method not found", func(t *testing.T) {
		orgID := int64(999)
		paymentMethodID := int64(999)

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE payment_methods SET is_default = false WHERE org_id").
			WithArgs(orgID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE payment_methods SET is_default = true WHERE id").
			WithArgs(paymentMethodID, orgID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		err := service.SetDefaultPaymentMethod(orgID, paymentMethodID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payment method not found")
	})

	t.Run("rows affected check error", func(t *testing.T) {
		orgID := int64(101)
		paymentMethodID := int64(1)

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE payment_methods SET is_default = false WHERE org_id").
			WithArgs(orgID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE payment_methods SET is_default = true WHERE id").
			WithArgs(paymentMethodID, orgID).
			WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))
		mock.ExpectRollback()

		err := service.SetDefaultPaymentMethod(orgID, paymentMethodID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected")
	})
}

func TestStripeRemovePaymentMethodEdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockOrg := &mockOrgService{}
	service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

	t.Run("database error during delete", func(t *testing.T) {
		orgID := int64(123)
		paymentMethodID := int64(1)

		mock.ExpectExec("DELETE FROM payment_methods WHERE id").
			WithArgs(paymentMethodID, orgID).
			WillReturnError(sql.ErrConnDone)

		err := service.RemovePaymentMethod(orgID, paymentMethodID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove payment method")
	})

	t.Run("rows affected check error", func(t *testing.T) {
		orgID := int64(456)
		paymentMethodID := int64(1)

		mock.ExpectExec("DELETE FROM payment_methods WHERE id").
			WithArgs(paymentMethodID, orgID).
			WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))

		err := service.RemovePaymentMethod(orgID, paymentMethodID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected")
	})
}

func TestStripeCalculateBillEdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()

	t.Run("subscription not found", func(t *testing.T) {
		mockOrg := &mockOrgService{}
		service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

		orgID := int64(123)
		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE org_id").
			WithArgs(orgID).
			WillReturnError(sql.ErrNoRows)

		amount, err := service.CalculateBill(orgID, now.AddDate(0, -1, 0), now)
		require.Error(t, err)
		assert.Equal(t, int64(0), amount)
		assert.Contains(t, err.Error(), "failed to get subscription")
	})

	t.Run("usage retrieval error", func(t *testing.T) {
		mockOrg := &mockOrgService{
			getUsageFunc: func(orgID int64) (*orgs.OrgUsage, error) {
				return nil, errors.New("usage error")
			},
		}
		service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

		orgID := int64(456)
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			1, orgID, orgs.QuotaTierMedium, "cus_456", "sub_456",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
			nil, nil, nil, []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE org_id").
			WithArgs(orgID).
			WillReturnRows(rows)

		amount, err := service.CalculateBill(orgID, now.AddDate(0, -1, 0), now)
		require.Error(t, err)
		assert.Equal(t, int64(0), amount)
		assert.Contains(t, err.Error(), "failed to get usage")
	})

	t.Run("quotas retrieval error", func(t *testing.T) {
		mockOrg := &mockOrgService{
			getUsageFunc: func(orgID int64) (*orgs.OrgUsage, error) {
				return &orgs.OrgUsage{
					OrgID:            orgID,
					StorageBytes:     5 * 1024 * 1024 * 1024,
					CompileJobsCount: 100,
					APIRequestsCount: 5000,
				}, nil
			},
			getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
				return nil, errors.New("quotas error")
			},
		}
		service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

		orgID := int64(789)
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			1, orgID, orgs.QuotaTierMedium, "cus_789", "sub_789",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
			nil, nil, nil, []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE org_id").
			WithArgs(orgID).
			WillReturnRows(rows)

		amount, err := service.CalculateBill(orgID, now.AddDate(0, -1, 0), now)
		require.Error(t, err)
		assert.Equal(t, int64(0), amount)
		assert.Contains(t, err.Error(), "failed to get quotas")
	})

	t.Run("calculate with compile jobs overage", func(t *testing.T) {
		mockOrg := &mockOrgService{
			getUsageFunc: func(orgID int64) (*orgs.OrgUsage, error) {
				return &orgs.OrgUsage{
					OrgID:            orgID,
					StorageBytes:     5 * 1024 * 1024 * 1024,
					CompileJobsCount: 1500, // 500 over limit
					APIRequestsCount: 5000,
				}, nil
			},
			getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
				return &orgs.OrgQuotas{
					MaxStorageBytes:        10 * 1024 * 1024 * 1024,
					MaxCompileJobsPerMonth: 1000,
					APIRateLimitPerHour:    10000,
				}, nil
			},
		}
		service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

		orgID := int64(999)
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			1, orgID, orgs.QuotaTierMedium, "cus_999", "sub_999",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
			nil, nil, nil, []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE org_id").
			WithArgs(orgID).
			WillReturnRows(rows)

		amount, err := service.CalculateBill(orgID, now.AddDate(0, -1, 0), now)
		require.NoError(t, err)
		// Base: 4900 + compile overage: 500 * 5 = 2500 = 7400
		assert.True(t, amount >= 4900)
	})

	t.Run("calculate with API requests overage", func(t *testing.T) {
		mockOrg := &mockOrgService{
			getUsageFunc: func(orgID int64) (*orgs.OrgUsage, error) {
				return &orgs.OrgUsage{
					OrgID:            orgID,
					StorageBytes:     5 * 1024 * 1024 * 1024,
					CompileJobsCount: 500,
					APIRequestsCount: 20000000, // 20M requests
				}, nil
			},
			getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
				return &orgs.OrgQuotas{
					MaxStorageBytes:        10 * 1024 * 1024 * 1024,
					MaxCompileJobsPerMonth: 1000,
					APIRateLimitPerHour:    10000, // 10k/hour * 24 * 30 = 7.2M/month
				}, nil
			},
		}
		service := NewPostgresService(db, "test_key", "test_secret", mockOrg)

		orgID := int64(1001)
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "plan", "stripe_customer_id", "stripe_subscription_id",
			"status", "current_period_start", "current_period_end", "cancel_at",
			"canceled_at", "trial_start", "trial_end", "metadata", "created_at", "updated_at",
		}).AddRow(
			1, orgID, orgs.QuotaTierMedium, "cus_1001", "sub_1001",
			SubscriptionStatusActive, now, now.AddDate(0, 1, 0), nil,
			nil, nil, nil, []byte("{}"), now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE org_id").
			WithArgs(orgID).
			WillReturnRows(rows)

		amount, err := service.CalculateBill(orgID, now.AddDate(0, -1, 0), now)
		require.NoError(t, err)
		// Should include base + API overage charges
		assert.True(t, amount >= 4900)
	})
}
