//go:build windows

package main

import (
	"context"
	"os"
	"os/signal"
)

// notifyContext returns a context that is canceled when an interrupt
// signal is received. Call stop() to release resources.
// Note: syscall.SIGTERM is not available on Windows.
func notifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt)
}
