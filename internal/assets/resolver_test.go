package assets

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAssetResolver(t *testing.T) {
	t.Parallel()

	t.Run("empty path uses embedded only", func(t *testing.T) {
		t.Parallel()

		resolver, err := NewAssetResolver("")
		if err != nil {
			t.Fatalf("NewAssetResolver(\"\") error = %v", err)
		}
		if resolver == nil {
			t.Fatal("NewAssetResolver() returned nil")
		}
		if resolver.HasCustomLoader() {
			t.Error("expected no custom loader for empty path")
		}
	})

	t.Run("valid custom path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		resolver, err := NewAssetResolver(tmpDir)
		if err != nil {
			t.Fatalf("NewAssetResolver() error = %v", err)
		}
		if !resolver.HasCustomLoader() {
			t.Error("expected custom loader for valid path")
		}
	})

	t.Run("invalid custom path returns error", func(t *testing.T) {
		t.Parallel()

		_, err := NewAssetResolver("/nonexistent/path/abc123xyz")
		if !errors.Is(err, ErrInvalidBasePath) {
			t.Errorf("NewAssetResolver() error = %v, want ErrInvalidBasePath", err)
		}
	})
}

func TestAssetResolver_LoadStyle_EmbeddedOnly(t *testing.T) {
	t.Parallel()

	resolver, err := NewAssetResolver("")
	if err != nil {
		t.Fatalf("NewAssetResolver() error = %v", err)
	}

	t.Run("loads embedded style", func(t *testing.T) {
		t.Parallel()

		got, err := resolver.LoadStyle("creative")
		if err != nil {
			t.Fatalf("LoadStyle() error = %v", err)
		}
		if got == "" {
			t.Error("LoadStyle() returned empty content")
		}
	})

	t.Run("returns error for nonexistent", func(t *testing.T) {
		t.Parallel()

		_, err := resolver.LoadStyle("nonexistent-xyz")
		if !errors.Is(err, ErrStyleNotFound) {
			t.Errorf("LoadStyle() error = %v, want ErrStyleNotFound", err)
		}
	})
}

func TestAssetResolver_LoadStyle_CustomWithFallback(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	stylesDir := filepath.Join(tmpDir, "styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		t.Fatalf("failed to create styles dir: %v", err)
	}

	// Create a custom style
	customCSS := "/* custom style */"
	if err := os.WriteFile(filepath.Join(stylesDir, "mystyle.css"), []byte(customCSS), 0644); err != nil {
		t.Fatalf("failed to write CSS file: %v", err)
	}

	resolver, err := NewAssetResolver(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetResolver() error = %v", err)
	}

	t.Run("loads custom style when available", func(t *testing.T) {
		t.Parallel()

		got, err := resolver.LoadStyle("mystyle")
		if err != nil {
			t.Fatalf("LoadStyle() error = %v", err)
		}
		if got != customCSS {
			t.Errorf("LoadStyle() = %q, want %q", got, customCSS)
		}
	})

	t.Run("falls back to embedded when custom not found", func(t *testing.T) {
		t.Parallel()

		got, err := resolver.LoadStyle("creative")
		if err != nil {
			t.Fatalf("LoadStyle() error = %v", err)
		}
		if got == "" {
			t.Error("LoadStyle() returned empty content from fallback")
		}
	})

	t.Run("returns error when neither has style", func(t *testing.T) {
		t.Parallel()

		_, err := resolver.LoadStyle("nonexistent-xyz")
		if !errors.Is(err, ErrStyleNotFound) {
			t.Errorf("LoadStyle() error = %v, want ErrStyleNotFound", err)
		}
	})
}

func TestAssetResolver_LoadStyle_CustomOverridesEmbedded(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	stylesDir := filepath.Join(tmpDir, "styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		t.Fatalf("failed to create styles dir: %v", err)
	}

	// Create a custom style with the same name as an embedded one
	customCSS := "/* CUSTOM OVERRIDE of creative */"
	if err := os.WriteFile(filepath.Join(stylesDir, "creative.css"), []byte(customCSS), 0644); err != nil {
		t.Fatalf("failed to write CSS file: %v", err)
	}

	resolver, err := NewAssetResolver(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetResolver() error = %v", err)
	}

	got, err := resolver.LoadStyle("creative")
	if err != nil {
		t.Fatalf("LoadStyle() error = %v", err)
	}
	if got != customCSS {
		t.Errorf("LoadStyle() = %q, want custom override %q", got, customCSS)
	}
}

func TestAssetResolver_LoadTemplateSet_EmbeddedOnly(t *testing.T) {
	t.Parallel()

	resolver, err := NewAssetResolver("")
	if err != nil {
		t.Fatalf("NewAssetResolver() error = %v", err)
	}

	t.Run("loads embedded template set", func(t *testing.T) {
		t.Parallel()

		ts, err := resolver.LoadTemplateSet("default")
		if err != nil {
			t.Fatalf("LoadTemplateSet() error = %v", err)
		}
		if ts.Cover == "" {
			t.Error("LoadTemplateSet() returned empty cover")
		}
		if ts.Signature == "" {
			t.Error("LoadTemplateSet() returned empty signature")
		}
	})

	t.Run("returns error for nonexistent", func(t *testing.T) {
		t.Parallel()

		_, err := resolver.LoadTemplateSet("nonexistent-xyz")
		if !errors.Is(err, ErrTemplateSetNotFound) {
			t.Errorf("LoadTemplateSet() error = %v, want ErrTemplateSetNotFound", err)
		}
	})
}

func TestAssetResolver_LoadTemplateSet_CustomWithFallback(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	setDir := filepath.Join(tmpDir, "templates", "custom")
	if err := os.MkdirAll(setDir, 0755); err != nil {
		t.Fatalf("failed to create template set dir: %v", err)
	}

	// Create a custom template set
	customCover := "<div>custom cover</div>"
	customSig := "<div>custom signature</div>"
	if err := os.WriteFile(filepath.Join(setDir, "cover.html"), []byte(customCover), 0644); err != nil {
		t.Fatalf("failed to write cover file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(setDir, "signature.html"), []byte(customSig), 0644); err != nil {
		t.Fatalf("failed to write signature file: %v", err)
	}

	resolver, err := NewAssetResolver(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetResolver() error = %v", err)
	}

	t.Run("loads custom template set when available", func(t *testing.T) {
		t.Parallel()

		ts, err := resolver.LoadTemplateSet("custom")
		if err != nil {
			t.Fatalf("LoadTemplateSet() error = %v", err)
		}
		if ts.Cover != customCover {
			t.Errorf("LoadTemplateSet() cover = %q, want %q", ts.Cover, customCover)
		}
	})

	t.Run("falls back to embedded when custom not found", func(t *testing.T) {
		t.Parallel()

		ts, err := resolver.LoadTemplateSet("default")
		if err != nil {
			t.Fatalf("LoadTemplateSet() error = %v", err)
		}
		if ts.Cover == "" {
			t.Error("LoadTemplateSet() returned empty cover from fallback")
		}
	})
}

func TestAssetResolver_LoadTemplateSet_CustomOverridesEmbedded(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	setDir := filepath.Join(tmpDir, "templates", "default")
	if err := os.MkdirAll(setDir, 0755); err != nil {
		t.Fatalf("failed to create template set dir: %v", err)
	}

	// Create a custom template set with the same name as the embedded one
	customCover := "<!-- CUSTOM OVERRIDE --><div>my cover</div>"
	customSig := "<!-- CUSTOM OVERRIDE --><div>my sig</div>"
	if err := os.WriteFile(filepath.Join(setDir, "cover.html"), []byte(customCover), 0644); err != nil {
		t.Fatalf("failed to write cover file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(setDir, "signature.html"), []byte(customSig), 0644); err != nil {
		t.Fatalf("failed to write signature file: %v", err)
	}

	resolver, err := NewAssetResolver(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetResolver() error = %v", err)
	}

	ts, err := resolver.LoadTemplateSet("default")
	if err != nil {
		t.Fatalf("LoadTemplateSet() error = %v", err)
	}
	if ts.Cover != customCover {
		t.Errorf("LoadTemplateSet() cover = %q, want custom override %q", ts.Cover, customCover)
	}
}

func TestAssetResolver_ValidationErrorsNotFallenBack(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	resolver, err := NewAssetResolver(tmpDir)
	if err != nil {
		t.Fatalf("NewAssetResolver() error = %v", err)
	}

	t.Run("style validation error not fallen back", func(t *testing.T) {
		t.Parallel()

		_, err := resolver.LoadStyle("../secret")
		if !errors.Is(err, ErrInvalidAssetName) {
			t.Errorf("LoadStyle() error = %v, want ErrInvalidAssetName (no fallback)", err)
		}
	})

	t.Run("template set validation error not fallen back", func(t *testing.T) {
		t.Parallel()

		_, err := resolver.LoadTemplateSet("../secret")
		if !errors.Is(err, ErrInvalidAssetName) {
			t.Errorf("LoadTemplateSet() error = %v, want ErrInvalidAssetName (no fallback)", err)
		}
	})
}

func TestAssetResolver_HasCustomLoader(t *testing.T) {
	t.Parallel()

	t.Run("false when no custom path", func(t *testing.T) {
		t.Parallel()

		resolver, _ := NewAssetResolver("")
		if resolver.HasCustomLoader() {
			t.Error("HasCustomLoader() = true, want false")
		}
	})

	t.Run("true when custom path set", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		resolver, _ := NewAssetResolver(tmpDir)
		if !resolver.HasCustomLoader() {
			t.Error("HasCustomLoader() = false, want true")
		}
	})
}

func TestAssetResolver_ImplementsAssetLoader(t *testing.T) {
	t.Parallel()

	var _ AssetLoader = (*AssetResolver)(nil)
}

func TestIsNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrStyleNotFound", ErrStyleNotFound, true},
		{"ErrTemplateNotFound", ErrTemplateNotFound, true},
		{"ErrTemplateSetNotFound", ErrTemplateSetNotFound, true},
		{"wrapped ErrStyleNotFound", errors.New("wrap: " + ErrStyleNotFound.Error()), false},
		{"ErrInvalidAssetName", ErrInvalidAssetName, false},
		{"ErrAssetRead", ErrAssetRead, false},
		{"generic error", errors.New("some error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isNotFoundError(tt.err)
			if got != tt.want {
				t.Errorf("isNotFoundError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
