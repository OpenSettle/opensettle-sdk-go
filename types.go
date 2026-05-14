package opensettle

import "encoding/json"

// ChainId is a supported settlement chain. Mirrors `ChainId` in
// `@opensettle/shared/schemas/wallet`.
type ChainId string

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

type CustomerStatus string

const (
	CustomerActive  CustomerStatus = "active"
	CustomerAtRisk  CustomerStatus = "at_risk"
	CustomerChurned CustomerStatus = "churned"
)

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

type CreateCustomerRequest struct {
	Email    string   `json:"email"`
	Name     string   `json:"name,omitempty"`
	Wallet   string   `json:"wallet,omitempty"`
	Country  string   `json:"country,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

type UpdateCustomerRequest struct {
	Name     *string  `json:"name,omitempty"`
	Wallet   *string  `json:"wallet,omitempty"`
	Country  *string  `json:"country,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

type ListCustomersQuery struct {
	Status CustomerStatus
	Q      string
	Cursor string
	Limit  int
}

// --- Product / Price --------------------------------------------------

type PriceInterval string

const (
	PriceOneTime PriceInterval = "one_time"
	PriceWeek    PriceInterval = "week"
	PriceMonth   PriceInterval = "month"
	PriceYear    PriceInterval = "year"
)

type Product struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspaceId"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Active      bool     `json:"active"`
	Metadata    Metadata `json:"metadata"`
	CreatedAt   string   `json:"createdAt"`
}

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

type CreateProductRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Active      *bool    `json:"active,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

type CreatePriceRequest struct {
	Amount   int           `json:"amount"`
	Currency string        `json:"currency,omitempty"`
	Interval PriceInterval `json:"interval"`
	Metadata Metadata      `json:"metadata,omitempty"`
}

type ListProductsQuery struct {
	Cursor string
	Limit  int
	Active *bool
}

// --- Invoice ----------------------------------------------------------

type InvoiceStatus string

const (
	InvoiceDraft   InvoiceStatus = "draft"
	InvoiceOpen    InvoiceStatus = "open"
	InvoicePaid    InvoiceStatus = "paid"
	InvoicePastDue InvoiceStatus = "past_due"
	InvoiceVoid    InvoiceStatus = "void"
)

type LineItem struct {
	Description     string `json:"description"`
	Quantity        int    `json:"quantity"`
	UnitAmountMinor int    `json:"unitAmountMinor"`
}

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

type ListInvoicesQuery struct {
	Cursor     string
	Limit      int
	CustomerID string
	Status     InvoiceStatus
}

// --- Payment ----------------------------------------------------------

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentConfirmed PaymentStatus = "confirmed"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
	PaymentReorged   PaymentStatus = "reorged"
)

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

type InitiateRefundRequest struct {
	AmountMinor      int    `json:"amountMinor,omitempty"`
	Reason           string `json:"reason,omitempty"`
	RecipientAddress string `json:"recipientAddress,omitempty"`
}

type UnsignedEvmTx struct {
	To              string `json:"to"`
	Data            string `json:"data"`
	Value           string `json:"value"`
	ChainID         int    `json:"chainId"`
	TokenAddress    string `json:"tokenAddress"`
	Recipient       string `json:"recipient"`
	AmountBaseUnits string `json:"amountBaseUnits"`
}

type UnsignedTxEnvelope struct {
	Chain        ChainId        `json:"chain"`
	Token        TokenSymbol    `json:"token"`
	To           string         `json:"to"`
	AmountMinor  int            `json:"amountMinor"`
	Memo         string         `json:"memo,omitempty"`
	Instructions string         `json:"instructions"`
	EVM          *UnsignedEvmTx `json:"evm,omitempty"`
}

type InitiateRefundResponse struct {
	Payment    Payment            `json:"payment"`
	UnsignedTx UnsignedTxEnvelope `json:"unsignedTx"`
}

type RecordRefundBroadcastRequest struct {
	RefundTxHash string `json:"refundTxHash"`
}

type ListPaymentsQuery struct {
	Cursor     string
	Limit      int
	CustomerID string
	Status     PaymentStatus
}

// --- Subscription -----------------------------------------------------

type SubscriptionStatus string

const (
	SubTrialing SubscriptionStatus = "trialing"
	SubActive   SubscriptionStatus = "active"
	SubPastDue  SubscriptionStatus = "past_due"
	SubPaused   SubscriptionStatus = "paused"
	SubCanceled SubscriptionStatus = "canceled"
)

type AutopayMode string

const (
	AutopayAllowance   AutopayMode = "allowance"
	AutopaySmartWallet AutopayMode = "smart-wallet"
	AutopayManual      AutopayMode = "manual"
)

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

type CancelSubscriptionRequest struct {
	Reason string     `json:"reason,omitempty"`
	Mode   CancelMode `json:"mode,omitempty"`
}

type ListSubscriptionsQuery struct {
	Cursor     string
	Limit      int
	CustomerID string
	Status     SubscriptionStatus
}

// --- Checkout ---------------------------------------------------------

type CheckoutMode string

const (
	CheckoutPayment      CheckoutMode = "payment"
	CheckoutSubscription CheckoutMode = "subscription"
)

type CheckoutStatus string

const (
	CheckoutOpen      CheckoutStatus = "open"
	CheckoutPending   CheckoutStatus = "pending"
	CheckoutSucceeded CheckoutStatus = "succeeded"
	CheckoutFailed    CheckoutStatus = "failed"
	CheckoutExpired   CheckoutStatus = "expired"
)

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

type WebhookEndpointStatus string

const (
	WebhookEnabled  WebhookEndpointStatus = "enabled"
	WebhookDisabled WebhookEndpointStatus = "disabled"
)

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

type CreateWebhookEndpointRequest struct {
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	Events      []string `json:"events,omitempty"`
}

type CreateWebhookEndpointResponse struct {
	Endpoint      WebhookEndpoint `json:"endpoint"`
	SigningSecret string          `json:"signingSecret"`
}

type UpdateWebhookEndpointRequest struct {
	URL         *string                `json:"url,omitempty"`
	Description *string                `json:"description,omitempty"`
	Events      []string               `json:"events,omitempty"`
	Status      *WebhookEndpointStatus `json:"status,omitempty"`
}

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

type TestWebhookEndpointRequest struct {
	EventType string `json:"eventType"`
}

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

// jsonRawShim helps tests assert the JSON the SDK sent. Not used by
// production code.
var _ = json.RawMessage(nil)
