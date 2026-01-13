package md2pdf

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTemplateSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tsName    string
		cover     string
		signature string
	}{
		{
			name:      "basic template set",
			tsName:    "test",
			cover:     "<div>cover</div>",
			signature: "<div>signature</div>",
		},
		{
			name:      "empty strings",
			tsName:    "",
			cover:     "",
			signature: "",
		},
		{
			name:      "with HTML content",
			tsName:    "full",
			cover:     "<!DOCTYPE html><html><body>Cover Page</body></html>",
			signature: "<div class=\"sig\">{{.Name}}</div>",
		},
		{
			name:      "with template variables",
			tsName:    "templated",
			cover:     "<h1>{{.Title}}</h1><p>{{.Author}}</p>",
			signature: "<p>Signed by {{.Name}} on {{.Date}}</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := NewTemplateSet(tt.tsName, tt.cover, tt.signature)

			if ts.Name != tt.tsName {
				t.Errorf("Name = %q, want %q", ts.Name, tt.tsName)
			}
			if ts.Cover != tt.cover {
				t.Errorf("Cover = %q, want %q", ts.Cover, tt.cover)
			}
			if ts.Signature != tt.signature {
				t.Errorf("Signature = %q, want %q", ts.Signature, tt.signature)
			}
		})
	}
}

func TestNewAssetLoader_EmptyPath(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader(\"\") error = %v", err)
	}

	// Verify it can load default style
	css, err := loader.LoadStyle(DefaultStyle)
	if err != nil {
		t.Errorf("LoadStyle(%q) error = %v", DefaultStyle, err)
	}
	if css == "" {
		t.Error("LoadStyle returned empty CSS for default style")
	}

	// Verify it can load default template set
	ts, err := loader.LoadTemplateSet(DefaultTemplateSet)
	if err != nil {
		t.Fatalf("LoadTemplateSet(%q) error = %v", DefaultTemplateSet, err)
	}
	if ts == nil {
		t.Fatal("LoadTemplateSet returned nil")
	}
	if ts.Cover == "" {
		t.Error("TemplateSet.Cover is empty")
	}
	if ts.Signature == "" {
		t.Error("TemplateSet.Signature is empty")
	}
}

func TestNewAssetLoader_InvalidPath(t *testing.T) {
	t.Parallel()

	_, err := NewAssetLoader("/nonexistent/path/to/assets")
	if err == nil {
		t.Fatal("NewAssetLoader() expected error for invalid path, got nil")
	}
	if !errors.Is(err, ErrInvalidAssetPath) {
		t.Errorf("NewAssetLoader() error = %v, want ErrInvalidAssetPath", err)
	}
}

func TestNewAssetLoader_ValidPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	loader, err := NewAssetLoader(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetLoader(%q) error = %v", tmpDir, err)
	}

	// Empty directory should fall back to embedded assets
	css, err := loader.LoadStyle(DefaultStyle)
	if err != nil {
		t.Errorf("LoadStyle with fallback error = %v", err)
	}
	if css == "" {
		t.Error("Fallback to embedded style failed")
	}
}

func TestNewAssetLoader_CustomStyleOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create custom style directory and file
	stylesDir := filepath.Join(tmpDir, "styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		t.Fatalf("failed to create styles dir: %v", err)
	}

	customCSS := "/* custom override */ body { color: red; }"
	if err := os.WriteFile(filepath.Join(stylesDir, "default.css"), []byte(customCSS), 0644); err != nil {
		t.Fatalf("failed to write custom CSS: %v", err)
	}

	loader, err := NewAssetLoader(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetLoader(%q) error = %v", tmpDir, err)
	}

	// Should load custom style instead of embedded
	css, err := loader.LoadStyle(DefaultStyle)
	if err != nil {
		t.Errorf("LoadStyle error = %v", err)
	}
	if css != customCSS {
		t.Errorf("LoadStyle = %q, want custom CSS %q", css, customCSS)
	}
}

func TestAssetLoader_StyleNotFound(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader error = %v", err)
	}

	_, err = loader.LoadStyle("nonexistent-style")
	if err == nil {
		t.Fatal("LoadStyle() expected error for nonexistent style, got nil")
	}
	if !errors.Is(err, ErrStyleNotFound) {
		t.Errorf("LoadStyle() error = %v, want ErrStyleNotFound", err)
	}
}

func TestAssetLoader_TemplateSetNotFound(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader error = %v", err)
	}

	_, err = loader.LoadTemplateSet("nonexistent-templates")
	if err == nil {
		t.Fatal("LoadTemplateSet() expected error for nonexistent template set, got nil")
	}
	if !errors.Is(err, ErrTemplateSetNotFound) {
		t.Errorf("LoadTemplateSet() error = %v, want ErrTemplateSetNotFound", err)
	}
}

func TestDefaultConstants(t *testing.T) {
	t.Parallel()

	if DefaultStyle != "default" {
		t.Errorf("DefaultStyle = %q, want \"default\"", DefaultStyle)
	}
	if DefaultTemplateSet != "default" {
		t.Errorf("DefaultTemplateSet = %q, want \"default\"", DefaultTemplateSet)
	}
}

func TestErrorWrapping_PreservesMessage(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader error = %v", err)
	}

	_, err = loader.LoadStyle("custom-style")

	// Error message should contain the style name
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("error message should not be empty")
	}
	// The message should mention the style name
	if !strings.Contains(errMsg, "custom-style") {
		t.Errorf("error message %q should contain style name", errMsg)
	}
}

func TestErrorWrapping_UnwrapsToSentinel(t *testing.T) {
	t.Parallel()

	loader, err := NewAssetLoader("")
	if err != nil {
		t.Fatalf("NewAssetLoader error = %v", err)
	}

	// Test ErrStyleNotFound
	_, styleErr := loader.LoadStyle("nonexistent")
	if !errors.Is(styleErr, ErrStyleNotFound) {
		t.Errorf("style error should unwrap to ErrStyleNotFound, got %v", styleErr)
	}

	// Test ErrTemplateSetNotFound
	_, tsErr := loader.LoadTemplateSet("nonexistent")
	if !errors.Is(tsErr, ErrTemplateSetNotFound) {
		t.Errorf("template set error should unwrap to ErrTemplateSetNotFound, got %v", tsErr)
	}
}

func TestNewAssetLoader_CustomTemplateOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create custom template directory and files
	templatesDir := filepath.Join(tmpDir, "templates", "default")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	customCover := "<div>Custom Cover</div>"
	customSig := "<div>Custom Signature</div>"
	if err := os.WriteFile(filepath.Join(templatesDir, "cover.html"), []byte(customCover), 0644); err != nil {
		t.Fatalf("failed to write cover.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "signature.html"), []byte(customSig), 0644); err != nil {
		t.Fatalf("failed to write signature.html: %v", err)
	}

	loader, err := NewAssetLoader(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetLoader(%q) error = %v", tmpDir, err)
	}

	// Should load custom templates instead of embedded
	ts, err := loader.LoadTemplateSet(DefaultTemplateSet)
	if err != nil {
		t.Errorf("LoadTemplateSet error = %v", err)
	}
	if ts.Cover != customCover {
		t.Errorf("Cover = %q, want %q", ts.Cover, customCover)
	}
	if ts.Signature != customSig {
		t.Errorf("Signature = %q, want %q", ts.Signature, customSig)
	}
}

func TestWrappedAssetError_Error(t *testing.T) {
	t.Parallel()

	original := errors.New("original error message")
	sentinel := errors.New("sentinel")

	wrapped := wrapError(sentinel, original)

	// Error() should return original message
	if wrapped.Error() != original.Error() {
		t.Errorf("Error() = %q, want %q", wrapped.Error(), original.Error())
	}
}

func TestWrappedAssetError_Unwrap(t *testing.T) {
	t.Parallel()

	original := errors.New("original error message")
	sentinel := errors.New("sentinel")

	wrapped := wrapError(sentinel, original)

	// Unwrap should return sentinel (for errors.Is)
	var unwrapped interface{ Unwrap() error }
	if errors.As(wrapped, &unwrapped) {
		if unwrapped.Unwrap() != sentinel {
			t.Errorf("Unwrap() = %v, want %v", unwrapped.Unwrap(), sentinel)
		}
	} else {
		t.Error("wrapped error should implement Unwrap()")
	}

	// errors.Is should match sentinel
	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is(wrapped, sentinel) should be true")
	}

	// errors.Is should NOT match original
	if errors.Is(wrapped, original) {
		t.Error("errors.Is(wrapped, original) should be false")
	}
}

func TestConvertAssetError_NilError(t *testing.T) {
	t.Parallel()

	result := convertAssetError(nil)
	if result != nil {
		t.Errorf("convertAssetError(nil) = %v, want nil", result)
	}
}

func TestIsError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same error",
			err:    sentinel,
			target: sentinel,
			want:   true,
		},
		{
			name:   "nil error nil target",
			err:    nil,
			target: nil,
			want:   true,
		},
		{
			name:   "nil error non-nil target",
			err:    nil,
			target: sentinel,
			want:   false,
		},
		{
			name:   "non-nil error nil target",
			err:    sentinel,
			target: nil,
			want:   false,
		},
		{
			name:   "different errors",
			err:    errors.New("other"),
			target: sentinel,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isError(tt.err, tt.target)
			if got != tt.want {
				t.Errorf("isError() = %v, want %v", got, tt.want)
			}
		})
	}
}
