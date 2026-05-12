package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

// signedHeader builds an x-opensettle-signature header for the given
// timestamp/body/secret. Used by every happy-path test.
func signedHeader(ts int64, body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write([]byte(body))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", ts, sig)
}

func fixedNow(ts int64) func() time.Time {
	return func() time.Time { return time.Unix(ts, 0) }
}

func TestVerify_HappyPath(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{"event":"payment.confirmed","data":{"id":"pay_1"}}`
	secret := "whsec_test_abc"
	v, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: signedHeader(ts, body, secret),
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if v.Timestamp != ts {
		t.Fatalf("ts: %d", v.Timestamp)
	}
	if string(v.Body) != body {
		t.Fatalf("body: %s", string(v.Body))
	}
}

func TestVerify_MissingHeader(t *testing.T) {
	_, err := Verify(Opts{RawBody: []byte("{}"), Secret: "x"})
	var ve *VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
	if ve.Reason != ReasonMissingHeader {
		t.Fatalf("reason: %s", ve.Reason)
	}
}

func TestVerify_MalformedHeader_NoComma(t *testing.T) {
	_, err := Verify(Opts{RawBody: []byte("{}"), SignatureHeader: "garbage", Secret: "x"})
	var ve *VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
	if ve.Reason != ReasonMalformedHeader {
		t.Fatalf("reason: %s", ve.Reason)
	}
}

func TestVerify_MalformedHeader_NoT(t *testing.T) {
	_, err := Verify(Opts{RawBody: []byte("{}"), SignatureHeader: "v1=deadbeef", Secret: "x"})
	var ve *VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
	if ve.Reason != ReasonMalformedHeader {
		t.Fatalf("reason: %s", ve.Reason)
	}
}

func TestVerify_MalformedHeader_NoV1(t *testing.T) {
	_, err := Verify(Opts{RawBody: []byte("{}"), SignatureHeader: "t=1700000000", Secret: "x"})
	var ve *VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
	if ve.Reason != ReasonMalformedHeader {
		t.Fatalf("reason: %s", ve.Reason)
	}
}

func TestVerify_MalformedHeader_NonNumericT(t *testing.T) {
	_, err := Verify(Opts{RawBody: []byte("{}"), SignatureHeader: "t=abc,v1=deadbeef", Secret: "x"})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonMalformedHeader {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_MalformedHeader_NonHexV1(t *testing.T) {
	_, err := Verify(Opts{RawBody: []byte("{}"), SignatureHeader: "t=1,v1=ZZZZ", Secret: "x"})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonMalformedHeader {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_MalformedHeader_NoEquals(t *testing.T) {
	_, err := Verify(Opts{RawBody: []byte("{}"), SignatureHeader: "t1700000000", Secret: "x"})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonMalformedHeader {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_StaleTimestamp(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "x"
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: signedHeader(ts, body, secret),
		Secret:          secret,
		Tolerance:       5 * time.Minute,
		Now:             fixedNow(ts + 7*60),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
	if ve.Reason != ReasonStaleTimestamp {
		t.Fatalf("reason: %s", ve.Reason)
	}
}

func TestVerify_FutureTimestampInTolerance(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "x"
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: signedHeader(ts, body, secret),
		Secret:          secret,
		Tolerance:       5 * time.Minute,
		Now:             fixedNow(ts - 60), // ts is "in the future" by 1 min
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_FutureTimestampOutsideTolerance(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "x"
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: signedHeader(ts, body, secret),
		Secret:          secret,
		Tolerance:       5 * time.Minute,
		Now:             fixedNow(ts - 7*60),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonStaleTimestamp {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_ExactlyAtTolerance(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "x"
	// 300s = exactly at the default 5-minute tolerance. Should pass —
	// the check is strict greater-than.
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: signedHeader(ts, body, secret),
		Secret:          secret,
		Tolerance:       5 * time.Minute,
		Now:             fixedNow(ts + 300),
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_ToleranceZeroIsDefault(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "x"
	// Tolerance zero in Opts means "use default" (5 min); a 4-minute
	// gap should still pass.
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: signedHeader(ts, body, secret),
		Secret:          secret,
		Now:             fixedNow(ts + 4*60),
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	header := signedHeader(ts, body, "real_secret")
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          "wrong_secret",
		Now:             fixedNow(ts),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
	if ve.Reason != ReasonSignatureMismatch {
		t.Fatalf("reason: %s", ve.Reason)
	}
}

func TestVerify_TamperedBody(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	header := signedHeader(ts, body, "secret")
	_, err := Verify(Opts{
		RawBody:         []byte(`{"tampered":true}`),
		SignatureHeader: header,
		Secret:          "secret",
		Now:             fixedNow(ts),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonSignatureMismatch {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_TamperedTimestamp(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	header := signedHeader(ts, body, "secret")
	// Swap the timestamp but keep the signature — body+sig anchored on
	// original ts so the recompute won't match.
	tampered := strings.Replace(header, fmt.Sprintf("t=%d", ts), fmt.Sprintf("t=%d", ts+1), 1)
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: tampered,
		Secret:          "secret",
		Now:             fixedNow(ts),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonSignatureMismatch {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_UnknownKeysIgnored(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "secret"
	base := signedHeader(ts, body, secret)
	// Insert an unknown v2 key to simulate forward-compat.
	withV2 := base + ",v2=somefuturealgo"
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: withV2,
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	if err != nil {
		t.Fatalf("unknown keys should be ignored: %v", err)
	}
}

func TestVerify_KVOrderIndependent(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + "." + body))
	sig := hex.EncodeToString(mac.Sum(nil))
	reordered := fmt.Sprintf("v1=%s,t=%d", sig, ts)
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: reordered,
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_HeaderWithSpaces(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "secret"
	base := signedHeader(ts, body, secret)
	withSpaces := strings.Replace(base, ",", " , ", 1)
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: withSpaces,
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_InvalidJSONBody(t *testing.T) {
	ts := int64(1_700_000_000)
	body := "not json"
	secret := "secret"
	header := signedHeader(ts, body, secret)
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
	if ve.Reason != ReasonInvalidBody {
		t.Fatalf("reason: %s", ve.Reason)
	}
}

func TestVerify_UppercaseHexAccepted(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{}`
	secret := "secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + "." + body))
	sig := strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	header := fmt.Sprintf("t=%d,v1=%s", ts, sig)
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	if err != nil {
		t.Fatalf("uppercase hex should verify: %v", err)
	}
}

func TestVerify_EmptyBody(t *testing.T) {
	ts := int64(1_700_000_000)
	body := ""
	secret := "secret"
	header := signedHeader(ts, body, secret)
	// Empty body is not valid JSON so we expect InvalidBody.
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonInvalidBody {
		t.Fatalf("got %v", err)
	}
}

func TestVerify_DefaultNowUsesTimeNow(t *testing.T) {
	ts := time.Now().Unix()
	body := `{}`
	secret := "secret"
	header := signedHeader(ts, body, secret)
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          secret,
		// Now omitted on purpose — should fall back to time.Now.
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
}

type sampleEvent struct {
	Event string `json:"event"`
	Data  struct {
		ID string `json:"id"`
	} `json:"data"`
}

func TestDecode_HappyPath(t *testing.T) {
	ts := int64(1_700_000_000)
	body := `{"event":"payment.confirmed","data":{"id":"pay_1"}}`
	secret := "secret"
	header := signedHeader(ts, body, secret)
	ev, gotTs, err := Decode[sampleEvent](Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if ev.Event != "payment.confirmed" || ev.Data.ID != "pay_1" {
		t.Fatalf("got %+v", ev)
	}
	if gotTs != ts {
		t.Fatalf("ts: %d", gotTs)
	}
}

func TestDecode_VerificationFailurePropagates(t *testing.T) {
	_, _, err := Decode[sampleEvent](Opts{
		RawBody:         []byte("{}"),
		SignatureHeader: "garbage",
		Secret:          "x",
	})
	var ve *VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
}

func TestVerificationError_String(t *testing.T) {
	e := &VerificationError{Reason: ReasonSignatureMismatch, Message: "x"}
	if !strings.Contains(e.Error(), "signature_mismatch") {
		t.Fatalf("got %q", e.Error())
	}
}

func TestVerify_ConstantTimeOnLengthMismatch(t *testing.T) {
	// Signature with wrong length — should still produce SignatureMismatch,
	// not panic.
	ts := int64(1_700_000_000)
	body := `{}`
	header := fmt.Sprintf("t=%d,v1=deadbeef", ts)
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          "secret",
		Now:             fixedNow(ts),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonSignatureMismatch {
		t.Fatalf("got %v", err)
	}
}

func TestReason_Values(t *testing.T) {
	cases := map[Reason]string{
		ReasonMissingHeader:     "missing_header",
		ReasonMalformedHeader:   "malformed_header",
		ReasonStaleTimestamp:    "stale_timestamp",
		ReasonSignatureMismatch: "signature_mismatch",
		ReasonInvalidBody:       "invalid_body",
	}
	for r, want := range cases {
		if string(r) != want {
			t.Errorf("got %q want %q", r, want)
		}
	}
}

func TestVerify_RawBodyBytePreserved(t *testing.T) {
	// Body contains escaped characters; signature must anchor on bytes
	// exactly as sent.
	ts := int64(1_700_000_000)
	body := `{"x":"a\tbA"}`
	secret := "secret"
	header := signedHeader(ts, body, secret)
	v, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          secret,
		Now:             fixedNow(ts),
	})
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if string(v.Body) != body {
		t.Fatalf("body lost bytes: %q vs %q", string(v.Body), body)
	}
}

func TestVerify_ZeroTimestampDisabledByExplicitTolerance(t *testing.T) {
	// To actually disable timestamp checks, the API doesn't expose a
	// "disable" knob — tolerance of 0 falls back to default. We document
	// the behavior here so it can't regress silently.
	ts := int64(1)
	body := `{}`
	secret := "secret"
	header := signedHeader(ts, body, secret)
	_, err := Verify(Opts{
		RawBody:         []byte(body),
		SignatureHeader: header,
		Secret:          secret,
		Tolerance:       0,
		Now:             fixedNow(ts + 999999),
	})
	var ve *VerificationError
	if !errors.As(err, &ve) || ve.Reason != ReasonStaleTimestamp {
		t.Fatalf("got %v; expected stale ts with default tolerance applied", err)
	}
}
