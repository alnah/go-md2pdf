package main

// Notes:
// - notifyContext: we only test the observable behavior (context creation,
//   cancellation via stop(), and parent context propagation). We do not test
//   actual OS signal delivery since it's non-deterministic and requires
//   platform-specific setup.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// TestNotifyContext - Context creation and cancellation behavior
// ---------------------------------------------------------------------------

func TestNotifyContext(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil context", func(t *testing.T) {
		t.Parallel()

		ctx, stop := notifyContext(context.Background())
		defer stop()

		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
	})

	t.Run("context starts not cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, stop := notifyContext(context.Background())
		defer stop()

		select {
		case <-ctx.Done():
			t.Fatal("context should not be cancelled initially")
		default:
			// Expected: context is not cancelled
		}
	})

	t.Run("stop function cancels context", func(t *testing.T) {
		t.Parallel()

		ctx, stop := notifyContext(context.Background())
		stop()

		select {
		case <-ctx.Done():
			// Expected: context is cancelled after stop()
		default:
			t.Fatal("context should be cancelled after stop()")
		}
	})

	t.Run("inherits parent cancellation", func(t *testing.T) {
		t.Parallel()

		parent, cancel := context.WithCancel(context.Background())
		ctx, stop := notifyContext(parent)
		defer stop()

		cancel() // Cancel parent

		select {
		case <-ctx.Done():
			// Expected: child context is cancelled when parent is
		default:
			t.Fatal("context should be cancelled when parent is cancelled")
		}
	})
}
