//go:build integration

package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestPandocConverter_ToHTML_Integration(t *testing.T) {
	t.Run("basic markdown", func(t *testing.T) {
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
	})

	t.Run("unicode content", func(t *testing.T) {
		content := `# Bonjour le monde

Ceci est un test avec des caracteres speciaux.`

		converter := NewPandocConverter()
		got, err := converter.ToHTML(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "Bonjour") {
			t.Errorf("expected output to contain unicode text, got %q", got)
		}
	})

	t.Run("code block with special chars", func(t *testing.T) {
		content := "# Code Example\n\n```go\nfunc main() {\n\tfmt.Println(\"<hello>\")\n}\n```"

		converter := NewPandocConverter()
		got, err := converter.ToHTML(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "<code") {
			t.Errorf("expected output to contain <code>, got %q", got)
		}
	})

	t.Run("table markdown", func(t *testing.T) {
		content := `# Table Test

| Name | Age |
|------|-----|
| Alice | 30 |
| Bob | 25 |`

		converter := NewPandocConverter()
		got, err := converter.ToHTML(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "<table") {
			t.Errorf("expected output to contain <table>, got %q", got)
		}
		if !strings.Contains(got, "Alice") {
			t.Errorf("expected output to contain table data, got %q", got)
		}
	})

	t.Run("nested list", func(t *testing.T) {
		content := `# List Test

- Item 1
  - Subitem 1.1
  - Subitem 1.2
- Item 2`

		converter := NewPandocConverter()
		got, err := converter.ToHTML(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "<ul") {
			t.Errorf("expected output to contain <ul>, got %q", got)
		}
		if !strings.Contains(got, "<li") {
			t.Errorf("expected output to contain <li>, got %q", got)
		}
	})

	t.Run("whitespace-only content is valid", func(t *testing.T) {
		content := "   \n\t\n   "

		converter := NewPandocConverter()
		_, err := converter.ToHTML(content)
		if err != nil {
			t.Fatalf("whitespace-only content should be valid, got error: %v", err)
		}
	})
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

	t.Run("captures both stdout and stderr simultaneously", func(t *testing.T) {
		stdout, stderr, err := runner.Run("sh", "-c", "echo out; echo err >&2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.TrimSpace(stdout) != "out" {
			t.Errorf("expected stdout %q, got %q", "out", stdout)
		}
		if strings.TrimSpace(stderr) != "err" {
			t.Errorf("expected stderr %q, got %q", "err", stderr)
		}
	})
}
