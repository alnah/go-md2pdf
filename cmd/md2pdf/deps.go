package main

import (
	"io"
	"os"
	"time"
)

// Dependencies holds injectable dependencies for testability.
type Dependencies struct {
	Now    func() time.Time
	Stdout io.Writer
	Stderr io.Writer
}

// DefaultDeps returns production dependencies.
func DefaultDeps() *Dependencies {
	return &Dependencies{
		Now:    time.Now,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}
