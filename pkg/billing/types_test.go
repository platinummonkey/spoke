package billing

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/orgs"
	"github.com/stretchr/testify/assert"
)

func TestDefaultPlanPricing(t *testing.T) {
	pricing := DefaultPlanPricing()

	// Test small plan (free tier)
	smallPlan := pricing[orgs.QuotaTierSmall]
	assert.Equal(t, int64(0), smallPlan.BasePriceCents)
	assert.Equal(t, int64(1*1024*1024*1024), smallPlan.IncludedStorage)
	assert.Equal(t, 100, smallPlan.IncludedCompileJobs)

	// Test medium plan ($49/month)
	mediumPlan := pricing[orgs.QuotaTierMedium]
	assert.Equal(t, int64(4900), mediumPlan.BasePriceCents) // $49
	assert.Equal(t, int64(10*1024*1024*1024), mediumPlan.IncludedStorage)
	assert.Equal(t, 1000, mediumPlan.IncludedCompileJobs)

	// Test large plan ($499/month)
	largePlan := pricing[orgs.QuotaTierLarge]
	assert.Equal(t, int64(49900), largePlan.BasePriceCents) // $499
	assert.Equal(t, int64(100*1024*1024*1024), largePlan.IncludedStorage)
	assert.Equal(t, 10000, largePlan.IncludedCompileJobs)
}

func TestSubscriptionStatuses(t *testing.T) {
	assert.Equal(t, SubscriptionStatus("active"), SubscriptionStatusActive)
	assert.Equal(t, SubscriptionStatus("canceled"), SubscriptionStatusCanceled)
	assert.Equal(t, SubscriptionStatus("past_due"), SubscriptionStatusPastDue)
	assert.Equal(t, SubscriptionStatus("incomplete"), SubscriptionStatusIncomplete)
	assert.Equal(t, SubscriptionStatus("trialing"), SubscriptionStatusTrialing)
}

func TestInvoiceStatuses(t *testing.T) {
	assert.Equal(t, InvoiceStatus("draft"), InvoiceStatusDraft)
	assert.Equal(t, InvoiceStatus("open"), InvoiceStatusOpen)
	assert.Equal(t, InvoiceStatus("paid"), InvoiceStatusPaid)
	assert.Equal(t, InvoiceStatus("void"), InvoiceStatusVoid)
	assert.Equal(t, InvoiceStatus("uncollectible"), InvoiceStatusUncollectible)
}

func TestPaymentMethodTypes(t *testing.T) {
	assert.Equal(t, PaymentMethodType("card"), PaymentMethodTypeCard)
	assert.Equal(t, PaymentMethodType("bank_account"), PaymentMethodTypeBankAccount)
}

func TestPlanPricingOverages(t *testing.T) {
	pricing := DefaultPlanPricing()
	mediumPlan := pricing[orgs.QuotaTierMedium]

	// Test storage overage calculation
	includedStorage := mediumPlan.IncludedStorage
	usedStorage := includedStorage + 5*1024*1024*1024 // 5GB overage
	overageGB := (usedStorage - includedStorage) / (1024 * 1024 * 1024)
	overageCost := overageGB * mediumPlan.StoragePricePerGB

	assert.Equal(t, int64(5), overageGB)
	assert.Equal(t, int64(2500), overageCost) // 5GB * $5/GB = $25

	// Test compile job overage
	includedJobs := mediumPlan.IncludedCompileJobs
	usedJobs := includedJobs + 200
	overageJobs := usedJobs - includedJobs
	jobCost := int64(overageJobs) * mediumPlan.CompileJobPrice

	assert.Equal(t, 200, overageJobs)
	assert.Equal(t, int64(1000), jobCost) // 200 jobs * $0.05/job = $10
}

func TestBillCalculation(t *testing.T) {
	pricing := DefaultPlanPricing()
	mediumPlan := pricing[orgs.QuotaTierMedium]

	baseCost := mediumPlan.BasePriceCents
	storageOverage := int64(2500) // $25 for 5GB overage
	jobOverage := int64(1000)     // $10 for 200 job overage
	apiOverage := int64(100)      // $1 for 10k API request overage

	totalCost := baseCost + storageOverage + jobOverage + apiOverage

	assert.Equal(t, int64(8500), totalCost) // $49 + $25 + $10 + $1 = $85
}
