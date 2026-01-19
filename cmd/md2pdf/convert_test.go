package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

func TestResolveCSSContent(t *testing.T) {
	t.Parallel()

	loader, _ := md2pdf.NewAssetLoader("")

	t.Run("empty style flag and no config returns default style", func(t *testing.T) {
		t.Parallel()
		got, err := resolveCSSContent("", nil, false, loader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "" {
			t.Error("expected default CSS content, got empty string")
		}
		// Verify it's the default style (contains our default.css markers)
		if !strings.Contains(got, "Default theme") {
			t.Error("expected default style content")
		}
	})

	t.Run("reads CSS file content", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "style.css")
		cssContent := "body { color: red; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		got, err := resolveCSSContent(cssPath, nil, false, loader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != cssContent {
			t.Errorf("got %q, want %q", got, cssContent)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		t.Parallel()

		_, err := resolveCSSContent("/nonexistent/style.css", nil, false, loader)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("config style loads from embedded assets", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Style: "creative"}
		got, err := resolveCSSContent("", cfg, false, loader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "" {
			t.Error("expected CSS content from embedded assets, got empty string")
		}
	})

	t.Run("css flag overrides config style", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "override.css")
		cssContent := "body { color: blue; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		cfg := &Config{Style: "creative"}
		got, err := resolveCSSContent(cssPath, cfg, false, loader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != cssContent {
			t.Errorf("got %q, want %q (flag should override config)", got, cssContent)
		}
	})

	t.Run("unknown config style returns error", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Style: "nonexistent"}
		_, err := resolveCSSContent("", cfg, false, loader)
		if err == nil {
			t.Error("expected error for unknown style")
		}
	})

	t.Run("noStyle flag returns empty even with config style", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Style: "creative"}
		got, err := resolveCSSContent("", cfg, true, loader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string (noStyle should disable CSS)", got)
		}
	})

	t.Run("noStyle flag returns empty even with css file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "style.css")
		if err := os.WriteFile(cssPath, []byte("body { color: red; }"), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		got, err := resolveCSSContent(cssPath, nil, true, loader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string (noStyle should disable CSS)", got)
		}
	})
}

func TestPrintResultsOutput(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for all success", func(t *testing.T) {
		t.Parallel()
		results := []ConversionResult{
			{InputPath: "a.md", OutputPath: "a.pdf", Err: nil},
			{InputPath: "b.md", OutputPath: "b.pdf", Err: nil},
		}
		failed := printResults(results, true, false)
		if failed != 0 {
			t.Errorf("failed = %d, want 0", failed)
		}
	})

	t.Run("returns count for failures", func(t *testing.T) {
		t.Parallel()

		results := []ConversionResult{
			{InputPath: "a.md", OutputPath: "a.pdf", Err: nil},
			{InputPath: "b.md", OutputPath: "b.pdf", Err: ErrReadMarkdown},
			{InputPath: "c.md", OutputPath: "c.pdf", Err: ErrReadMarkdown},
		}
		failed := printResults(results, true, false)
		if failed != 2 {
			t.Errorf("failed = %d, want 2", failed)
		}
	})

	t.Run("returns zero for empty results", func(t *testing.T) {
		t.Parallel()

		failed := printResults(nil, true, false)
		if failed != 0 {
			t.Errorf("failed = %d, want 0", failed)
		}
	})
}

func TestConvertFile_ErrorPaths(t *testing.T) {
	t.Parallel()

	// Mock converter that returns success
	mockConv := &staticMockConverter{result: []byte("%PDF-1.4 mock")}

	t.Run("mkdir failure returns error", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create a file where directory should be (blocks mkdir)
		blockingFile := filepath.Join(tempDir, "blocked")
		if err := os.WriteFile(blockingFile, []byte("blocker"), 0644); err != nil {
			t.Fatalf("failed to create blocking file: %v", err)
		}

		// Create input file
		inputPath := filepath.Join(tempDir, "doc.md")
		if err := os.WriteFile(inputPath, []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		// Try to output to a path under the blocking file (will fail mkdir)
		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(blockingFile, "subdir", "out.pdf"),
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Error("expected error when mkdir fails")
		}
	})

	t.Run("write failure returns ErrWritePDF", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create input file
		inputPath := filepath.Join(tempDir, "doc.md")
		if err := os.WriteFile(inputPath, []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		// Create output directory as read-only
		outDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(outDir, 0750); err != nil {
			t.Fatalf("failed to create output dir: %v", err)
		}
		if err := os.Chmod(outDir, 0500); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		t.Cleanup(func() {
			os.Chmod(outDir, 0750) // Restore for cleanup
		})

		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(outDir, "out.pdf"),
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Error("expected error when write fails")
		}
		if !errors.Is(result.Err, ErrWritePDF) {
			t.Errorf("expected ErrWritePDF, got: %v", result.Err)
		}
	})

	t.Run("read failure returns ErrReadMarkdown", func(t *testing.T) {
		t.Parallel()

		f := FileToConvert{
			InputPath:  "/nonexistent/doc.md",
			OutputPath: "/tmp/out.pdf",
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Error("expected error when read fails")
		}
		if !errors.Is(result.Err, ErrReadMarkdown) {
			t.Errorf("expected ErrReadMarkdown, got: %v", result.Err)
		}
	})
}

func TestHtmlOutputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pdfPath string
		want    string
	}{
		{
			name:    "simple pdf extension",
			pdfPath: "output.pdf",
			want:    "output.html",
		},
		{
			name:    "absolute path with pdf extension",
			pdfPath: "/path/to/doc.pdf",
			want:    "/path/to/doc.html",
		},
		{
			name:    "no pdf extension",
			pdfPath: "file",
			want:    "file.html",
		},
		{
			name:    "uppercase PDF not replaced (case-sensitive)",
			pdfPath: "doc.PDF",
			want:    "doc.PDF.html",
		},
		{
			name:    "multiple dots in filename",
			pdfPath: "my.report.v2.pdf",
			want:    "my.report.v2.html",
		},
		{
			name:    "Windows path",
			pdfPath: "C:\\Documents\\report.pdf",
			want:    "C:\\Documents\\report.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := htmlOutputPath(tt.pdfPath)
			if got != tt.want {
				t.Errorf("htmlOutputPath(%q) = %q, want %q", tt.pdfPath, got, tt.want)
			}
		})
	}
}

func TestLoadTemplateSetFromDir(t *testing.T) {
	t.Parallel()

	t.Run("valid directory with both templates", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		coverContent := "<div class=\"cover\">Cover Page</div>"
		sigContent := "<div class=\"signature\">Signature Block</div>"

		if err := os.WriteFile(filepath.Join(tmpDir, "cover.html"), []byte(coverContent), 0644); err != nil {
			t.Fatalf("failed to write cover.html: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "signature.html"), []byte(sigContent), 0644); err != nil {
			t.Fatalf("failed to write signature.html: %v", err)
		}

		ts, err := loadTemplateSetFromDir(tmpDir)
		if err != nil {
			t.Fatalf("loadTemplateSetFromDir() error = %v", err)
		}

		if ts.Cover != coverContent {
			t.Errorf("Cover = %q, want %q", ts.Cover, coverContent)
		}
		if ts.Signature != sigContent {
			t.Errorf("Signature = %q, want %q", ts.Signature, sigContent)
		}
		if ts.Name != tmpDir {
			t.Errorf("Name = %q, want %q", ts.Name, tmpDir)
		}
	})

	t.Run("missing cover returns ErrIncompleteTemplateSet", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "signature.html"), []byte("<sig/>"), 0644); err != nil {
			t.Fatalf("failed to write signature.html: %v", err)
		}

		_, err := loadTemplateSetFromDir(tmpDir)
		if !errors.Is(err, md2pdf.ErrIncompleteTemplateSet) {
			t.Errorf("loadTemplateSetFromDir() error = %v, want ErrIncompleteTemplateSet", err)
		}
	})

	t.Run("missing signature returns ErrIncompleteTemplateSet", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "cover.html"), []byte("<cover/>"), 0644); err != nil {
			t.Fatalf("failed to write cover.html: %v", err)
		}

		_, err := loadTemplateSetFromDir(tmpDir)
		if !errors.Is(err, md2pdf.ErrIncompleteTemplateSet) {
			t.Errorf("loadTemplateSetFromDir() error = %v, want ErrIncompleteTemplateSet", err)
		}
	})

	t.Run("empty directory returns ErrTemplateSetNotFound", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		_, err := loadTemplateSetFromDir(tmpDir)
		if !errors.Is(err, md2pdf.ErrTemplateSetNotFound) {
			t.Errorf("loadTemplateSetFromDir() error = %v, want ErrTemplateSetNotFound", err)
		}
	})

	t.Run("nonexistent directory returns ErrTemplateSetNotFound", func(t *testing.T) {
		t.Parallel()

		_, err := loadTemplateSetFromDir("/nonexistent/path/to/templates")
		if !errors.Is(err, md2pdf.ErrTemplateSetNotFound) {
			t.Errorf("loadTemplateSetFromDir() error = %v, want ErrTemplateSetNotFound", err)
		}
	})
}

func TestResolveTemplateSet(t *testing.T) {
	t.Parallel()

	t.Run("empty value loads default template set", func(t *testing.T) {
		t.Parallel()

		loader := &mockTemplateLoader{
			templateSets: map[string]*md2pdf.TemplateSet{
				"default": {Name: "default", Cover: "<cover/>", Signature: "<sig/>"},
			},
		}

		ts, err := resolveTemplateSet("", loader)
		if err != nil {
			t.Fatalf("resolveTemplateSet() error = %v", err)
		}
		if ts.Name != "default" {
			t.Errorf("Name = %q, want %q", ts.Name, "default")
		}
	})

	t.Run("name loads from loader", func(t *testing.T) {
		t.Parallel()

		loader := &mockTemplateLoader{
			templateSets: map[string]*md2pdf.TemplateSet{
				"corporate": {Name: "corporate", Cover: "<corp-cover/>", Signature: "<corp-sig/>"},
			},
		}

		ts, err := resolveTemplateSet("corporate", loader)
		if err != nil {
			t.Fatalf("resolveTemplateSet() error = %v", err)
		}
		if ts.Name != "corporate" {
			t.Errorf("Name = %q, want %q", ts.Name, "corporate")
		}
	})

	t.Run("path loads from filesystem", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "cover.html"), []byte("<cover/>"), 0644); err != nil {
			t.Fatalf("failed to write cover.html: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "signature.html"), []byte("<sig/>"), 0644); err != nil {
			t.Fatalf("failed to write signature.html: %v", err)
		}

		// Use path-like value (contains /)
		pathValue := tmpDir + "/"

		loader := &mockTemplateLoader{} // Should not be called for paths

		ts, err := resolveTemplateSet(pathValue, loader)
		if err != nil {
			t.Fatalf("resolveTemplateSet() error = %v", err)
		}
		if ts.Cover != "<cover/>" {
			t.Errorf("Cover = %q, want %q", ts.Cover, "<cover/>")
		}
	})

	t.Run("nonexistent name returns error", func(t *testing.T) {
		t.Parallel()

		loader := &mockTemplateLoader{
			templateSets: map[string]*md2pdf.TemplateSet{},
		}

		_, err := resolveTemplateSet("nonexistent", loader)
		if !errors.Is(err, md2pdf.ErrTemplateSetNotFound) {
			t.Errorf("resolveTemplateSet() error = %v, want ErrTemplateSetNotFound", err)
		}
	})
}

