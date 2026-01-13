package main

import (
	"bytes"
	"os"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
)

func TestDefaultEnv(t *testing.T) {
	t.Parallel()

	env := DefaultEnv()

	t.Run("Now returns real time", func(t *testing.T) {
		before := time.Now()
		got := env.Now()
		after := time.Now()

		if got.Before(before) || got.After(after) {
			t.Errorf("Now() = %v, should be between %v and %v", got, before, after)
		}
	})

	t.Run("Stdout is os.Stdout", func(t *testing.T) {
		if env.Stdout != os.Stdout {
			t.Error("Stdout should be os.Stdout")
		}
	})

	t.Run("Stderr is os.Stderr", func(t *testing.T) {
		if env.Stderr != os.Stderr {
			t.Error("Stderr should be os.Stderr")
		}
	})

	t.Run("AssetLoader is not nil", func(t *testing.T) {
		if env.AssetLoader == nil {
			t.Error("AssetLoader should not be nil")
		}
	})
}

func TestEnvironmentInjection(t *testing.T) {
	t.Parallel()

	loader, _ := md2pdf.NewAssetLoader("")

	t.Run("mock time is used", func(t *testing.T) {
		t.Parallel()

		fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
		env := &Environment{
			Now:         func() time.Time { return fixedTime },
			Stdout:      &bytes.Buffer{},
			Stderr:      &bytes.Buffer{},
			AssetLoader: loader,
		}

		got := env.Now()
		if !got.Equal(fixedTime) {
			t.Errorf("Now() = %v, want %v", got, fixedTime)
		}
	})

	t.Run("mock stdout captures output", func(t *testing.T) {
		t.Parallel()

		var stdout bytes.Buffer
		env := &Environment{
			Now:         time.Now,
			Stdout:      &stdout,
			Stderr:      &bytes.Buffer{},
			AssetLoader: loader,
		}

		// Simulate writing to stdout
		env.Stdout.Write([]byte("test output"))

		if stdout.String() != "test output" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "test output")
		}
	})

	t.Run("mock stderr captures errors", func(t *testing.T) {
		t.Parallel()

		var stderr bytes.Buffer
		env := &Environment{
			Now:         time.Now,
			Stdout:      &bytes.Buffer{},
			Stderr:      &stderr,
			AssetLoader: loader,
		}

		// Simulate writing to stderr
		env.Stderr.Write([]byte("error output"))

		if stderr.String() != "error output" {
			t.Errorf("stderr = %q, want %q", stderr.String(), "error output")
		}
	})
}
