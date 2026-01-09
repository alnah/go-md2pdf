package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
)

// wrongTypeConverter is a Converter that is NOT *md2pdf.Service.
type wrongTypeConverter struct{}

func (w *wrongTypeConverter) Convert(_ context.Context, _ md2pdf.Input) ([]byte, error) {
	return []byte("%PDF-1.4 mock"), nil
}

func TestPoolAdapter_Release_WrongType(t *testing.T) {
	// Create a real pool with size 1
	pool := md2pdf.NewServicePool(1)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	// Capture stderr to verify error message
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Release with wrong type - should log error, not panic
	wrongType := &wrongTypeConverter{}
	adapter.Release(wrongType) // Should not panic

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify error was logged
	expectedMsg := "poolAdapter.Release: unexpected type"
	if output == "" || !bytes.Contains([]byte(output), []byte(expectedMsg)) {
		t.Errorf("expected error message containing %q, got %q", expectedMsg, output)
	}
}

func TestPoolAdapter_Size(t *testing.T) {
	pool := md2pdf.NewServicePool(3)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	if adapter.Size() != 3 {
		t.Errorf("Size() = %d, want 3", adapter.Size())
	}
}

func TestPoolAdapter_AcquireRelease(t *testing.T) {
	pool := md2pdf.NewServicePool(1)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	// Acquire should return a non-nil Converter
	svc := adapter.Acquire()
	if svc == nil {
		t.Fatal("Acquire() returned nil")
	}

	// Release should not panic
	adapter.Release(svc)
}

func TestVersion(t *testing.T) {
	// Version variable should be set (default is "dev")
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Capture output to verify version format
	expected := fmt.Sprintf("go-md2pdf %s\n", Version)
	_ = expected // Used in actual main() but we can't easily test that
}

func TestIsCommand(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"convert", true},
		{"version", true},
		{"help", true},
		{"foo", false},
		{"", false},
		{"doc.md", false},
		{"Convert", false}, // case sensitive
		{"VERSION", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isCommand(tt.input)
			if got != tt.want {
				t.Errorf("isCommand(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLooksLikeMarkdown(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"doc.md", true},
		{"doc.markdown", true},
		{"/path/to/doc.md", true},
		{"/path/to/doc.markdown", true},
		{"doc.txt", false},
		{"doc", false},
		{"", false},
		{"md.txt", false},
		{"markdown.pdf", false},
		{".md", true},
		{"file.MD", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeMarkdown(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRunMain(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantCode     int
		wantInStdout []string
		wantInStderr []string
	}{
		{
			name:         "no args shows usage and exits 1",
			args:         []string{"md2pdf"},
			wantCode:     1,
			wantInStderr: []string{"Usage: md2pdf"},
		},
		{
			name:         "version command exits 0",
			args:         []string{"md2pdf", "version"},
			wantCode:     0,
			wantInStdout: []string{"md2pdf"},
		},
		{
			name:         "help command exits 0",
			args:         []string{"md2pdf", "help"},
			wantCode:     0,
			wantInStdout: []string{"Usage: md2pdf", "Commands:"},
		},
		{
			name:         "help convert shows convert help",
			args:         []string{"md2pdf", "help", "convert"},
			wantCode:     0,
			wantInStdout: []string{"Usage: md2pdf convert"},
		},
		{
			name:         "unknown command exits 1",
			args:         []string{"md2pdf", "unknown"},
			wantCode:     1,
			wantInStderr: []string{"unknown command: unknown"},
		},
		{
			name:         "legacy .md detection shows deprecation warning",
			args:         []string{"md2pdf", "nonexistent.md"},
			wantCode:     1, // Will fail because file doesn't exist
			wantInStderr: []string{"DEPRECATED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			deps := &Dependencies{
				Now:    func() time.Time { return time.Now() },
				Stdout: &stdout,
				Stderr: &stderr,
			}

			code := runMain(tt.args, deps)

			if code != tt.wantCode {
				t.Errorf("runMain() = %d, want %d", code, tt.wantCode)
			}

			stdoutStr := stdout.String()
			stderrStr := stderr.String()

			for _, want := range tt.wantInStdout {
				if !bytes.Contains([]byte(stdoutStr), []byte(want)) {
					t.Errorf("stdout should contain %q, got %q", want, stdoutStr)
				}
			}

			for _, want := range tt.wantInStderr {
				if !bytes.Contains([]byte(stderrStr), []byte(want)) {
					t.Errorf("stderr should contain %q, got %q", want, stderrStr)
				}
			}
		})
	}
}
