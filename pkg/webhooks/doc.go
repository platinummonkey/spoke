// Package webhooks provides event-driven webhook delivery for schema registry events.
//
// # Overview
//
// This package manages webhook registration, delivery, retries, and monitoring with
// automatic retry logic, rate limiting, and HMAC signature verification.
//
// # Webhook Events
//
// module.created, module.updated, module.deleted
// version.published, version.deprecated
// compilation.started, compilation.completed, compilation.failed
// validation.failed
//
// # Usage Example
//
// Register webhook:
//
//	webhook := &webhooks.Webhook{
//		URL:    "https://api.example.com/webhooks",
//		Events: []string{"module.created", "version.published"},
//		Secret: "webhook-secret",
//	}
//	manager.Register(ctx, webhook)
//
// Trigger event:
//
//	event := &webhooks.Event{
//		Type: "module.created",
//		Data: map[string]interface{}{
//			"module_name": "user-service",
//			"created_by":  userID,
//		},
//	}
//	manager.Dispatch(ctx, event)
//
// Verify signature (receiver side):
//
//	sig := r.Header.Get("X-Spoke-Signature")
//	if !webhooks.VerifySignature(body, secret, sig) {
//		return errors.New("invalid signature")
//	}
//
// # Retry Policy
//
// Exponential backoff: 1s, 2s, 4s, 8s, 16s
// Max retries: 5
// Timeout per attempt: 10s
//
// # Related Packages
//
//   - pkg/async: Asynchronous delivery
package webhooks
