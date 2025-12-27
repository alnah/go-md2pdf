package main

import (
	"errors"
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
