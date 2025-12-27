package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

type MockRunner struct {
	Stdout     string
	Stderr     string
	Err        error
	CalledWith []string
}

func (m *MockRunner) Run(name string, args ...string) (string, string, error) {
	m.CalledWith = append([]string{name}, args...)
	return m.Stdout, m.Stderr, m.Err
}

func TestPandocConverter_ToHTML(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		mock       *MockRunner
		wantErr    error
		wantAnyErr bool
		wantOutput string
	}{
		{
			name:    "empty content returns ErrEmptyContent",
			content: "",
			mock:    &MockRunner{},
			wantErr: ErrEmptyContent,
		},
		{
			name:    "pandoc succeeds returns HTML",
			content: "# Test",
			mock: &MockRunner{
				Stdout: "<html><body><h1>Test</h1></body></html>",
			},
			wantOutput: "<html><body><h1>Test</h1></body></html>",
		},
		{
			name:    "pandoc fails returns error with stderr",
			content: "# Test",
			mock: &MockRunner{
				Stderr: "pandoc: unknown option --bad",
				Err:    errors.New("exit status 1"),
			},
			wantAnyErr: true,
		},
		{
			name:    "whitespace-only content is valid",
			content: "   \n\t\n   ",
			mock: &MockRunner{
				Stdout: "<html><body></body></html>",
			},
			wantOutput: "<html><body></body></html>",
		},
		{
			name:    "unicode content succeeds",
			content: "# Bonjour le monde",
			mock: &MockRunner{
				Stdout: "<html><body><h1>Bonjour le monde</h1></body></html>",
			},
			wantOutput: "<html><body><h1>Bonjour le monde</h1></body></html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &PandocConverter{Runner: tt.mock}
			got, err := converter.ToHTML(tt.content)

			if tt.wantAnyErr || tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantOutput != "" && got != tt.wantOutput {
				t.Errorf("expected output %q, got %q", tt.wantOutput, got)
			}

			// Verify pandoc was called with a temp file
			if len(tt.mock.CalledWith) < 2 {
				t.Fatal("expected pandoc to be called with arguments")
			}
			if tt.mock.CalledWith[0] != "pandoc" {
				t.Errorf("expected command 'pandoc', got %q", tt.mock.CalledWith[0])
			}
			// Temp file path should contain our prefix
			if !strings.Contains(tt.mock.CalledWith[1], "go-md2pdf-") {
				t.Errorf("expected temp file path with 'go-md2pdf-', got %q", tt.mock.CalledWith[1])
			}
		})
	}
}

func TestWriteTempMarkdown(t *testing.T) {
	content := "# Test Markdown"

	path, cleanup, err := writeTempMarkdown(content)
	if err != nil {
		t.Fatalf("writeTempMarkdown() error = %v", err)
	}

	t.Run("file exists", func(t *testing.T) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("temp file does not exist at %s", path)
		}
	})

	t.Run("file has .md extension pattern", func(t *testing.T) {
		if !strings.Contains(path, "go-md2pdf-") || !strings.HasSuffix(path, ".md") {
			t.Errorf("path %q does not match expected pattern go-md2pdf-*.md", path)
		}
	})

	t.Run("file contains expected content", func(t *testing.T) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}
		if string(data) != content {
			t.Errorf("file content = %q, want %q", string(data), content)
		}
	})

	t.Run("cleanup removes file", func(t *testing.T) {
		cleanup()
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("temp file still exists after cleanup at %s", path)
		}
	})
}

func TestWriteTempMarkdown_Unicode(t *testing.T) {
	content := "# Bonjour le monde\n\nCeci est un test avec des caracteres speciaux: e, a, u"

	path, cleanup, err := writeTempMarkdown(content)
	if err != nil {
		t.Fatalf("writeTempMarkdown() error = %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}
	if string(data) != content {
		t.Errorf("unicode content not preserved: got %q, want %q", string(data), content)
	}
}

func TestWriteTempMarkdown_EmptyString(t *testing.T) {
	path, cleanup, err := writeTempMarkdown("")
	if err != nil {
		t.Fatalf("writeTempMarkdown() error = %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}
	if string(data) != "" {
		t.Errorf("expected empty file, got %q", string(data))
	}
}
