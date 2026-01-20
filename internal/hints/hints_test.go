package hints

// Notes:
// - ForBrowserConnect tests cannot use t.Parallel() because they:
//   1. Use t.Setenv() which modifies process environment
//   2. Modify the package-level IsInContainer variable
// These are acceptable gaps: we test observable behavior through environment manipulation.

import (
	"strings"
	"testing"
)

func TestForBrowserConnect_InCI(t *testing.T) {
	// Save and restore IsInContainer (not parallel-safe, see package notes)
	orig := IsInContainer
	defer func() { IsInContainer = orig }()
	IsInContainer = func() bool { return false }

	t.Setenv("CI", "true")
	t.Setenv("ROD_NO_SANDBOX", "")
	t.Setenv("ROD_BROWSER_BIN", "")

	hint := ForBrowserConnect()

	if !strings.Contains(hint, "hint:") {
		t.Error("expected hint prefix")
	}
	if !strings.Contains(hint, "ROD_NO_SANDBOX") {
		t.Error("expected ROD_NO_SANDBOX suggestion in CI")
	}
	if !strings.Contains(hint, "ROD_BROWSER_BIN") {
		t.Error("expected ROD_BROWSER_BIN suggestion")
	}
}

func TestForBrowserConnect_InDocker(t *testing.T) {
	orig := IsInContainer
	defer func() { IsInContainer = orig }()
	IsInContainer = func() bool { return true }

	t.Setenv("CI", "")
	t.Setenv("ROD_NO_SANDBOX", "")
	t.Setenv("ROD_BROWSER_BIN", "")

	hint := ForBrowserConnect()

	if !strings.Contains(hint, "ROD_NO_SANDBOX") {
		t.Error("expected ROD_NO_SANDBOX suggestion in Docker")
	}
}

func TestForBrowserConnect_SandboxAlreadySet(t *testing.T) {
	orig := IsInContainer
	defer func() { IsInContainer = orig }()
	IsInContainer = func() bool { return true }

	t.Setenv("CI", "")
	t.Setenv("ROD_NO_SANDBOX", "1")
	t.Setenv("ROD_BROWSER_BIN", "")

	hint := ForBrowserConnect()

	if strings.Contains(hint, "ROD_NO_SANDBOX") {
		t.Error("should not suggest ROD_NO_SANDBOX when already set")
	}
}

func TestForBrowserConnect_BrowserBinAlreadySet(t *testing.T) {
	orig := IsInContainer
	defer func() { IsInContainer = orig }()
	IsInContainer = func() bool { return false }

	t.Setenv("CI", "")
	t.Setenv("ROD_NO_SANDBOX", "")
	t.Setenv("ROD_BROWSER_BIN", "/usr/bin/chrome")

	hint := ForBrowserConnect()

	if strings.Contains(hint, "ROD_BROWSER_BIN") {
		t.Error("should not suggest ROD_BROWSER_BIN when already set")
	}
}

func TestForBrowserConnect_NoHintsNeeded(t *testing.T) {
	orig := IsInContainer
	defer func() { IsInContainer = orig }()
	IsInContainer = func() bool { return false }

	t.Setenv("CI", "")
	t.Setenv("ROD_NO_SANDBOX", "")
	t.Setenv("ROD_BROWSER_BIN", "/usr/bin/chrome")

	hint := ForBrowserConnect()

	// Should have no sandbox hint (not in CI/Docker) but no browser hint
	if strings.Contains(hint, "ROD_BROWSER_BIN") {
		t.Error("should not suggest ROD_BROWSER_BIN when set")
	}
}

func TestForBrowserConnect_AllConfigured(t *testing.T) {
	orig := IsInContainer
	defer func() { IsInContainer = orig }()
	IsInContainer = func() bool { return true } // In Docker

	t.Setenv("CI", "true")
	t.Setenv("ROD_NO_SANDBOX", "1")
	t.Setenv("ROD_BROWSER_BIN", "/usr/bin/chrome")

	hint := ForBrowserConnect()

	// Both env vars set, should return empty hint
	if hint != "" {
		t.Errorf("expected empty hint when all configured, got %q", hint)
	}
}

func TestForTimeout(t *testing.T) {
	hint := ForTimeout()

	if !strings.Contains(hint, "hint:") {
		t.Error("expected hint prefix")
	}
	if !strings.Contains(hint, "--timeout") {
		t.Error("expected --timeout flag mention")
	}
}

func TestForConfigNotFound(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		wantHint bool
		contains string
	}{
		{
			name:     "empty paths",
			paths:    []string{},
			wantHint: true,
			contains: "--config",
		},
		{
			name:     "with paths",
			paths:    []string{"./foo.yaml", "~/.config/go-md2pdf/foo.yaml"},
			wantHint: true,
			contains: "go-md2pdf/foo.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := ForConfigNotFound(tt.paths)

			if tt.wantHint && !strings.Contains(hint, "hint:") {
				t.Error("expected hint prefix")
			}
			if !strings.Contains(hint, tt.contains) {
				t.Errorf("expected hint to contain %q, got %q", tt.contains, hint)
			}
		})
	}
}

func TestForOutputDirectory(t *testing.T) {
	hint := ForOutputDirectory()

	if !strings.Contains(hint, "hint:") {
		t.Error("expected hint prefix")
	}
	if !strings.Contains(hint, "parent directory") {
		t.Error("expected parent directory mention")
	}
}

func TestForStyleNotFound(t *testing.T) {
	tests := []struct {
		name      string
		available []string
		wantEmpty bool
		contains  string
	}{
		{
			name:      "empty available",
			available: []string{},
			wantEmpty: true,
		},
		{
			name:      "with styles",
			available: []string{"default", "technical"},
			contains:  "default, technical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := ForStyleNotFound(tt.available)

			if tt.wantEmpty && hint != "" {
				t.Errorf("expected empty hint, got %q", hint)
			}
			if !tt.wantEmpty && !strings.Contains(hint, tt.contains) {
				t.Errorf("expected hint to contain %q, got %q", tt.contains, hint)
			}
		})
	}
}

func TestForSignatureImage(t *testing.T) {
	hint := ForSignatureImage()

	if !strings.Contains(hint, "hint:") {
		t.Error("expected hint prefix")
	}
	if !strings.Contains(hint, "PNG") {
		t.Error("expected PNG format mention")
	}
	if !strings.Contains(hint, "URL") {
		t.Error("expected URL mention")
	}
}

func TestFormat_Consistency(t *testing.T) {
	// All hints should start with newline, spaces, and "hint:"
	hints := []string{
		ForTimeout(),
		ForOutputDirectory(),
		ForSignatureImage(),
	}

	for _, h := range hints {
		if !strings.HasPrefix(h, "\n  hint: ") {
			t.Errorf("hint format inconsistent: %q", h)
		}
	}
}
