package opensettle

import (
	"encoding/json"
	"fmt"
)

// ErrorCode is the API's stable error taxonomy. Values mirror
// `packages/sdk/src/errors.ts` exactly.
type ErrorCode string

const (
	CodeInvalidRequest            ErrorCode = "invalid_request"
	CodeInvalidStateTransition    ErrorCode = "invalid_state_transition"
	CodeUnauthorized              ErrorCode = "unauthorized"
	CodeForbidden                 ErrorCode = "forbidden"
	CodeNotFound                  ErrorCode = "not_found"
	CodeConflict                  ErrorCode = "conflict"
	CodeRateLimited               ErrorCode = "rate_limited"
	CodeInternalError             ErrorCode = "internal_error"
	CodeChainReverted             ErrorCode = "chain_reverted"
	CodeInsufficientConfirmations ErrorCode = "insufficient_confirmations"
	CodeSigningRequired           ErrorCode = "signing_required"
	CodeAALRequired               ErrorCode = "aal_required"
	CodeNetworkError              ErrorCode = "network_error"
)

// OpenSettleError is the base type. Every concrete error in this package
// embeds it, so callers can either match the broad type via
// errors.As(err, &*OpenSettleError{}) or branch on the specific subtype.
type OpenSettleError struct {
	Code      ErrorCode
	Message   string
	Status    int
	RequestID string
	Param     string
}

func (e *OpenSettleError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("opensettle: %s (code=%s, status=%d, request_id=%s)", e.Message, e.Code, e.Status, e.RequestID)
	}
	return fmt.Sprintf("opensettle: %s (code=%s, status=%d)", e.Message, e.Code, e.Status)
}

// Concrete subtypes — one per stable error code family. Each embeds
// *OpenSettleError so the base fields are accessible directly.

type InvalidRequestError struct{ *OpenSettleError }

func (e *InvalidRequestError) Unwrap() error { return e.OpenSettleError }

type InvalidStateTransitionError struct{ *OpenSettleError }

func (e *InvalidStateTransitionError) Unwrap() error { return e.OpenSettleError }

type AuthenticationError struct{ *OpenSettleError }

func (e *AuthenticationError) Unwrap() error { return e.OpenSettleError }

type ForbiddenError struct{ *OpenSettleError }

func (e *ForbiddenError) Unwrap() error { return e.OpenSettleError }

type NotFoundError struct{ *OpenSettleError }

func (e *NotFoundError) Unwrap() error { return e.OpenSettleError }

type ConflictError struct{ *OpenSettleError }

func (e *ConflictError) Unwrap() error { return e.OpenSettleError }

// RateLimitError carries the optional Retry-After hint advertised by the
// API. Value is in seconds. A zero RetryAfter means the server didn't
// advertise one — caller should fall back to its own backoff.
type RateLimitError struct {
	*OpenSettleError
	RetryAfter float64
}

type SettlementError struct{ *OpenSettleError }

func (e *SettlementError) Unwrap() error { return e.OpenSettleError }

type StepUpRequiredError struct{ *OpenSettleError }

func (e *StepUpRequiredError) Unwrap() error { return e.OpenSettleError }

type APIError struct{ *OpenSettleError }

func (e *APIError) Unwrap() error { return e.OpenSettleError }

type NetworkError struct{ *OpenSettleError }

func (e *NetworkError) Unwrap() error { return e.OpenSettleError }

func (e *RateLimitError) Unwrap() error { return e.OpenSettleError }

// errorEnvelope is the JSON shape the API returns on non-2xx.
type errorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Param     string `json:"param,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	} `json:"error"`
}

// FromEnvelope maps a parsed envelope + HTTP status to the right typed
// error. Unknown codes fall back to *APIError so a new server-side code
// can't crash older SDKs.
//
// retryAfter is the parsed Retry-After header value (seconds); 0 means
// the server didn't send one.
func FromEnvelope(body []byte, status int, retryAfter float64) error {
	var env errorEnvelope
	if len(body) > 0 {
		// Best-effort decode. A non-JSON 5xx body is normal during
		// outages — fall through with empty fields.
		_ = json.Unmarshal(body, &env)
	}

	code := ErrorCode(env.Error.Code)
	message := env.Error.Message
	if message == "" {
		message = fmt.Sprintf("Request failed with status %d", status)
	}

	base := &OpenSettleError{
		Code:      code,
		Message:   message,
		Status:    status,
		RequestID: env.Error.RequestID,
		Param:     env.Error.Param,
	}

	switch code {
	case CodeInvalidRequest:
		return &InvalidRequestError{base}
	case CodeInvalidStateTransition:
		return &InvalidStateTransitionError{base}
	case CodeUnauthorized:
		return &AuthenticationError{base}
	case CodeForbidden:
		return &ForbiddenError{base}
	case CodeNotFound:
		return &NotFoundError{base}
	case CodeConflict:
		return &ConflictError{base}
	case CodeRateLimited:
		return &RateLimitError{OpenSettleError: base, RetryAfter: retryAfter}
	case CodeChainReverted, CodeInsufficientConfirmations, CodeSigningRequired:
		return &SettlementError{base}
	case CodeAALRequired:
		return &StepUpRequiredError{base}
	case CodeInternalError:
		return &APIError{base}
	default:
		// Unknown or empty code — preserve the raw value when present so
		// debugging isn't blind, but classify as APIError.
		if base.Code == "" {
			base.Code = CodeInternalError
		}
		return &APIError{base}
	}
}

// newNetworkError constructs a transport-layer failure (DNS, ECONNREFUSED,
// context-deadline, etc.). status is 0 since no response was received.
func newNetworkError(message string) error {
	return &NetworkError{&OpenSettleError{
		Code:    CodeNetworkError,
		Message: message,
		Status:  0,
	}}
}
