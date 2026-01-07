//go:build !windows

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// notifyContext returns a context that is canceled when an interrupt
// or termination signal is received. Call stop() to release resources.
func notifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}
