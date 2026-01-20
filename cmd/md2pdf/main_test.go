package main

// Notes:
// - poolAdapter: we test Acquire/Release/Size and panic on wrong type.
// - isCommand: we test command name matching.
// - looksLikeMarkdown: we test file extension detection.
// - runMain: we test exit codes for various scenarios. We don't test actual
//   file conversion here (covered by integration tests).
// - resolveTimeoutWithEnv: we test duration parsing, validation, and priority.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
)

// ---------------------------------------------------------------------------
// Test Infrastructure - Mock converter
// ---------------------------------------------------------------------------

// wrongTypeConverter is a Converter that is NOT *md2pdf.Service.
type wrongTypeConverter struct{}

func (w *wrongTypeConverter) Convert(_ context.Context, _ md2pdf.Input) (*md2pdf.ConvertResult, error) {
	return &md2pdf.ConvertResult{PDF: []byte("%PDF-1.4 mock")}, nil
}

// ---------------------------------------------------------------------------
// TestPoolAdapter_Release_WrongType - Pool adapter type safety
// ---------------------------------------------------------------------------

func TestPoolAdapter_Release_WrongType(t *testing.T) {
	t.Parallel()

	// Create a real pool with size 1
	pool := md2pdf.NewConverterPool(1)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	// Release with wrong type should panic (programmer error)
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for wrong type, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if !strings.Contains(msg, "unexpected type") {
			t.Errorf("panic message should contain 'unexpected type', got %q", msg)
		}
	}()

	wrongType := &wrongTypeConverter{}
	adapter.Release(wrongType)
}

// ---------------------------------------------------------------------------
// TestPoolAdapter_Size - Pool size reporting
// ---------------------------------------------------------------------------

func TestPoolAdapter_Size(t *testing.T) {
	t.Parallel()

	pool := md2pdf.NewConverterPool(3)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	if adapter.Size() != 3 {
		t.Errorf("Size() = %d, want 3", adapter.Size())
	}
}

// ---------------------------------------------------------------------------
// TestPoolAdapter_AcquireRelease - Pool acquire and release
// ---------------------------------------------------------------------------

func TestPoolAdapter_AcquireRelease(t *testing.T) {
	t.Parallel()

	pool := md2pdf.NewConverterPool(1)
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

// ---------------------------------------------------------------------------
// TestVersion - Version variable
// ---------------------------------------------------------------------------

func TestVersion(t *testing.T) {
	t.Parallel()

	// Version variable should be set (default is "dev")
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Capture output to verify version format
	expected := fmt.Sprintf("go-md2pdf %s\n", Version)
	_ = expected // Used in actual main() but we can't easily test that
}

// ---------------------------------------------------------------------------
// TestIsCommand - Command name detection
// ---------------------------------------------------------------------------

func TestIsCommand(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := isCommand(tt.input)
			if got != tt.want {
				t.Errorf("isCommand(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveTimeoutWithEnv - Timeout duration resolution with env var support
// ---------------------------------------------------------------------------

func TestResolveTimeoutWithEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		flagValue   string
		envValue    time.Duration
		configValue string
		want        time.Duration
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "all empty uses default",
			flagValue:   "",
			envValue:    0,
			configValue: "",
			want:        0,
			wantErr:     false,
		},
		{
			name:        "flag only",
			flagValue:   "2m",
			envValue:    0,
			configValue: "",
			want:        2 * time.Minute,
			wantErr:     false,
		},
		{
			name:        "env only",
			flagValue:   "",
			envValue:    45 * time.Second,
			configValue: "",
			want:        45 * time.Second,
			wantErr:     false,
		},
		{
			name:        "config only",
			flagValue:   "",
			envValue:    0,
			configValue: "30s",
			want:        30 * time.Second,
			wantErr:     false,
		},
		{
			name:        "flag overrides env and config",
			flagValue:   "5m",
			envValue:    45 * time.Second,
			configValue: "30s",
			want:        5 * time.Minute,
			wantErr:     false,
		},
		{
			name:        "env overrides config",
			flagValue:   "",
			envValue:    2 * time.Minute,
			configValue: "30s",
			want:        2 * time.Minute,
			wantErr:     false,
		},
		{
			name:        "combined duration",
			flagValue:   "1m30s",
			envValue:    0,
			configValue: "",
			want:        90 * time.Second,
			wantErr:     false,
		},
		{
			name:        "invalid flag format",
			flagValue:   "abc",
			envValue:    0,
			configValue: "",
			wantErr:     true,
			errSubstr:   "invalid timeout",
		},
		{
			name:        "invalid config format",
			flagValue:   "",
			envValue:    0,
			configValue: "xyz",
			wantErr:     true,
			errSubstr:   "invalid timeout",
		},
		{
			name:        "negative duration",
			flagValue:   "-5s",
			envValue:    0,
			configValue: "",
			wantErr:     true,
			errSubstr:   "must be positive",
		},
		{
			name:        "zero duration",
			flagValue:   "0s",
			envValue:    0,
			configValue: "",
			wantErr:     true,
			errSubstr:   "must be positive",
		},
		{
			name:        "hours duration",
			flagValue:   "1h",
			envValue:    0,
			configValue: "",
			want:        time.Hour,
			wantErr:     false,
		},
		{
			name:        "fractional seconds",
			flagValue:   "500ms",
			envValue:    0,
			configValue: "",
			want:        500 * time.Millisecond,
			wantErr:     false,
		},
		{
			name:        "complex duration",
			flagValue:   "1h30m45s",
			envValue:    0,
			configValue: "",
			want:        time.Hour + 30*time.Minute + 45*time.Second,
			wantErr:     false,
		},
		{
			name:        "invalid flag overrides valid env and config",
			flagValue:   "invalid",
			envValue:    time.Minute,
			configValue: "30s",
			wantErr:     true,
			errSubstr:   "invalid timeout",
		},
		{
			name:        "zero flag overrides valid env and config",
			flagValue:   "0s",
			envValue:    time.Minute,
			configValue: "30s",
			wantErr:     true,
			errSubstr:   "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveTimeoutWithEnv(tt.flagValue, tt.envValue, tt.configValue)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error should contain %q, got: %v", tt.errSubstr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("resolveTimeoutWithEnv(%q, %v, %q) = %v, want %v",
					tt.flagValue, tt.envValue, tt.configValue, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestLooksLikeMarkdown - Markdown file extension detection
// ---------------------------------------------------------------------------

func TestLooksLikeMarkdown(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := looksLikeMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeMarkdown(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRunMain - Main entry point exit codes
// ---------------------------------------------------------------------------

func TestRunMain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantCode     int
		wantInStdout []string
		wantInStderr []string
	}{
		{
			name:         "no args shows usage and exits with ExitUsage",
			args:         []string{"md2pdf"},
			wantCode:     ExitUsage,
			wantInStderr: []string{"Usage: md2pdf"},
		},
		{
			name:         "version command exits 0",
			args:         []string{"md2pdf", "version"},
			wantCode:     ExitSuccess,
			wantInStdout: []string{"md2pdf"},
		},
		{
			name:         "help command exits 0",
			args:         []string{"md2pdf", "help"},
			wantCode:     ExitSuccess,
			wantInStdout: []string{"Usage: md2pdf", "Commands:"},
		},
		{
			name:         "help convert shows convert help",
			args:         []string{"md2pdf", "help", "convert"},
			wantCode:     ExitSuccess,
			wantInStdout: []string{"Usage: md2pdf convert"},
		},
		{
			name:         "unknown command exits with ExitUsage",
			args:         []string{"md2pdf", "unknown"},
			wantCode:     ExitUsage,
			wantInStderr: []string{"unknown command: unknown"},
		},
		{
			name:         "legacy .md detection shows deprecation warning",
			args:         []string{"md2pdf", "nonexistent.md"},
			wantCode:     ExitIO, // File doesn't exist
			wantInStderr: []string{"DEPRECATED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader, _ := md2pdf.NewAssetLoader("")
			var stdout, stderr bytes.Buffer
			env := &Environment{
				Now:         func() time.Time { return time.Now() },
				Stdout:      &stdout,
				Stderr:      &stderr,
				AssetLoader: loader,
			}

			code := runMain(tt.args, env)

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

// ---------------------------------------------------------------------------
// TestRunMain_ExitCodes - Integration tests for semantic exit codes
// ---------------------------------------------------------------------------

func TestRunMain_ExitCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		// ExitSuccess (0)
		{
			name:     "version returns ExitSuccess",
			args:     []string{"md2pdf", "version"},
			wantCode: ExitSuccess,
		},
		{
			name:     "help returns ExitSuccess",
			args:     []string{"md2pdf", "help"},
			wantCode: ExitSuccess,
		},

		// ExitUsage (2)
		{
			name:     "no args returns ExitUsage",
			args:     []string{"md2pdf"},
			wantCode: ExitUsage,
		},
		{
			name:     "unknown command returns ExitUsage",
			args:     []string{"md2pdf", "badcmd"},
			wantCode: ExitUsage,
		},
		{
			name:     "unsupported shell returns ExitUsage",
			args:     []string{"md2pdf", "completion", "badshell"},
			wantCode: ExitUsage,
		},

		// ExitIO (3)
		{
			name:     "nonexistent file returns ExitIO",
			args:     []string{"md2pdf", "convert", "nonexistent.md"},
			wantCode: ExitIO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader, _ := md2pdf.NewAssetLoader("")
			var stdout, stderr bytes.Buffer
			env := &Environment{
				Now:         func() time.Time { return time.Now() },
				Stdout:      &stdout,
				Stderr:      &stderr,
				AssetLoader: loader,
			}

			code := runMain(tt.args, env)

			if code != tt.wantCode {
				t.Errorf("runMain(%v) = %d, want %d\nstderr: %s", tt.args, code, tt.wantCode, stderr.String())
			}
		})
	}
}
