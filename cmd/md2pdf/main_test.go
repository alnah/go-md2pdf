package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
)

// wrongTypeConverter is a Converter that is NOT *md2pdf.Service.
type wrongTypeConverter struct{}

func (w *wrongTypeConverter) Convert(_ context.Context, _ md2pdf.Input) (*md2pdf.ConvertResult, error) {
	return &md2pdf.ConvertResult{PDF: []byte("%PDF-1.4 mock")}, nil
}

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

func TestPoolAdapter_Size(t *testing.T) {
	t.Parallel()

	pool := md2pdf.NewConverterPool(3)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	if adapter.Size() != 3 {
		t.Errorf("Size() = %d, want 3", adapter.Size())
	}
}

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

func TestResolveTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		flagValue   string
		configValue string
		want        time.Duration
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "both empty uses default",
			flagValue:   "",
			configValue: "",
			want:        0,
			wantErr:     false,
		},
		{
			name:        "flag only",
			flagValue:   "2m",
			configValue: "",
			want:        2 * time.Minute,
			wantErr:     false,
		},
		{
			name:        "config only",
			flagValue:   "",
			configValue: "30s",
			want:        30 * time.Second,
			wantErr:     false,
		},
		{
			name:        "flag overrides config",
			flagValue:   "5m",
			configValue: "30s",
			want:        5 * time.Minute,
			wantErr:     false,
		},
		{
			name:        "combined duration",
			flagValue:   "1m30s",
			configValue: "",
			want:        90 * time.Second,
			wantErr:     false,
		},
		{
			name:        "invalid flag format",
			flagValue:   "abc",
			configValue: "",
			wantErr:     true,
			errSubstr:   "invalid timeout",
		},
		{
			name:        "invalid config format",
			flagValue:   "",
			configValue: "xyz",
			wantErr:     true,
			errSubstr:   "invalid timeout",
		},
		{
			name:        "negative duration",
			flagValue:   "-5s",
			configValue: "",
			wantErr:     true,
			errSubstr:   "must be positive",
		},
		{
			name:        "zero duration",
			flagValue:   "0s",
			configValue: "",
			wantErr:     true,
			errSubstr:   "must be positive",
		},
		{
			name:        "hours duration",
			flagValue:   "1h",
			configValue: "",
			want:        time.Hour,
			wantErr:     false,
		},
		{
			name:        "fractional seconds",
			flagValue:   "500ms",
			configValue: "",
			want:        500 * time.Millisecond,
			wantErr:     false,
		},
		{
			name:        "complex duration",
			flagValue:   "1h30m45s",
			configValue: "",
			want:        time.Hour + 30*time.Minute + 45*time.Second,
			wantErr:     false,
		},
		{
			name:        "invalid flag overrides valid config",
			flagValue:   "invalid",
			configValue: "30s",
			wantErr:     true,
			errSubstr:   "invalid timeout",
		},
		{
			name:        "zero flag overrides valid config",
			flagValue:   "0s",
			configValue: "30s",
			wantErr:     true,
			errSubstr:   "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveTimeout(tt.flagValue, tt.configValue)
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
				t.Errorf("resolveTimeout(%q, %q) = %v, want %v", tt.flagValue, tt.configValue, got, tt.want)
			}
		})
	}
}

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
