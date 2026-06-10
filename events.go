package opensettle

// WebhookEventType is the `event` field carried by every webhook delivery
// (and accepted in [CreateWebhookEndpointRequest.Events] /
// [UpdateWebhookEndpointRequest.Events] as an allow-list). The constants
// below are the complete set the API emits; switch on them when routing a
// verified delivery.
//
// These are plain strings underneath, so a WebhookEventType is directly
// assignable to the `[]string` Events allow-list (convert per element) and
// comparable to the decoded `event` field of a webhook envelope.
type WebhookEventType string

const (
	EventAllowanceDepleted WebhookEventType = "allowance.depleted"

	EventCheckoutCreated   WebhookEventType = "checkout.created"
	EventCheckoutExpired   WebhookEventType = "checkout.expired"
	EventCheckoutSucceeded WebhookEventType = "checkout.succeeded"

	EventCustomerCreated WebhookEventType = "customer.created"
	EventCustomerDeleted WebhookEventType = "customer.deleted"
	EventCustomerUpdated WebhookEventType = "customer.updated"

	EventInvoiceCreated      WebhookEventType = "invoice.created"
	EventInvoicePaid         WebhookEventType = "invoice.paid"
	EventInvoicePastDue      WebhookEventType = "invoice.past_due"
	EventInvoiceReminderSent WebhookEventType = "invoice.reminder_sent"
	EventInvoiceSent         WebhookEventType = "invoice.sent"
	EventInvoiceVoided       WebhookEventType = "invoice.voided"

	EventPaymentConfirmed      WebhookEventType = "payment.confirmed"
	EventPaymentFailed         WebhookEventType = "payment.failed"
	EventPaymentPending        WebhookEventType = "payment.pending"
	EventPaymentRefunded       WebhookEventType = "payment.refunded"
	EventPaymentReorgSuspected WebhookEventType = "payment.reorg_suspected"
	EventPaymentReorged        WebhookEventType = "payment.reorged"
	EventPaymentReversed       WebhookEventType = "payment.reversed"

	EventPriceCreated WebhookEventType = "price.created"
	EventPriceUpdated WebhookEventType = "price.updated"

	EventProductCreated WebhookEventType = "product.created"
	EventProductUpdated WebhookEventType = "product.updated"

	EventRefundBroadcast WebhookEventType = "refund.broadcast"
	EventRefundConfirmed WebhookEventType = "refund.confirmed"
	EventRefundInitiated WebhookEventType = "refund.initiated"

	EventSubscriptionCanceled      WebhookEventType = "subscription.canceled"
	EventSubscriptionCreated       WebhookEventType = "subscription.created"
	EventSubscriptionPastDue       WebhookEventType = "subscription.past_due"
	EventSubscriptionPaused        WebhookEventType = "subscription.paused"
	EventSubscriptionPaymentFailed WebhookEventType = "subscription.payment_failed"
	EventSubscriptionPlanChanged   WebhookEventType = "subscription.plan_changed"
	EventSubscriptionRenewed       WebhookEventType = "subscription.renewed"
	EventSubscriptionResumed       WebhookEventType = "subscription.resumed"
	EventSubscriptionTrialEnded    WebhookEventType = "subscription.trial_ended"

	EventWalletConnected WebhookEventType = "wallet.connected"
	EventWalletRemoved   WebhookEventType = "wallet.removed"
	EventWalletVerified  WebhookEventType = "wallet.verified"

	EventWebhookEndpointCreated WebhookEventType = "webhook.endpoint.created"
	EventWebhookEndpointTest    WebhookEventType = "webhook.endpoint.test"
)

// AllWebhookEvents is the full set of event types the API emits, in a
// stable order. Useful for building an explicit "subscribe to everything"
// allow-list or for validating a caller-supplied selection.
//
// (Passing an empty Events slice to a webhook endpoint already subscribes
// to all events; this slice is for when you want the names enumerated.)
var AllWebhookEvents = []WebhookEventType{
	EventAllowanceDepleted,
	EventCheckoutCreated,
	EventCheckoutExpired,
	EventCheckoutSucceeded,
	EventCustomerCreated,
	EventCustomerDeleted,
	EventCustomerUpdated,
	EventInvoiceCreated,
	EventInvoicePaid,
	EventInvoicePastDue,
	EventInvoiceReminderSent,
	EventInvoiceSent,
	EventInvoiceVoided,
	EventPaymentConfirmed,
	EventPaymentFailed,
	EventPaymentPending,
	EventPaymentRefunded,
	EventPaymentReorgSuspected,
	EventPaymentReorged,
	EventPaymentReversed,
	EventPriceCreated,
	EventPriceUpdated,
	EventProductCreated,
	EventProductUpdated,
	EventRefundBroadcast,
	EventRefundConfirmed,
	EventRefundInitiated,
	EventSubscriptionCanceled,
	EventSubscriptionCreated,
	EventSubscriptionPastDue,
	EventSubscriptionPaused,
	EventSubscriptionPaymentFailed,
	EventSubscriptionPlanChanged,
	EventSubscriptionRenewed,
	EventSubscriptionResumed,
	EventSubscriptionTrialEnded,
	EventWalletConnected,
	EventWalletRemoved,
	EventWalletVerified,
	EventWebhookEndpointCreated,
	EventWebhookEndpointTest,
}
