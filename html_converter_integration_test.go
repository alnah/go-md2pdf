//go:build integration

package main

import (
	"strings"
	"testing"
)

func TestGoldmarkConverter_ToHTML_Integration(t *testing.T) {
	t.Run("basic markdown", func(t *testing.T) {
		content := `# Hello

World`
		converter := NewGoldmarkConverter()
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

		converter := NewGoldmarkConverter()
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

		converter := NewGoldmarkConverter()
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

		converter := NewGoldmarkConverter()
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

		converter := NewGoldmarkConverter()
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

		converter := NewGoldmarkConverter()
		_, err := converter.ToHTML(content)
		if err != nil {
			t.Fatalf("whitespace-only content should be valid, got error: %v", err)
		}
	})
}
