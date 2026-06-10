// Package webhooks verifies signed OpenSettle webhook deliveries.
//
// The signing scheme matches the API's deliverer exactly:
//
//	header  : x-opensettle-signature: t=<unix>,v1=<hex_hmac_sha256>
//	message : <unix_seconds>.<raw_body>
//	secret  : the per-endpoint signing secret returned by
//	          POST /v1/workspaces/:ws/webhook_endpoints (or its
//	          rotation endpoint)
//
// Verify is constant-time. It returns a typed *VerificationError on
// every failure path so handlers can return 400 with confidence —
// the request didn't come from OpenSettle.
package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Reason names every distinct failure surface so handlers can branch.
type Reason string

const (
	ReasonMissingHeader     Reason = "missing_header"
	ReasonMalformedHeader   Reason = "malformed_header"
	ReasonStaleTimestamp    Reason = "stale_timestamp"
	ReasonSignatureMismatch Reason = "signature_mismatch"
	ReasonInvalidBody       Reason = "invalid_body"
	// ReasonMissingSecret fires when Verify is called with an empty
	// Secret — surfaces a deployment-time misconfiguration loudly
	// instead of silently HMAC-ing with an empty key and reporting every
	// delivery as ReasonSignatureMismatch.
	//
	// 2026-05-20 night-run round 6 — NR6 cross-SDK F-2 parity with Node.
	ReasonMissingSecret Reason = "missing_secret"
)

// VerificationError is the only error type Verify returns. Handlers
// typically just respond with HTTP 400 — every reason indicates the
// request was not authentic.
type VerificationError struct {
	Reason  Reason
	Message string
}

func (e *VerificationError) Error() string {
	return fmt.Sprintf("webhooks: %s: %s", e.Reason, e.Message)
}

// Verified wraps the decoded body alongside the timestamp the signing
// payload was anchored on.
type Verified struct {
	Body      json.RawMessage
	Timestamp int64
}

// Opts are the inputs Verify needs. RawBody is the exact request body
// bytes — verification anchors on those exact bytes, so re-marshaling
// the JSON before passing it in will break the signature.
type Opts struct {
	RawBody         []byte
	SignatureHeader string
	Secret          string
	// Tolerance bounds how stale a timestamp may be. Default 5 minutes
	// (the zero value triggers the default — Go-style ergonomics, but
	// see DisableTimestampCheck if you actually want to skip the gate).
	Tolerance time.Duration
	// DisableTimestampCheck skips the tolerance window entirely.
	// Intended for replay-driven tests and historical backfills.
	//
	// 2026-05-20 night-run round 6 — NR6 cross-SDK F-1: prior to this
	// flag, passing Tolerance: 0 silently re-enabled the 5-minute
	// default (Go's zero-value convention), which diverged from the
	// Node/Python/Rust SDKs where 0 is the documented opt-out. The new
	// flag preserves the zero-value-means-default behaviour for
	// existing callers while restoring cross-SDK parity for new ones.
	DisableTimestampCheck bool
	// Now is the reference time used for tolerance checks. Defaults to
	// time.Now. Override in tests.
	Now func() time.Time
}

const defaultTolerance = 5 * time.Minute

// Verify checks the signature and tolerance window, returning the
// parsed body and timestamp on success.
func Verify(opts Opts) (*Verified, error) {
	// 2026-05-20 night-run round 6 — NR6 cross-SDK F-2: loud-fail on an
	// empty secret to match the Node SDK. A merchant deploying with
	// OPENSETTLE_WEBHOOK_SECRET unset/empty would otherwise see every
	// delivery rejected as ReasonSignatureMismatch (HMAC over the empty
	// key) and assume their endpoint is under attack.
	if opts.Secret == "" {
		return nil, &VerificationError{
			Reason:  ReasonMissingSecret,
			Message: "OpenSettle webhook secret is empty — set the endpoint's `whsec_…` value before verifying deliveries",
		}
	}

	if opts.SignatureHeader == "" {
		return nil, &VerificationError{Reason: ReasonMissingHeader, Message: "missing x-opensettle-signature header"}
	}

	ts, v1, err := parseHeader(opts.SignatureHeader)
	if err != nil {
		return nil, err
	}

	tolerance := opts.Tolerance
	if tolerance == 0 {
		tolerance = defaultTolerance
	}
	if !opts.DisableTimestampCheck && tolerance > 0 {
		nowFn := opts.Now
		if nowFn == nil {
			nowFn = time.Now
		}
		now := nowFn().Unix()
		delta := now - ts
		if delta < 0 {
			delta = -delta
		}
		if time.Duration(delta)*time.Second > tolerance {
			return nil, &VerificationError{
				Reason:  ReasonStaleTimestamp,
				Message: fmt.Sprintf("timestamp %d is outside the %s tolerance window (now=%d)", ts, tolerance, now),
			}
		}
	}

	mac := hmac.New(sha256.New, []byte(opts.Secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(opts.RawBody)
	expected := hex.EncodeToString(mac.Sum(nil))

	// hmac.Equal handles equal-length comparison in constant time, but
	// short-circuits on length mismatch — that's safe here because the
	// length itself is not secret.
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(v1))) {
		return nil, &VerificationError{Reason: ReasonSignatureMismatch, Message: "signature mismatch"}
	}

	if !json.Valid(opts.RawBody) {
		return nil, &VerificationError{Reason: ReasonInvalidBody, Message: "body is not valid JSON"}
	}

	body := make(json.RawMessage, len(opts.RawBody))
	copy(body, opts.RawBody)
	return &Verified{Body: body, Timestamp: ts}, nil
}

// Decode is a generic helper that verifies the signature, then
// unmarshals the body into T.
func Decode[T any](opts Opts) (*T, int64, error) {
	v, err := Verify(opts)
	if err != nil {
		var zero *T
		return zero, 0, err
	}
	out := new(T)
	if err := json.Unmarshal(v.Body, out); err != nil {
		return nil, 0, &VerificationError{Reason: ReasonInvalidBody, Message: fmt.Sprintf("decode body: %v", err)}
	}
	return out, v.Timestamp, nil
}

// parseHeader splits "t=<unix>,v1=<hex>" into (timestamp, v1hex).
// Unknown keys are ignored — forward-compat for a future v2 signature.
func parseHeader(h string) (int64, string, error) {
	var (
		tsRaw string
		v1    string
	)
	parts := strings.Split(h, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		eq := strings.IndexByte(p, '=')
		if eq <= 0 {
			return 0, "", &VerificationError{Reason: ReasonMalformedHeader, Message: "malformed signature header"}
		}
		k := strings.TrimSpace(p[:eq])
		v := strings.TrimSpace(p[eq+1:])
		switch k {
		case "t":
			tsRaw = v
		case "v1":
			v1 = v
		}
	}
	if tsRaw == "" || v1 == "" {
		return 0, "", &VerificationError{Reason: ReasonMalformedHeader, Message: "missing t or v1 field"}
	}
	ts, err := strconv.ParseInt(tsRaw, 10, 64)
	if err != nil {
		return 0, "", &VerificationError{Reason: ReasonMalformedHeader, Message: "t is not an integer"}
	}
	for _, c := range v1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return 0, "", &VerificationError{Reason: ReasonMalformedHeader, Message: "v1 is not hex"}
		}
	}
	return ts, v1, nil
}
