package assets

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFilesystemLoader(t *testing.T) {
	t.Parallel()

	t.Run("valid directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader() error = %v", err)
		}
		if loader == nil {
			t.Fatal("NewFilesystemLoader() returned nil")
		}
	})

	t.Run("empty path returns error", func(t *testing.T) {
		t.Parallel()

		_, err := NewFilesystemLoader("")
		if !errors.Is(err, ErrInvalidBasePath) {
			t.Errorf("NewFilesystemLoader(\"\") error = %v, want ErrInvalidBasePath", err)
		}
	})

	t.Run("nonexistent directory returns error", func(t *testing.T) {
		t.Parallel()

		_, err := NewFilesystemLoader("/nonexistent/path/abc123xyz")
		if !errors.Is(err, ErrInvalidBasePath) {
			t.Errorf("NewFilesystemLoader() error = %v, want ErrInvalidBasePath", err)
		}
	})

	t.Run("file instead of directory returns error", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := NewFilesystemLoader(filePath)
		if !errors.Is(err, ErrInvalidBasePath) {
			t.Errorf("NewFilesystemLoader() error = %v, want ErrInvalidBasePath", err)
		}
	})
}

func TestFilesystemLoader_LoadStyle(t *testing.T) {
	t.Parallel()

	t.Run("loads existing style", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		stylesDir := filepath.Join(tmpDir, "styles")
		if err := os.MkdirAll(stylesDir, 0755); err != nil {
			t.Fatalf("failed to create styles dir: %v", err)
		}

		cssContent := "body { color: red; }"
		if err := os.WriteFile(filepath.Join(stylesDir, "custom.css"), []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader() error = %v", err)
		}

		got, err := loader.LoadStyle("custom")
		if err != nil {
			t.Fatalf("LoadStyle() error = %v", err)
		}
		if got != cssContent {
			t.Errorf("LoadStyle() = %q, want %q", got, cssContent)
		}
	})

	t.Run("returns ErrStyleNotFound for nonexistent", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		stylesDir := filepath.Join(tmpDir, "styles")
		if err := os.MkdirAll(stylesDir, 0755); err != nil {
			t.Fatalf("failed to create styles dir: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader() error = %v", err)
		}

		_, err = loader.LoadStyle("nonexistent")
		if !errors.Is(err, ErrStyleNotFound) {
			t.Errorf("LoadStyle() error = %v, want ErrStyleNotFound", err)
		}
	})

	t.Run("returns ErrInvalidAssetName for invalid name", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader() error = %v", err)
		}

		tests := []string{"", "../secret", "..\\secret", "style.evil"}
		for _, name := range tests {
			_, err := loader.LoadStyle(name)
			if !errors.Is(err, ErrInvalidAssetName) {
				t.Errorf("LoadStyle(%q) error = %v, want ErrInvalidAssetName", name, err)
			}
		}
	})
}

func TestFilesystemLoader_LoadTemplate(t *testing.T) {
	t.Parallel()

	t.Run("loads existing template", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		templatesDir := filepath.Join(tmpDir, "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		htmlContent := "<div>custom template</div>"
		if err := os.WriteFile(filepath.Join(templatesDir, "mytemplate.html"), []byte(htmlContent), 0644); err != nil {
			t.Fatalf("failed to write HTML file: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader() error = %v", err)
		}

		got, err := loader.LoadTemplate("mytemplate")
		if err != nil {
			t.Fatalf("LoadTemplate() error = %v", err)
		}
		if got != htmlContent {
			t.Errorf("LoadTemplate() = %q, want %q", got, htmlContent)
		}
	})

	t.Run("returns ErrTemplateNotFound for nonexistent", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		templatesDir := filepath.Join(tmpDir, "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader() error = %v", err)
		}

		_, err = loader.LoadTemplate("nonexistent")
		if !errors.Is(err, ErrTemplateNotFound) {
			t.Errorf("LoadTemplate() error = %v, want ErrTemplateNotFound", err)
		}
	})

	t.Run("returns ErrInvalidAssetName for invalid name", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader() error = %v", err)
		}

		tests := []string{"", "../secret", "..\\secret", "template.evil"}
		for _, name := range tests {
			_, err := loader.LoadTemplate(name)
			if !errors.Is(err, ErrInvalidAssetName) {
				t.Errorf("LoadTemplate(%q) error = %v, want ErrInvalidAssetName", name, err)
			}
		}
	})
}

func TestFilesystemLoader_PathContainment(t *testing.T) {
	t.Parallel()

	t.Run("rejects symlink escape attempt", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		stylesDir := filepath.Join(tmpDir, "styles")
		if err := os.MkdirAll(stylesDir, 0755); err != nil {
			t.Fatalf("failed to create styles dir: %v", err)
		}

		// Create a secret file outside the base path
		secretDir := t.TempDir()
		secretFile := filepath.Join(secretDir, "secret.css")
		if err := os.WriteFile(secretFile, []byte("secret content"), 0644); err != nil {
			t.Fatalf("failed to write secret file: %v", err)
		}

		// Create symlink inside styles pointing outside
		symlinkPath := filepath.Join(stylesDir, "evil.css")
		if err := os.Symlink(secretFile, symlinkPath); err != nil {
			t.Skipf("symlink creation not supported: %v", err)
		}

		loader, err := NewFilesystemLoader(tmpDir)
		if err != nil {
			t.Fatalf("NewFilesystemLoader() error = %v", err)
		}

		// The symlink resolves to a path outside basePath
		// verifyPathContainment uses EvalSymlinks to detect this
		_, err = loader.LoadStyle("evil")
		if !errors.Is(err, ErrPathTraversal) {
			t.Errorf("LoadStyle() with symlink escape error = %v, want ErrPathTraversal", err)
		}
	})
}

func TestFilesystemLoader_ImplementsAssetLoader(t *testing.T) {
	t.Parallel()

	var _ AssetLoader = (*FilesystemLoader)(nil)
}
