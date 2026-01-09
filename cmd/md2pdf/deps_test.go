package main

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestDefaultDeps(t *testing.T) {
	t.Parallel()

	deps := DefaultDeps()

	t.Run("Now returns real time", func(t *testing.T) {
		before := time.Now()
		got := deps.Now()
		after := time.Now()

		if got.Before(before) || got.After(after) {
			t.Errorf("Now() = %v, should be between %v and %v", got, before, after)
		}
	})

	t.Run("Stdout is os.Stdout", func(t *testing.T) {
		if deps.Stdout != os.Stdout {
			t.Error("Stdout should be os.Stdout")
		}
	})

	t.Run("Stderr is os.Stderr", func(t *testing.T) {
		if deps.Stderr != os.Stderr {
			t.Error("Stderr should be os.Stderr")
		}
	})
}

func TestDependencyInjection(t *testing.T) {
	t.Parallel()

	t.Run("mock time is used", func(t *testing.T) {
		t.Parallel()

		fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
		deps := &Dependencies{
			Now:    func() time.Time { return fixedTime },
			Stdout: &bytes.Buffer{},
			Stderr: &bytes.Buffer{},
		}

		got := deps.Now()
		if !got.Equal(fixedTime) {
			t.Errorf("Now() = %v, want %v", got, fixedTime)
		}
	})

	t.Run("mock stdout captures output", func(t *testing.T) {
		t.Parallel()

		var stdout bytes.Buffer
		deps := &Dependencies{
			Now:    time.Now,
			Stdout: &stdout,
			Stderr: &bytes.Buffer{},
		}

		// Simulate writing to stdout
		deps.Stdout.Write([]byte("test output"))

		if stdout.String() != "test output" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "test output")
		}
	})

	t.Run("mock stderr captures errors", func(t *testing.T) {
		t.Parallel()

		var stderr bytes.Buffer
		deps := &Dependencies{
			Now:    time.Now,
			Stdout: &bytes.Buffer{},
			Stderr: &stderr,
		}

		// Simulate writing to stderr
		deps.Stderr.Write([]byte("error output"))

		if stderr.String() != "error output" {
			t.Errorf("stderr = %q, want %q", stderr.String(), "error output")
		}
	})
}
