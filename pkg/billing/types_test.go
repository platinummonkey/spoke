package billing

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/orgs"
	"github.com/stretchr/testify/assert"
)

func TestDefaultPlanPricing(t *testing.T) {
	pricing := DefaultPlanPricing()

	// Test free plan
	freePlan := pricing[orgs.PlanFree]
	assert.Equal(t, int64(0), freePlan.BasePriceCents)
	assert.Equal(t, int64(1*1024*1024*1024), freePlan.IncludedStorage)
	assert.Equal(t, 100, freePlan.IncludedCompileJobs)

	// Test pro plan
	proPlan := pricing[orgs.PlanPro]
	assert.Equal(t, int64(4900), proPlan.BasePriceCents) // $49
	assert.Equal(t, int64(10*1024*1024*1024), proPlan.IncludedStorage)
	assert.Equal(t, 1000, proPlan.IncludedCompileJobs)

	// Test enterprise plan
	entPlan := pricing[orgs.PlanEnterprise]
	assert.Equal(t, int64(49900), entPlan.BasePriceCents) // $499
	assert.Equal(t, int64(100*1024*1024*1024), entPlan.IncludedStorage)
	assert.Equal(t, 10000, entPlan.IncludedCompileJobs)
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
	proPlan := pricing[orgs.PlanPro]

	// Test storage overage calculation
	includedStorage := proPlan.IncludedStorage
	usedStorage := includedStorage + 5*1024*1024*1024 // 5GB overage
	overageGB := (usedStorage - includedStorage) / (1024 * 1024 * 1024)
	overageCost := overageGB * proPlan.StoragePricePerGB

	assert.Equal(t, int64(5), overageGB)
	assert.Equal(t, int64(2500), overageCost) // 5GB * $5/GB = $25

	// Test compile job overage
	includedJobs := proPlan.IncludedCompileJobs
	usedJobs := includedJobs + 200
	overageJobs := usedJobs - includedJobs
	jobCost := int64(overageJobs) * proPlan.CompileJobPrice

	assert.Equal(t, 200, overageJobs)
	assert.Equal(t, int64(1000), jobCost) // 200 jobs * $0.05/job = $10
}

func TestBillCalculation(t *testing.T) {
	pricing := DefaultPlanPricing()
	proPlan := pricing[orgs.PlanPro]

	baseCost := proPlan.BasePriceCents
	storageOverage := int64(2500) // $25 for 5GB overage
	jobOverage := int64(1000)     // $10 for 200 job overage
	apiOverage := int64(100)      // $1 for 10k API request overage

	totalCost := baseCost + storageOverage + jobOverage + apiOverage

	assert.Equal(t, int64(8500), totalCost) // $49 + $25 + $10 + $1 = $85
}
