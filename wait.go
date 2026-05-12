package opensettle

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Wait helpers — block until a resource reaches a target state by
// polling its Retrieve method. Useful in scripts and CI; webhooks are
// the right tool for production.

// WaitOptions tunes polling behaviour.
type WaitOptions struct {
	// Hard timeout. Defaults to 2 minutes when zero.
	Timeout time.Duration
	// Poll interval. Defaults to 2 seconds when zero.
	Interval time.Duration
	// Sleep, exposed for tests. Defaults to time.Sleep wrapped in a
	// context-aware sleep so cancellation works.
	sleep func(ctx context.Context, d time.Duration) error
	// Now, exposed for tests. Defaults to time.Now.
	now func() time.Time
}

const (
	defaultWaitTimeout  = 2 * time.Minute
	defaultWaitInterval = 2 * time.Second
)

// WaitTimeoutError is returned when the target state is not reached
// before the timeout elapses. It carries the last-observed resource as
// an opaque any value — type-assert to the expected pointer type to
// inspect it.
type WaitTimeoutError struct {
	ResourceID string
	Timeout    time.Duration
	// Last is the most-recently-observed resource (typed as the
	// concrete *T that the Retrieve closure returns).
	Last any
}

func (e *WaitTimeoutError) Error() string {
	return fmt.Sprintf(
		"opensettle: %s did not reach target state within %s",
		e.ResourceID, e.Timeout,
	)
}

func contextSleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitFor polls retrieve(ctx, id) every opts.Interval until until(r)
// returns true, then returns the resource. Returns *WaitTimeoutError
// when opts.Timeout elapses first, or the wrapped ctx error on
// cancellation. Any error returned by retrieve aborts the loop
// immediately.
//
// Type parameter T is the concrete resource pointed at by Retrieve
// (e.g. *Payment, *Invoice).
func WaitFor[T any](
	ctx context.Context,
	retrieve func(ctx context.Context, id string) (T, error),
	resourceID string,
	until func(T) bool,
	opts WaitOptions,
) (T, error) {
	var zero T
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultWaitTimeout
	}
	interval := opts.Interval
	if interval == 0 {
		interval = defaultWaitInterval
	}
	sleep := opts.sleep
	if sleep == nil {
		sleep = contextSleep
	}
	nowFn := opts.now
	if nowFn == nil {
		nowFn = time.Now
	}

	deadline := nowFn().Add(timeout)
	var last T
	for {
		resource, err := retrieve(ctx, resourceID)
		if err != nil {
			return zero, err
		}
		last = resource
		if until(resource) {
			return resource, nil
		}
		if !nowFn().Before(deadline) {
			return zero, &WaitTimeoutError{
				ResourceID: resourceID,
				Timeout:    timeout,
				Last:       last,
			}
		}
		if err := sleep(ctx, interval); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return zero, err
			}
			return zero, err
		}
	}
}
