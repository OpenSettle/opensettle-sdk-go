package opensettle

import (
	"errors"
	"strings"
	"testing"
)

func TestAllErrorSubtypes_UnwrapToOpenSettleError(t *testing.T) {
	base := &OpenSettleError{Code: CodeInternalError, Message: "test", Status: 500}
	cases := []struct {
		name string
		err  error
	}{
		{"InvalidRequest", &InvalidRequestError{base}},
		{"InvalidStateTransition", &InvalidStateTransitionError{base}},
		{"Authentication", &AuthenticationError{base}},
		{"Forbidden", &ForbiddenError{base}},
		{"NotFound", &NotFoundError{base}},
		{"Conflict", &ConflictError{base}},
		{"RateLimit", &RateLimitError{OpenSettleError: base, RetryAfter: 5}},
		{"Settlement", &SettlementError{base}},
		{"StepUpRequired", &StepUpRequiredError{base}},
		{"API", &APIError{base}},
		{"Network", &NetworkError{base}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Direct Unwrap must return the embedded base.
			unwrapped := errors.Unwrap(tc.err)
			if unwrapped == nil {
				t.Fatalf("%s: errors.Unwrap returned nil", tc.name)
			}
			var got *OpenSettleError
			if !errors.As(unwrapped, &got) {
				t.Fatalf("%s: unwrapped err is not *OpenSettleError: %T", tc.name, unwrapped)
			}
			if got != base {
				t.Fatalf("%s: unwrapped pointer mismatch", tc.name)
			}
			// errors.As walking through subtype → base must succeed.
			var asBase *OpenSettleError
			if !errors.As(tc.err, &asBase) {
				t.Fatalf("%s: errors.As to *OpenSettleError failed", tc.name)
			}
			// IsOpenSettleError convenience must match.
			if !IsOpenSettleError(tc.err) {
				t.Fatalf("%s: IsOpenSettleError returned false", tc.name)
			}
		})
	}
}

func TestFromEnvelope_InvalidRequest(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"invalid_request","message":"bad","param":"email","request_id":"req_1"}}`), 400, 0)
	var target *InvalidRequestError
	if !errors.As(err, &target) {
		t.Fatalf("want *InvalidRequestError, got %T", err)
	}
	if target.Code != CodeInvalidRequest {
		t.Fatalf("code: %s", target.Code)
	}
	if target.Status != 400 {
		t.Fatalf("status: %d", target.Status)
	}
	if target.RequestID != "req_1" {
		t.Fatalf("requestID: %s", target.RequestID)
	}
	if target.Param != "email" {
		t.Fatalf("param: %s", target.Param)
	}
	if target.Message != "bad" {
		t.Fatalf("message: %s", target.Message)
	}
}

func TestFromEnvelope_InvalidStateTransition(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"invalid_state_transition","message":"bad"}}`), 422, 0)
	var target *InvalidStateTransitionError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_Unauthorized(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"unauthorized","message":"no"}}`), 401, 0)
	var target *AuthenticationError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_Forbidden(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"forbidden","message":"no"}}`), 403, 0)
	var target *ForbiddenError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_NotFound(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"not_found","message":"gone"}}`), 404, 0)
	var target *NotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_Conflict(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"conflict","message":"dup"}}`), 409, 0)
	var target *ConflictError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_RateLimited(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"rate_limited","message":"slow"}}`), 429, 12)
	var target *RateLimitError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.RetryAfter != 12 {
		t.Fatalf("retryAfter: %v", target.RetryAfter)
	}
}

func TestFromEnvelope_RateLimitedNoRetryAfter(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"rate_limited","message":"slow"}}`), 429, 0)
	var target *RateLimitError
	if !errors.As(err, &target) {
		t.Fatal()
	}
	if target.RetryAfter != 0 {
		t.Fatalf("retryAfter: %v", target.RetryAfter)
	}
}

func TestFromEnvelope_ChainReverted(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"chain_reverted","message":"x"}}`), 400, 0)
	var target *SettlementError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Code != CodeChainReverted {
		t.Fatalf("code: %s", target.Code)
	}
}

func TestFromEnvelope_InsufficientConfirmations(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"insufficient_confirmations","message":"x"}}`), 400, 0)
	var target *SettlementError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_SigningRequired(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"signing_required","message":"x"}}`), 400, 0)
	var target *SettlementError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_AALRequired(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"aal_required","message":"x"}}`), 401, 0)
	var target *StepUpRequiredError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_InternalError(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"internal_error","message":"x"}}`), 500, 0)
	var target *APIError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_UnknownCodeFallsBackToAPIError(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"someday_invented","message":"x"}}`), 500, 0)
	var target *APIError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_EmptyBody(t *testing.T) {
	err := FromEnvelope(nil, 502, 0)
	var target *APIError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if !strings.Contains(target.Message, "502") {
		t.Fatalf("message: %s", target.Message)
	}
	if target.Status != 502 {
		t.Fatalf("status: %d", target.Status)
	}
}

func TestFromEnvelope_NonJSONBody(t *testing.T) {
	err := FromEnvelope([]byte("not json"), 503, 0)
	var target *APIError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
}

func TestFromEnvelope_RequestIDPropagation(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"not_found","message":"x","request_id":"req_xyz"}}`), 404, 0)
	var target *NotFoundError
	if !errors.As(err, &target) {
		t.Fatal()
	}
	if target.RequestID != "req_xyz" {
		t.Fatalf("got %s", target.RequestID)
	}
}

func TestFromEnvelope_BaseTypeAssertable(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"not_found","message":"x"}}`), 404, 0)
	var base *OpenSettleError
	if !errors.As(err, &base) {
		t.Fatalf("could not extract base type")
	}
}

func TestOpenSettleError_StringWithoutRequestID(t *testing.T) {
	err := &OpenSettleError{Code: CodeNotFound, Message: "gone", Status: 404}
	if !strings.Contains(err.Error(), "code=not_found") {
		t.Fatalf("got %q", err.Error())
	}
	if strings.Contains(err.Error(), "request_id") {
		t.Fatalf("should omit empty request_id: %q", err.Error())
	}
}

func TestOpenSettleError_StringWithRequestID(t *testing.T) {
	err := &OpenSettleError{Code: CodeNotFound, Message: "gone", Status: 404, RequestID: "req_1"}
	if !strings.Contains(err.Error(), "request_id=req_1") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestNetworkError_StatusZero(t *testing.T) {
	err := newNetworkError("dns failure")
	var target *NetworkError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Status != 0 {
		t.Fatalf("status: %d", target.Status)
	}
	if target.Code != CodeNetworkError {
		t.Fatalf("code: %s", target.Code)
	}
}

func TestFromEnvelope_EmptyCodeBecomesInternalError(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"message":"x"}}`), 500, 0)
	var target *APIError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Code != CodeInternalError {
		t.Fatalf("code: %s", target.Code)
	}
}

func TestFromEnvelope_PreservesAllFields(t *testing.T) {
	err := FromEnvelope([]byte(`{"error":{"code":"conflict","message":"dup","param":"id","request_id":"req_2"}}`), 409, 0)
	var target *ConflictError
	if !errors.As(err, &target) {
		t.Fatal()
	}
	if target.Code != CodeConflict {
		t.Fatal("code")
	}
	if target.Status != 409 {
		t.Fatal("status")
	}
	if target.Message != "dup" {
		t.Fatal("message")
	}
	if target.Param != "id" {
		t.Fatal("param")
	}
	if target.RequestID != "req_2" {
		t.Fatal("requestID")
	}
}

func TestFromEnvelope_MalformedJSON(t *testing.T) {
	err := FromEnvelope([]byte("{not json"), 500, 0)
	var target *APIError
	if !errors.As(err, &target) {
		t.Fatalf("got %T", err)
	}
	if target.Status != 500 {
		t.Fatal()
	}
}

// Table-driven sanity sweep — every code must round-trip.
func TestFromEnvelope_AllCodes(t *testing.T) {
	cases := []struct {
		code   ErrorCode
		isType func(error) bool
	}{
		{CodeInvalidRequest, func(e error) bool { return errors.As(e, new(*InvalidRequestError)) }},
		{CodeInvalidStateTransition, func(e error) bool { return errors.As(e, new(*InvalidStateTransitionError)) }},
		{CodeUnauthorized, func(e error) bool { return errors.As(e, new(*AuthenticationError)) }},
		{CodeForbidden, func(e error) bool { return errors.As(e, new(*ForbiddenError)) }},
		{CodeNotFound, func(e error) bool { return errors.As(e, new(*NotFoundError)) }},
		{CodeConflict, func(e error) bool { return errors.As(e, new(*ConflictError)) }},
		{CodeRateLimited, func(e error) bool { return errors.As(e, new(*RateLimitError)) }},
		{CodeChainReverted, func(e error) bool { return errors.As(e, new(*SettlementError)) }},
		{CodeInsufficientConfirmations, func(e error) bool { return errors.As(e, new(*SettlementError)) }},
		{CodeSigningRequired, func(e error) bool { return errors.As(e, new(*SettlementError)) }},
		{CodeAALRequired, func(e error) bool { return errors.As(e, new(*StepUpRequiredError)) }},
		{CodeInternalError, func(e error) bool { return errors.As(e, new(*APIError)) }},
	}
	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			body := `{"error":{"code":"` + string(tc.code) + `","message":"x"}}`
			err := FromEnvelope([]byte(body), 400, 0)
			if !tc.isType(err) {
				t.Fatalf("wrong type for %s: %T", tc.code, err)
			}
		})
	}
}

func TestErrorCode_StringValues(t *testing.T) {
	cases := map[ErrorCode]string{
		CodeInvalidRequest:            "invalid_request",
		CodeInvalidStateTransition:    "invalid_state_transition",
		CodeUnauthorized:              "unauthorized",
		CodeForbidden:                 "forbidden",
		CodeNotFound:                  "not_found",
		CodeConflict:                  "conflict",
		CodeRateLimited:               "rate_limited",
		CodeInternalError:             "internal_error",
		CodeChainReverted:             "chain_reverted",
		CodeInsufficientConfirmations: "insufficient_confirmations",
		CodeSigningRequired:           "signing_required",
		CodeAALRequired:               "aal_required",
		CodeNetworkError:              "network_error",
	}
	for code, want := range cases {
		if string(code) != want {
			t.Errorf("%v: want %q, got %q", code, want, string(code))
		}
	}
}
