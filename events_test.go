package opensettle

import "testing"

// TestWebhookEventConstants pins the literal wire values so a rename can't
// silently drift the SDK away from what the API emits.
func TestWebhookEventConstants(t *testing.T) {
	cases := map[WebhookEventType]string{
		EventAllowanceDepleted:         "allowance.depleted",
		EventCheckoutCreated:           "checkout.created",
		EventCheckoutExpired:           "checkout.expired",
		EventCheckoutSucceeded:         "checkout.succeeded",
		EventCommissionAccrued:         "commission.accrued",
		EventCommissionAdjusted:        "commission.adjusted",
		EventCommissionPaid:            "commission.paid",
		EventCommissionVoided:          "commission.voided",
		EventCustomerCreated:           "customer.created",
		EventCustomerDeleted:           "customer.deleted",
		EventCustomerUpdated:           "customer.updated",
		EventInvoiceCreated:            "invoice.created",
		EventInvoicePaid:               "invoice.paid",
		EventInvoicePastDue:            "invoice.past_due",
		EventInvoiceReminderSent:       "invoice.reminder_sent",
		EventInvoiceSent:               "invoice.sent",
		EventInvoiceVoided:             "invoice.voided",
		EventPaymentConfirmed:          "payment.confirmed",
		EventPaymentFailed:             "payment.failed",
		EventPaymentPending:            "payment.pending",
		EventPaymentRefunded:           "payment.refunded",
		EventPaymentReorgSuspected:     "payment.reorg_suspected",
		EventPaymentReorged:            "payment.reorged",
		EventPaymentReversed:           "payment.reversed",
		EventPriceCreated:              "price.created",
		EventPriceUpdated:              "price.updated",
		EventProductCreated:            "product.created",
		EventProductUpdated:            "product.updated",
		EventRefundBroadcast:           "refund.broadcast",
		EventRefundConfirmed:           "refund.confirmed",
		EventRefundInitiated:           "refund.initiated",
		EventSubscriptionCanceled:      "subscription.canceled",
		EventSubscriptionCreated:       "subscription.created",
		EventSubscriptionPastDue:       "subscription.past_due",
		EventSubscriptionPaused:        "subscription.paused",
		EventSubscriptionPaymentFailed: "subscription.payment_failed",
		EventSubscriptionPlanChanged:   "subscription.plan_changed",
		EventSubscriptionRenewed:       "subscription.renewed",
		EventSubscriptionResumed:       "subscription.resumed",
		EventSubscriptionTrialEnded:    "subscription.trial_ended",
		EventWalletConnected:           "wallet.connected",
		EventWalletRemoved:             "wallet.removed",
		EventWalletVerified:            "wallet.verified",
		EventWebhookEndpointCreated:    "webhook.endpoint.created",
		EventWebhookEndpointTest:       "webhook.endpoint.test",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("event const = %q, want %q", string(got), want)
		}
	}
}

// TestAllWebhookEvents guards against the prior hallucinated-event
// regression (payment.screened / allowance.recorded do NOT exist) by
// pinning the exact count and asserting no duplicates and no stray names
// outside the known set. (webhook.endpoint.created / webhook.endpoint.test
// ARE real events the backend emits via emitEvent.)
func TestAllWebhookEvents(t *testing.T) {
	const want = 45
	if len(AllWebhookEvents) != want {
		t.Fatalf("AllWebhookEvents has %d entries, want %d", len(AllWebhookEvents), want)
	}
	known := map[WebhookEventType]bool{
		EventAllowanceDepleted: true, EventCheckoutCreated: true, EventCheckoutExpired: true,
		EventCheckoutSucceeded: true, EventCommissionAccrued: true, EventCommissionAdjusted: true,
		EventCommissionPaid: true, EventCommissionVoided: true,
		EventCustomerCreated: true, EventCustomerDeleted: true,
		EventCustomerUpdated: true, EventInvoiceCreated: true, EventInvoicePaid: true,
		EventInvoicePastDue: true, EventInvoiceReminderSent: true, EventInvoiceSent: true,
		EventInvoiceVoided: true, EventPaymentConfirmed: true, EventPaymentFailed: true,
		EventPaymentPending: true, EventPaymentRefunded: true, EventPaymentReorgSuspected: true,
		EventPaymentReorged: true, EventPaymentReversed: true, EventPriceCreated: true,
		EventPriceUpdated: true, EventProductCreated: true, EventProductUpdated: true,
		EventRefundBroadcast: true, EventRefundConfirmed: true, EventRefundInitiated: true,
		EventSubscriptionCanceled: true, EventSubscriptionCreated: true, EventSubscriptionPastDue: true,
		EventSubscriptionPaused: true, EventSubscriptionPaymentFailed: true, EventSubscriptionPlanChanged: true,
		EventSubscriptionRenewed: true, EventSubscriptionResumed: true, EventSubscriptionTrialEnded: true,
		EventWalletConnected: true, EventWalletRemoved: true, EventWalletVerified: true,
		EventWebhookEndpointCreated: true, EventWebhookEndpointTest: true,
	}
	if len(known) != want {
		t.Fatalf("known set has %d entries, want %d", len(known), want)
	}
	seen := map[WebhookEventType]bool{}
	for _, e := range AllWebhookEvents {
		if !known[e] {
			t.Errorf("AllWebhookEvents contains unknown/hallucinated event %q", string(e))
		}
		if seen[e] {
			t.Errorf("AllWebhookEvents contains duplicate %q", string(e))
		}
		seen[e] = true
	}
}
