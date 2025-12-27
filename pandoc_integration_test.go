//go:build integration

package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestPandocConverter_ToHTML_Integration(t *testing.T) {
	content := `# Hello

World`

	converter := NewPandocConverter()
	got, err := converter.ToHTML(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "<h1") {
		t.Errorf("expected output to contain <h1>, got %q", got)
	}

	if !strings.Contains(got, "Hello") {
		t.Errorf("expected output to contain 'Hello', got %q", got)
	}

	if !strings.Contains(got, "<p>World</p>") && !strings.Contains(got, "<p>World") {
		t.Errorf("expected output to contain paragraph with 'World', got %q", got)
	}
}

func TestExecRunner_Run(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows: tests use unix commands")
	}

	runner := &ExecRunner{}

	t.Run("captures stdout", func(t *testing.T) {
		stdout, stderr, err := runner.Run("echo", "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.TrimSpace(stdout) != "hello" {
			t.Errorf("expected stdout %q, got %q", "hello", stdout)
		}
		if stderr != "" {
			t.Errorf("expected empty stderr, got %q", stderr)
		}
	})

	t.Run("captures stderr", func(t *testing.T) {
		stdout, stderr, err := runner.Run("sh", "-c", "echo error >&2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdout != "" {
			t.Errorf("expected empty stdout, got %q", stdout)
		}
		if strings.TrimSpace(stderr) != "error" {
			t.Errorf("expected stderr %q, got %q", "error", stderr)
		}
	})

	t.Run("returns error on command failure", func(t *testing.T) {
		_, stderr, err := runner.Run("sh", "-c", "echo fail >&2; exit 1")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if strings.TrimSpace(stderr) != "fail" {
			t.Errorf("expected stderr %q, got %q", "fail", stderr)
		}
	})

	t.Run("returns error on non-existent command", func(t *testing.T) {
		_, _, err := runner.Run("command-that-does-not-exist-12345")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
