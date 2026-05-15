package opensettle

// ChainID is a supported settlement chain. Mirrors `ChainId` in
// `@opensettle/shared/schemas/wallet`.
type ChainID string

// ChainId is the original (pre-Go-idiomatic) name for ChainID. Kept as a
// type alias for backward compatibility; new code should prefer ChainID.
//
// Deprecated: use [ChainID] instead.
type ChainId = ChainID

const (
	ChainBase     ChainId = "base"
	ChainEthereum ChainId = "ethereum"
	ChainPolygon  ChainId = "polygon"
	ChainArbitrum ChainId = "arbitrum"
	ChainTron     ChainId = "tron"
	ChainSolana   ChainId = "solana"
)

// TokenSymbol is a supported stablecoin. The platform settles only in
// USDC and USDT.
type TokenSymbol string

const (
	TokenUSDC TokenSymbol = "USDC"
	TokenUSDT TokenSymbol = "USDT"
)

// CursorPage wraps a paginated list response. The API returns a cursor
// envelope around every listed collection.
type CursorPage[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore,omitempty"`
}

// Metadata is the free-form key/value blob attached to most resources.
// Values are arbitrary JSON; Go callers can marshal whatever the API
// accepts on their side. Nil = no metadata.
type Metadata map[string]any

// --- Customer ---------------------------------------------------------

// CustomerStatus is the derived health bucket OpenSettle assigns based on
// recent payment + subscription activity. It is not directly settable;
// the platform updates it as a side effect of billing events.
type CustomerStatus string

const (
	CustomerActive  CustomerStatus = "active"
	CustomerAtRisk  CustomerStatus = "at_risk"
	CustomerChurned CustomerStatus = "churned"
)

// Customer is a merchant's customer record. Email and Name are stored
// encrypted at rest; the API returns the decrypted view on Retrieve.
// ActiveSubscriptions and LifetimeValue are server-computed rollups.
// DeletedAt is set when the row is soft-deleted; reads still return it
// for audit purposes.
type Customer struct {
	ID                  string         `json:"id"`
	WorkspaceID         string         `json:"workspaceId"`
	Email               string         `json:"email"`
	Name                string         `json:"name"`
	Wallet              *string        `json:"wallet"`
	Country             *string        `json:"country"`
	Status              CustomerStatus `json:"status"`
	ActiveSubscriptions int            `json:"activeSubscriptions"`
	LifetimeValue       int            `json:"lifetimeValue"`
	Metadata            Metadata       `json:"metadata"`
	CreatedAt           string         `json:"createdAt"`
	DeletedAt           *string        `json:"deletedAt"`
}

// CreateCustomerRequest is the body for POST /customers. Email is the
// only required field; Wallet (when present) is validated against the
// workspace's enabled chains on the server.
type CreateCustomerRequest struct {
	Email    string   `json:"email"`
	Name     string   `json:"name,omitempty"`
	Wallet   string   `json:"wallet,omitempty"`
	Country  string   `json:"country,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// UpdateCustomerRequest is the body for PATCH /customers/<id>. Fields are
// PATCH-style: omit a field (leave the pointer nil) to leave the existing
// value unchanged. To clear a string field, pass a pointer to the empty
// string.
type UpdateCustomerRequest struct {
	Name     *string  `json:"name,omitempty"`
	Wallet   *string  `json:"wallet,omitempty"`
	Country  *string  `json:"country,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// ListCustomersQuery filters GET /customers. Q is a free-text search over
// email + name. Status filters by derived health bucket; leave empty for
// all. Cursor + Limit drive pagination (Limit max 100).
type ListCustomersQuery struct {
	Status CustomerStatus
	Q      string
	Cursor string
	Limit  int
}

// --- Product / Price --------------------------------------------------

// PriceInterval is the billing cadence of a Price. "one_time" is a
// non-recurring price suitable for invoices and single-payment checkouts;
// the others are recurring intervals consumed by subscriptions.
type PriceInterval string

const (
	PriceOneTime PriceInterval = "one_time"
	PriceWeek    PriceInterval = "week"
	PriceMonth   PriceInterval = "month"
	PriceYear    PriceInterval = "year"
)

// Product is a sellable item in the workspace catalog. Active=false hides
// it from new checkouts but does not affect subscriptions already using
// its prices.
type Product struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspaceId"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Active      bool     `json:"active"`
	Metadata    Metadata `json:"metadata"`
	CreatedAt   string   `json:"createdAt"`
}

// Price is a (product, amount, interval) tuple. Amount is in minor units
// (e.g. cents). Currency is the fiat denomination (USD, EUR, …); the chain
// + token used to settle are chosen per-charge, not on the price.
type Price struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspaceId"`
	ProductID   string        `json:"productId"`
	Amount      int           `json:"amount"`
	Currency    string        `json:"currency"`
	Interval    PriceInterval `json:"interval"`
	Active      bool          `json:"active"`
	Metadata    Metadata      `json:"metadata"`
	CreatedAt   string        `json:"createdAt"`
}

// CreateProductRequest is the body for POST /products.
type CreateProductRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

// UpdateProductRequest is the body for PATCH /products/<id>. Fields are
// PATCH-style: omit a field (leave the pointer nil) to leave the existing
// value unchanged. Setting Active=false hides the product from new
// checkouts.
type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Active      *bool    `json:"active,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

// CreatePriceRequest is the body for POST /products/<id>/prices. Amount
// is required and in minor units (e.g. cents). Currency defaults to the
// workspace's default fiat when empty.
type CreatePriceRequest struct {
	Amount   int           `json:"amount"`
	Currency string        `json:"currency,omitempty"`
	Interval PriceInterval `json:"interval"`
	Metadata Metadata      `json:"metadata,omitempty"`
}

// ListProductsQuery filters GET /products. Active is a tri-state pointer:
// nil = both active and inactive, &true = active only, &false = inactive
// only.
type ListProductsQuery struct {
	Cursor string
	Limit  int
	Active *bool
}

// --- Invoice ----------------------------------------------------------

// InvoiceStatus is the lifecycle state of an invoice. draft is the only
// editable state; open is sent-but-unpaid; paid and void are terminal;
// past_due is open + past DueAt.
type InvoiceStatus string

const (
	InvoiceDraft   InvoiceStatus = "draft"
	InvoiceOpen    InvoiceStatus = "open"
	InvoicePaid    InvoiceStatus = "paid"
	InvoicePastDue InvoiceStatus = "past_due"
	InvoiceVoid    InvoiceStatus = "void"
)

// LineItem is a single row on an invoice. UnitAmountMinor is in minor
// units (e.g. cents); the line total is Quantity * UnitAmountMinor.
type LineItem struct {
	Description     string `json:"description"`
	Quantity        int    `json:"quantity"`
	UnitAmountMinor int    `json:"unitAmountMinor"`
}

// Invoice is a billable document. AmountMinor is in minor units of
// Currency (fiat). Chain + Token specify how the customer will settle on
// chain. HostedURL is the buyer-facing page the customer pays from.
// PaymentID is set once a payment confirms.
type Invoice struct {
	ID             string        `json:"id"`
	WorkspaceID    string        `json:"workspaceId"`
	Number         string        `json:"number"`
	CustomerID     string        `json:"customerId"`
	SubscriptionID *string       `json:"subscriptionId"`
	AmountMinor    int           `json:"amountMinor"`
	Currency       string        `json:"currency"`
	Chain          ChainId       `json:"chain"`
	Token          TokenSymbol   `json:"token"`
	Status         InvoiceStatus `json:"status"`
	LineItems      []LineItem    `json:"lineItems"`
	Memo           *string       `json:"memo"`
	PaymentID      *string       `json:"paymentId"`
	HostedURL      string        `json:"hostedUrl"`
	IssuedAt       *string       `json:"issuedAt"`
	DueAt          string        `json:"dueAt"`
	PaidAt         *string       `json:"paidAt"`
	VoidedAt       *string       `json:"voidedAt"`
	Metadata       Metadata      `json:"metadata"`
	CreatedAt      string        `json:"createdAt"`
}

// CreateInvoiceRequest is the body for POST /invoices. DueInDays is an
// integer offset from the server clock; the API converts it to DueAt.
// SubscriptionID is set automatically for subscription-generated invoices
// and should normally be left empty by callers.
type CreateInvoiceRequest struct {
	CustomerID     string      `json:"customerId"`
	Chain          ChainId     `json:"chain"`
	Token          TokenSymbol `json:"token"`
	Currency       string      `json:"currency,omitempty"`
	LineItems      []LineItem  `json:"lineItems"`
	Memo           string      `json:"memo,omitempty"`
	DueInDays      int         `json:"dueInDays,omitempty"`
	SubscriptionID string      `json:"subscriptionId,omitempty"`
	Metadata       Metadata    `json:"metadata,omitempty"`
}

// ListInvoicesQuery filters GET /invoices. CustomerID narrows to one
// customer; Status filters by lifecycle bucket. Cursor + Limit drive
// pagination.
type ListInvoicesQuery struct {
	Cursor     string
	Limit      int
	CustomerID string
	Status     InvoiceStatus
}

// --- Payment ----------------------------------------------------------

// PaymentStatus is the on-chain lifecycle of a payment. pending means
// broadcast-but-unconfirmed; confirmed has met the workspace's required
// confirmation depth; refunded means a refund tx has confirmed; reorged
// means a previously-confirmed payment was rolled back by a chain
// reorganization.
type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentConfirmed PaymentStatus = "confirmed"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
	PaymentReorged   PaymentStatus = "reorged"
)

// Payment is a single on-chain settlement attempt. AmountMinor, FeeMinor,
// and NetMinor are denominated in minor units of Currency (fiat); the
// on-chain settlement value is in Token base units, derived at quote
// time. TxHash is set once the customer broadcasts. Refund fields are
// populated only after Refund + RefundBroadcast are called.
type Payment struct {
	ID                string        `json:"id"`
	WorkspaceID       string        `json:"workspaceId"`
	CustomerID        *string       `json:"customerId"`
	SubscriptionID    *string       `json:"subscriptionId"`
	InvoiceID         *string       `json:"invoiceId"`
	WalletID          *string       `json:"walletId"`
	AmountMinor       int           `json:"amountMinor"`
	FeeMinor          int           `json:"feeMinor"`
	NetMinor          int           `json:"netMinor"`
	Currency          string        `json:"currency"`
	Token             TokenSymbol   `json:"token"`
	Chain             ChainId       `json:"chain"`
	Status            PaymentStatus `json:"status"`
	FailureReason     *string       `json:"failureReason"`
	Description       *string       `json:"description"`
	TxHash            *string       `json:"txHash"`
	BlockNumber       *int          `json:"blockNumber"`
	Confirmations     int           `json:"confirmations"`
	RefundTxHash      *string       `json:"refundTxHash"`
	RefundAmountMinor *int          `json:"refundAmountMinor"`
	RefundBroadcastAt *string       `json:"refundBroadcastAt"`
	RefundedAt        *string       `json:"refundedAt"`
	RefundReason      *string       `json:"refundReason"`
	CreatedAt         string        `json:"createdAt"`
	ConfirmedAt       *string       `json:"confirmedAt"`
}

// InitiateRefundRequest is the body for POST /payments/<id>/refund.
// AmountMinor omitted (zero) means refund the full remaining amount.
// RecipientAddress overrides the default refund destination (the
// original payer); the server validates it against the payment's chain.
type InitiateRefundRequest struct {
	AmountMinor      int    `json:"amountMinor,omitempty"`
	Reason           string `json:"reason,omitempty"`
	RecipientAddress string `json:"recipientAddress,omitempty"`
}

// UnsignedEVMTx is the EVM-shaped portion of an unsigned refund payload.
// Value is the native-currency wei value (typically "0" for ERC-20
// transfers); AmountBaseUnits is the token amount in its smallest unit
// (e.g. 6-decimal USDC). The merchant signs this with their wallet and
// broadcasts; OpenSettle never holds keys.
type UnsignedEVMTx struct {
	To              string `json:"to"`
	Data            string `json:"data"`
	Value           string `json:"value"`
	ChainID         int    `json:"chainId"`
	TokenAddress    string `json:"tokenAddress"`
	Recipient       string `json:"recipient"`
	AmountBaseUnits string `json:"amountBaseUnits"`
}

// UnsignedEvmTx is the pre-Go-idiomatic name for [UnsignedEVMTx]. Kept as
// a type alias for backward compatibility; new code should prefer
// UnsignedEVMTx.
//
// Deprecated: use [UnsignedEVMTx] instead.
type UnsignedEvmTx = UnsignedEVMTx

// UnsignedTxEnvelope wraps a chain-agnostic refund description plus an
// optional chain-specific payload (today: EVM). Instructions is a
// human-readable hint for wallet UIs. AmountMinor is in fiat minor units;
// the chain-specific block carries the token-base-unit amount.
type UnsignedTxEnvelope struct {
	Chain        ChainId        `json:"chain"`
	Token        TokenSymbol    `json:"token"`
	To           string         `json:"to"`
	AmountMinor  int            `json:"amountMinor"`
	Memo         string         `json:"memo,omitempty"`
	Instructions string         `json:"instructions"`
	EVM          *UnsignedEvmTx `json:"evm,omitempty"`
}

// InitiateRefundResponse is the multi-key envelope returned by
// POST /payments/<id>/refund. The Payment is in status refund_pending;
// the UnsignedTx must be signed by the merchant wallet and broadcast,
// then reported back via RefundBroadcast.
type InitiateRefundResponse struct {
	Payment    Payment            `json:"payment"`
	UnsignedTx UnsignedTxEnvelope `json:"unsignedTx"`
}

// RecordRefundBroadcastRequest is the body for
// POST /payments/<id>/refund/broadcast. The caller supplies the tx hash
// returned by their wallet after broadcasting the unsigned refund tx.
type RecordRefundBroadcastRequest struct {
	RefundTxHash string `json:"refundTxHash"`
}

// ListPaymentsQuery filters GET /payments. CustomerID narrows to one
// customer; Status filters by on-chain lifecycle bucket. Cursor + Limit
// drive pagination.
type ListPaymentsQuery struct {
	Cursor     string
	Limit      int
	CustomerID string
	Status     PaymentStatus
}

// --- Subscription -----------------------------------------------------

// SubscriptionStatus is the lifecycle bucket of a subscription. trialing
// is pre-billing; active is current; past_due is one or more failed
// renewals; paused stops billing without canceling; canceled is terminal.
type SubscriptionStatus string

const (
	SubTrialing SubscriptionStatus = "trialing"
	SubActive   SubscriptionStatus = "active"
	SubPastDue  SubscriptionStatus = "past_due"
	SubPaused   SubscriptionStatus = "paused"
	SubCanceled SubscriptionStatus = "canceled"
)

// AutopayMode controls how a subscription's renewal charge is collected.
// allowance uses an ERC-20 spend approval against the merchant's
// collector; smart-wallet uses a session-key-style preauthorization;
// manual prompts the customer to sign each renewal in their wallet.
type AutopayMode string

const (
	AutopayAllowance   AutopayMode = "allowance"
	AutopaySmartWallet AutopayMode = "smart-wallet"
	AutopayManual      AutopayMode = "manual"
)

// Subscription is a recurring billing arrangement. AmountMinor is in
// minor units of Currency (fiat); MRRMinor is the normalized monthly
// recurring revenue contribution. AllowanceTx/AllowanceRemaining are
// populated only when Autopay=allowance. CurrentPeriodEnd and
// NextBillingDate are server-managed.
type Subscription struct {
	ID                 string             `json:"id"`
	WorkspaceID        string             `json:"workspaceId"`
	CustomerID         string             `json:"customerId"`
	ProductID          string             `json:"productId"`
	PriceID            string             `json:"priceId"`
	AmountMinor        int                `json:"amountMinor"`
	Currency           string             `json:"currency"`
	Chain              ChainId            `json:"chain"`
	Token              TokenSymbol        `json:"token"`
	Status             SubscriptionStatus `json:"status"`
	Autopay            AutopayMode        `json:"autopay"`
	AllowanceTx        *string            `json:"allowanceTx"`
	AllowanceRemaining *int               `json:"allowanceRemaining"`
	TrialEndsAt        *string            `json:"trialEndsAt"`
	StartedAt          string             `json:"startedAt"`
	CurrentPeriodEnd   string             `json:"currentPeriodEnd"`
	NextBillingDate    string             `json:"nextBillingDate"`
	CanceledAt         *string            `json:"canceledAt"`
	CancelReason       *string            `json:"cancelReason"`
	PausedAt           *string            `json:"pausedAt"`
	MRRMinor           int                `json:"mrrMinor"`
	Metadata           Metadata           `json:"metadata"`
	CreatedAt          string             `json:"createdAt"`
}

// CreateSubscriptionRequest is the body for POST /subscriptions. Autopay
// defaults to manual when empty. TrialDays > 0 starts the subscription
// in trialing status; the first charge fires at trial end.
type CreateSubscriptionRequest struct {
	CustomerID string      `json:"customerId"`
	PriceID    string      `json:"priceId"`
	Chain      ChainId     `json:"chain"`
	Token      TokenSymbol `json:"token"`
	Autopay    AutopayMode `json:"autopay,omitempty"`
	TrialDays  int         `json:"trialDays,omitempty"`
	Metadata   Metadata    `json:"metadata,omitempty"`
}

// ProrationMode controls when a plan change takes effect.
type ProrationMode string

const (
	ProrationImmediately ProrationMode = "immediately"
	ProrationAtPeriodEnd ProrationMode = "at_period_end"
)

// ChangePlanRequest is the body for POST /subscriptions/<id>/change_plan.
// ProrationMode controls when the swap takes effect; default is
// immediately when empty.
type ChangePlanRequest struct {
	PriceID       string        `json:"priceId"`
	ProrationMode ProrationMode `json:"prorationMode,omitempty"`
}

// CancelMode controls whether cancellation is immediate or deferred to
// the next billing boundary.
type CancelMode string

const (
	CancelImmediately CancelMode = "immediately"
	CancelAtPeriodEnd CancelMode = "at_period_end"
)

// CancelSubscriptionRequest is the body for POST /subscriptions/<id>/cancel.
// Reason is recorded on the audit log. Mode defaults to at_period_end
// when empty.
type CancelSubscriptionRequest struct {
	Reason string     `json:"reason,omitempty"`
	Mode   CancelMode `json:"mode,omitempty"`
}

// ListSubscriptionsQuery filters GET /subscriptions. CustomerID narrows
// to one customer; Status filters by lifecycle bucket. Cursor + Limit
// drive pagination.
type ListSubscriptionsQuery struct {
	Cursor     string
	Limit      int
	CustomerID string
	Status     SubscriptionStatus
}

// --- Checkout ---------------------------------------------------------

// CheckoutMode selects what a hosted checkout session does. "payment" is
// a one-shot charge (typically attached to an invoice); "subscription"
// creates a recurring Subscription on successful completion.
type CheckoutMode string

const (
	CheckoutPayment      CheckoutMode = "payment"
	CheckoutSubscription CheckoutMode = "subscription"
)

// CheckoutStatus is the lifecycle bucket of a hosted checkout session.
// open is awaiting buyer; pending is buyer signed and broadcast,
// awaiting chain confirmation; succeeded, failed, and expired are
// terminal.
type CheckoutStatus string

const (
	CheckoutOpen      CheckoutStatus = "open"
	CheckoutPending   CheckoutStatus = "pending"
	CheckoutSucceeded CheckoutStatus = "succeeded"
	CheckoutFailed    CheckoutStatus = "failed"
	CheckoutExpired   CheckoutStatus = "expired"
)

// Checkout is a hosted payment session. HostedURL is the relative path
// (e.g. "/checkout/<token>") to redirect the buyer to; concatenate with
// the OpenSettle web origin. AmountMinor + Currency are populated once
// the underlying invoice/price is resolved.
type Checkout struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspaceId"`
	Mode        CheckoutMode   `json:"mode"`
	Status      CheckoutStatus `json:"status"`
	CustomerID  string         `json:"customerId"`
	InvoiceID   *string        `json:"invoiceId"`
	PriceID     *string        `json:"priceId"`
	AmountMinor int            `json:"amountMinor"`
	Currency    string         `json:"currency"`
	Chain       ChainId        `json:"chain"`
	Token       TokenSymbol    `json:"token"`
	Description *string        `json:"description"`
	SuccessURL  string         `json:"successUrl"`
	CancelURL   *string        `json:"cancelUrl"`
	ExpiresAt   string         `json:"expiresAt"`
	CompletedAt *string        `json:"completedAt"`
	Metadata    Metadata       `json:"metadata"`
	CreatedAt   string         `json:"createdAt"`
	// HostedURL is a relative URL path (e.g. "/checkout/<hostedToken>");
	// concatenate with the web origin (e.g. "https://opensettle.io"+HostedURL)
	// to get the buyer-facing redirect URL.
	HostedURL string `json:"hostedUrl"`
}

// CreateCheckoutRequest is the body for POST /checkouts. Exactly one of
// (CustomerID) or (CustomerEmail [+ CustomerName]) should be supplied —
// the latter form auto-creates a Customer on the fly. For Mode=payment
// pass InvoiceID; for Mode=subscription pass PriceID. Chain/Token are
// optional pre-selections; if omitted, the buyer picks on the hosted
// page.
//
// Hosted checkout is currently EVM-only (Base, Ethereum, Polygon,
// Arbitrum). The Chain field's type accepts "solana" and "tron" — the
// API + wallet-verification layer support them and the chain reader
// will detect inbound SPL / TRC-20 deposits to verified wallets — but
// the customer-facing hosted checkout page does not yet render those
// networks. Pass an EVM ChainId here, or omit Chain and let the buyer
// pick on the hosted page (only EVM options will appear).
type CreateCheckoutRequest struct {
	Mode             CheckoutMode `json:"mode"`
	CustomerID       string       `json:"customerId,omitempty"`
	CustomerEmail    string       `json:"customerEmail,omitempty"`
	CustomerName     string       `json:"customerName,omitempty"`
	InvoiceID        string       `json:"invoiceId,omitempty"`
	PriceID          string       `json:"priceId,omitempty"`
	SuccessURL       string       `json:"successUrl"`
	CancelURL        string       `json:"cancelUrl,omitempty"`
	Chain            ChainId      `json:"chain,omitempty"`
	Token            TokenSymbol  `json:"token,omitempty"`
	ExpiresInMinutes int          `json:"expiresInMinutes,omitempty"`
	Metadata         Metadata     `json:"metadata,omitempty"`
}

// --- Webhook endpoint -------------------------------------------------

// WebhookEndpointStatus is the delivery state of a webhook endpoint.
// disabled stops new deliveries (existing in-flight retries still run);
// enabled is the normal state.
type WebhookEndpointStatus string

const (
	WebhookEnabled  WebhookEndpointStatus = "enabled"
	WebhookDisabled WebhookEndpointStatus = "disabled"
)

// WebhookEndpoint is a merchant-configured HTTPS destination for event
// deliveries. SuccessRate is the server-computed rolling success ratio
// over the recent delivery window. RotationGraceUntil is set during a
// signing-secret rotation: until then, both the old and new secrets
// produce valid signatures.
type WebhookEndpoint struct {
	ID                 string                `json:"id"`
	WorkspaceID        string                `json:"workspaceId"`
	URL                string                `json:"url"`
	Description        *string               `json:"description"`
	Events             []string              `json:"events"`
	Status             WebhookEndpointStatus `json:"status"`
	SuccessRate        float64               `json:"successRate"`
	RotationGraceUntil *string               `json:"rotationGraceUntil"`
	CreatedAt          string                `json:"createdAt"`
}

// CreateWebhookEndpointRequest is the body for POST /webhook_endpoints.
// Events is an allow-list of event-type names (e.g. "payment.confirmed");
// omit or pass an empty slice to subscribe to all events.
type CreateWebhookEndpointRequest struct {
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	Events      []string `json:"events,omitempty"`
}

// CreateWebhookEndpointResponse is the multi-key envelope returned by
// POST /webhook_endpoints (and by /rotate). SigningSecret is the plaintext
// HMAC secret returned exactly once — store it immediately; subsequent
// reads only return the endpoint metadata.
type CreateWebhookEndpointResponse struct {
	Endpoint      WebhookEndpoint `json:"endpoint"`
	SigningSecret string          `json:"signingSecret"`
}

// UpdateWebhookEndpointRequest is the body for PATCH /webhook_endpoints/<id>.
// Fields are PATCH-style: omit a field (leave the pointer nil) to leave
// the existing value unchanged. Passing a non-nil Events replaces the
// allow-list entirely.
type UpdateWebhookEndpointRequest struct {
	URL         *string                `json:"url,omitempty"`
	Description *string                `json:"description,omitempty"`
	Events      []string               `json:"events,omitempty"`
	Status      *WebhookEndpointStatus `json:"status,omitempty"`
}

// RotateWebhookSecretRequest is the body for POST /webhook_endpoints/<id>/rotate.
// GraceSeconds is the dual-signing window during which both old and new
// secrets produce valid signatures; default is server-side when zero.
type RotateWebhookSecretRequest struct {
	GraceSeconds int `json:"graceSeconds,omitempty"`
}

// RotateWebhookSecretResponse is now an alias for
// CreateWebhookEndpointResponse — the rotate endpoint returns the same
// {endpoint, signingSecret} envelope as create.
//
// Deprecated: use *CreateWebhookEndpointResponse directly. This alias
// will be removed in a future release.
type RotateWebhookSecretResponse = CreateWebhookEndpointResponse

// TestWebhookEndpointRequest is the body for POST /webhook_endpoints/<id>/test.
// EventType is the event-type name to fire as a sample (e.g.
// "payment.confirmed"); the payload is server-generated.
type TestWebhookEndpointRequest struct {
	EventType string `json:"eventType"`
}

// TestWebhookEndpointResponse reports the result of a synchronous test
// delivery. Status is the HTTP status the endpoint returned to
// OpenSettle; LatencyMs is the round-trip latency observed by the
// dispatcher. OK is true when Status was 2xx.
type TestWebhookEndpointResponse struct {
	OK        bool `json:"ok"`
	Status    int  `json:"status"`
	LatencyMs int  `json:"latencyMs"`
}

// rawList wraps an envelope shape some endpoints use: { "data": [...] }
// without nextCursor. Kept private — resource code unmarshals into it
// and returns the inner slice.
type rawList[T any] struct {
	Data []T `json:"data"`
}
