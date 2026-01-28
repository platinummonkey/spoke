// Package billing provides subscription management and usage-based billing with Stripe integration.
//
// # Overview
//
// This package implements three-tier subscription plans with usage-based overage charges,
// invoice generation, and payment method management.
//
// # Subscription Plans
//
// Small (Free):
//   - $0/month base
//   - Included: 10 modules, 100 versions, 5 GB storage
//   - Overages: $1/module, $0.10/GB storage
//
// Medium ($49/month):
//   - Included: 50 modules, 500 versions, 25 GB storage
//   - Overages: $0.75/module, $0.08/GB storage
//
// Large ($499/month):
//   - Included: 200 modules, 2000 versions, 100 GB storage
//   - Overages: $0.50/module, $0.05/GB storage
//
// # Usage Example
//
// Create subscription:
//
//	sub, err := service.CreateSubscription(ctx, &billing.CreateSubscriptionRequest{
//		OrganizationID: orgID,
//		PlanTier:       billing.PlanTierMedium,
//		PaymentMethodID: pmID,
//	})
//
// Generate invoice:
//
//	invoice, err := service.GenerateInvoice(ctx, orgID, billingPeriod)
//	fmt.Printf("Amount due: $%.2f\n", invoice.TotalAmount/100.0)
//
// # Related Packages
//
//   - pkg/orgs: Quota enforcement and usage tracking
package billing
