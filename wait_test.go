package opensettle

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWaitFor_ReturnsImmediatelyWhenSatisfied(t *testing.T) {
	retrieve := func(ctx context.Context, id string) (*Payment, error) {
		return &Payment{ID: id, Status: PaymentConfirmed}, nil
	}
	slept := false
	got, err := WaitFor(
		bgCtx(),
		retrieve,
		"pay_1",
		func(p *Payment) bool { return p.Status == PaymentConfirmed },
		WaitOptions{
			sleep: func(ctx context.Context, d time.Duration) error {
				slept = true
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Status != PaymentConfirmed {
		t.Fatalf("status: %s", got.Status)
	}
	if slept {
		t.Fatalf("should not have slept")
	}
}

func TestWaitFor_PollsUntilSatisfied(t *testing.T) {
	states := []PaymentStatus{PaymentPending, PaymentPending, PaymentConfirmed}
	idx := 0
	retrieve := func(ctx context.Context, id string) (*Payment, error) {
		s := states[idx]
		idx++
		return &Payment{ID: id, Status: s}, nil
	}
	sleeps := []time.Duration{}
	got, err := WaitFor(
		bgCtx(),
		retrieve,
		"pay_1",
		func(p *Payment) bool { return p.Status == PaymentConfirmed },
		WaitOptions{
			Interval: 250 * time.Millisecond,
			sleep: func(ctx context.Context, d time.Duration) error {
				sleeps = append(sleeps, d)
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Status != PaymentConfirmed {
		t.Fatalf("status: %s", got.Status)
	}
	if len(sleeps) != 2 {
		t.Fatalf("sleeps: %v", sleeps)
	}
}

func TestWaitFor_TimesOut(t *testing.T) {
	retrieve := func(ctx context.Context, id string) (*Payment, error) {
		return &Payment{ID: id, Status: PaymentPending}, nil
	}
	clock := []time.Time{
		time.Unix(0, 0),
		time.Unix(1, 0),
		time.Unix(3, 0),
	}
	idx := 0
	_, err := WaitFor(
		bgCtx(),
		retrieve,
		"pay_x",
		func(p *Payment) bool { return p.Status == PaymentConfirmed },
		WaitOptions{
			Timeout:  2 * time.Second,
			Interval: 1 * time.Second,
			sleep:    func(ctx context.Context, d time.Duration) error { return nil },
			now: func() time.Time {
				t := clock[idx]
				if idx < len(clock)-1 {
					idx++
				}
				return t
			},
		},
	)
	var te *WaitTimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("expected *WaitTimeoutError, got %T: %v", err, err)
	}
	last, ok := te.Last.(*Payment)
	if !ok {
		t.Fatalf("Last not *Payment: %T", te.Last)
	}
	if last.Status != PaymentPending {
		t.Fatalf("last status: %s", last.Status)
	}
}

func TestWaitFor_PropagatesRetrieveError(t *testing.T) {
	wantErr := errors.New("boom")
	retrieve := func(ctx context.Context, id string) (*Payment, error) {
		return nil, wantErr
	}
	_, err := WaitFor(
		bgCtx(),
		retrieve,
		"pay_1",
		func(p *Payment) bool { return true },
		WaitOptions{},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped err, got %v", err)
	}
}

func TestWaitFor_RespectsContextCancellation(t *testing.T) {
	retrieve := func(ctx context.Context, id string) (*Payment, error) {
		return &Payment{ID: id, Status: PaymentPending}, nil
	}
	_, err := WaitFor(
		bgCtx(),
		retrieve,
		"pay_x",
		func(p *Payment) bool { return p.Status == PaymentConfirmed },
		WaitOptions{
			Timeout:  10 * time.Second,
			Interval: 1 * time.Second,
			sleep: func(ctx context.Context, d time.Duration) error {
				return context.Canceled
			},
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestWaitTimeoutError_Error(t *testing.T) {
	e := &WaitTimeoutError{ResourceID: "pay_42", Timeout: 90 * time.Second}
	msg := e.Error()
	if !strings.Contains(msg, "pay_42") {
		t.Fatalf("error missing resource id: %q", msg)
	}
	if !strings.Contains(msg, "1m30s") && !strings.Contains(msg, "90s") {
		t.Fatalf("error missing timeout: %q", msg)
	}
	if !strings.Contains(msg, "opensettle:") {
		t.Fatalf("error missing package prefix: %q", msg)
	}
}

func TestContextSleep_FiresTimer(t *testing.T) {
	start := time.Now()
	err := contextSleep(bgCtx(), 30*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 25*time.Millisecond {
		t.Fatalf("returned too early: %v", elapsed)
	}
}

func TestContextSleep_RespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(bgCtx())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	err := contextSleep(ctx, 10*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestContextSleep_RespectsAlreadyCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(bgCtx())
	cancel()
	err := contextSleep(ctx, 1*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
