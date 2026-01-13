//go:build integration

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

// concurrentTestFiles is the number of files to create for concurrent conversion tests.
const concurrentTestFiles = 10

// integrationPool wraps md2pdf.ServicePool for integration testing.
type integrationPool struct {
	pool *md2pdf.ServicePool
}

func newIntegrationPool(size int) *integrationPool {
	return &integrationPool{pool: md2pdf.NewServicePool(size)}
}

func (p *integrationPool) Acquire() Converter {
	return p.pool.Acquire()
}

func (p *integrationPool) Release(c Converter) {
	if svc, ok := c.(*md2pdf.Service); ok {
		p.pool.Release(svc)
	}
}

func (p *integrationPool) Size() int {
	return p.pool.Size()
}

func (p *integrationPool) Close() error {
	return p.pool.Close()
}

// runIntegration is a helper that runs the CLI with a real service pool.
func runIntegration(args []string) error {
	pool := newIntegrationPool(2)
	defer pool.Close()

	env := DefaultEnv()
	// Skip "md2pdf" in args if present (legacy behavior)
	if len(args) > 0 && args[0] == "md2pdf" {
		args = args[1:]
	}
	flags, positional, err := parseConvertFlags(args)
	if err != nil {
		return err
	}

	// Load config if specified (mirrors runConvertCmd behavior)
	if flags.common.config != "" {
		env.Config, err = config.LoadConfig(flags.common.config)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}

	return runConvert(context.Background(), positional, flags, pool, env)
}

// setupTestDir creates a temp directory with the given file structure.
// Files map paths to content. Returns the temp directory path.
func setupTestDir(t *testing.T, files map[string]string) string {
	t.Helper()
	tempDir := t.TempDir()

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
			t.Fatalf("failed to create dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	return tempDir
}

// assertValidPDFFile verifies that a file exists and contains valid PDF data.
func assertValidPDFFile(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read PDF file %s: %v", path, err)
	}

	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		t.Errorf("file %s does not have PDF magic bytes, got: %q", path, data[:min(10, len(data))])
	}

	if len(data) < 100 {
		t.Errorf("PDF file %s is suspiciously small: %d bytes", path, len(data))
	}
}

func TestBatchConversion_SingleFile(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Hello World",
	})

	inputPath := filepath.Join(tempDir, "doc.md")
	expectedOutput := filepath.Join(tempDir, "doc.pdf")

	err := runIntegration([]string{"md2pdf", inputPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertValidPDFFile(t, expectedOutput)
}

func TestBatchConversion_SingleFileWithOutputFile(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test",
	})

	inputPath := filepath.Join(tempDir, "doc.md")
	outputPath := filepath.Join(tempDir, "custom.pdf")

	err := runIntegration([]string{"md2pdf", "-o", outputPath, inputPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertValidPDFFile(t, outputPath)
}

func TestBatchConversion_SingleFileWithOutputDir(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test",
	})

	inputPath := filepath.Join(tempDir, "doc.md")
	outputDir := filepath.Join(tempDir, "out")
	expectedOutput := filepath.Join(outputDir, "doc.pdf")

	err := runIntegration([]string{"md2pdf", "-o", outputDir, inputPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertValidPDFFile(t, expectedOutput)
}

func TestBatchConversion_Directory(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc1.md":       "# Doc 1",
		"doc2.md":       "# Doc 2",
		"doc3.markdown": "# Doc 3",
		"ignored.txt":   "ignored",
	})

	err := runIntegration([]string{"md2pdf", tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDFs created next to sources (ignoring .txt)
	expectedPDFs := []string{
		filepath.Join(tempDir, "doc1.pdf"),
		filepath.Join(tempDir, "doc2.pdf"),
		filepath.Join(tempDir, "doc3.pdf"),
	}
	for _, pdf := range expectedPDFs {
		assertValidPDFFile(t, pdf)
	}
}

func TestBatchConversion_DirectoryMirror(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc1.md":             "# Doc 1",
		"subdir/doc2.md":      "# Doc 2",
		"subdir/deep/doc3.md": "# Doc 3",
	})

	inputDir := tempDir
	outputDir := filepath.Join(tempDir, "output")

	err := runIntegration([]string{"md2pdf", "-o", outputDir, inputDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mirrored structure
	expectedPDFs := []string{
		filepath.Join(outputDir, "doc1.pdf"),
		filepath.Join(outputDir, "subdir", "doc2.pdf"),
		filepath.Join(outputDir, "subdir", "deep", "doc3.pdf"),
	}
	for _, pdf := range expectedPDFs {
		assertValidPDFFile(t, pdf)
	}
}

func TestBatchConversion_MixedSuccessFailure(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"good.md": "# Good Document\n\nThis should convert successfully.",
		// Empty markdown will cause ErrEmptyMarkdown
		"bad.md": "",
	})

	err := runIntegration([]string{"md2pdf", tempDir})

	// Should return error indicating 1 failure
	if err == nil {
		t.Fatal("expected error for partial failure")
	}

	// Good file should still be converted
	goodPDF := filepath.Join(tempDir, "good.pdf")
	assertValidPDFFile(t, goodPDF)

	// Bad file should not have PDF (empty markdown causes error)
	badPDF := filepath.Join(tempDir, "bad.pdf")
	if _, statErr := os.Stat(badPDF); !os.IsNotExist(statErr) {
		t.Error("bad.pdf should not exist for empty markdown")
	}
}

func TestBatchConversion_EmptyDirectory(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"ignored.txt":  "ignored",
		"ignored.html": "ignored",
	})

	err := runIntegration([]string{"md2pdf", tempDir})

	// Should return error for no markdown files
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestBatchConversion_ConfigDefaultDir(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"input/doc.md": "# From Config",
	})

	// Create config file
	configContent := `input:
  defaultDir: "` + filepath.Join(tempDir, "input") + `"
`
	configPath := filepath.Join(tempDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Run without specifying input, using config
	err := runIntegration([]string{"md2pdf", "--config", configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDF was created
	expectedPDF := filepath.Join(tempDir, "input", "doc.pdf")
	assertValidPDFFile(t, expectedPDF)
}

func TestBatchConversion_CSSPassedToConverter(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc.md":    "# Styled Document\n\nThis document has custom CSS.",
		"style.css": "body { color: blue; }",
	})

	inputPath := filepath.Join(tempDir, "doc.md")
	cssPath := filepath.Join(tempDir, "style.css")
	expectedOutput := filepath.Join(tempDir, "doc.pdf")

	err := runIntegration([]string{"md2pdf", "--style", cssPath, inputPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDF was created (CSS is applied internally, we just verify output exists)
	assertValidPDFFile(t, expectedOutput)
}

func TestBatchConversion_NoInput(t *testing.T) {
	t.Parallel()

	err := runIntegration([]string{"md2pdf"})

	// Should return ErrNoInput
	if !errors.Is(err, ErrNoInput) {
		t.Errorf("expected ErrNoInput, got %v", err)
	}
}

func TestBatchConversion_ConcurrentExecution(t *testing.T) {
	t.Parallel()

	// Create many files to test concurrent processing
	files := make(map[string]string)
	for i := 0; i < concurrentTestFiles; i++ {
		files["doc"+string(rune('A'+i))+".md"] = "# Document " + string(rune('A'+i))
	}
	tempDir := setupTestDir(t, files)

	err := runIntegration([]string{"md2pdf", tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All PDFs should exist and be valid
	for i := 0; i < concurrentTestFiles; i++ {
		pdf := filepath.Join(tempDir, "doc"+string(rune('A'+i))+".pdf")
		assertValidPDFFile(t, pdf)
	}
}

func TestBatchConversion_PageBreaksFlags(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Chapter 1\n\n## Section 1\n\nContent here.\n\n# Chapter 2\n\nMore content.",
	})

	inputPath := filepath.Join(tempDir, "doc.md")
	expectedOutput := filepath.Join(tempDir, "doc.pdf")

	// Test with page break flags
	err := runIntegration([]string{
		"md2pdf",
		"--break-before", "h1,h2",
		"--orphans", "3",
		"--widows", "4",
		inputPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDF was created (page breaks are applied internally)
	assertValidPDFFile(t, expectedOutput)
}

func TestBatchConversion_NoPageBreaksFlag(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test Document\n\nSome content here.",
	})

	// Create config with page breaks enabled
	configContent := `pageBreaks:
  enabled: true
  beforeH1: true
  orphans: 5
`
	configPath := filepath.Join(tempDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	inputPath := filepath.Join(tempDir, "doc.md")
	expectedOutput := filepath.Join(tempDir, "doc.pdf")

	// Test --no-page-breaks overrides config
	err := runIntegration([]string{
		"md2pdf",
		"--config", configPath,
		"--no-page-breaks",
		inputPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDF was created
	assertValidPDFFile(t, expectedOutput)
}

func TestBatchConversion_PageBreaksFromConfig(t *testing.T) {
	t.Parallel()

	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test Document\n\nSome content here.",
	})

	// Create config with page breaks settings
	configContent := `pageBreaks:
  enabled: true
  beforeH1: true
  beforeH2: false
  beforeH3: true
  orphans: 4
  widows: 5
`
	configPath := filepath.Join(tempDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	inputPath := filepath.Join(tempDir, "doc.md")
	expectedOutput := filepath.Join(tempDir, "doc.pdf")

	err := runIntegration([]string{"md2pdf", "--config", configPath, inputPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDF was created (config settings are applied internally)
	assertValidPDFFile(t, expectedOutput)
}

func TestIntegration_AuthorInfoDRY(t *testing.T) {
	t.Parallel()

	t.Run("author config flows to cover and signature", func(t *testing.T) {
		t.Parallel()

		tempDir := setupTestDir(t, map[string]string{
			"doc.md": "# Test Document\n\nSome content here.",
		})

		// Create config with author info
		configContent := `author:
  name: "John Doe"
  title: "Senior Developer"
  email: "john@example.com"
  organization: "Acme Corp"
cover:
  enabled: true
signature:
  enabled: true
`
		configPath := filepath.Join(tempDir, "test.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		inputPath := filepath.Join(tempDir, "doc.md")
		expectedOutput := filepath.Join(tempDir, "doc.pdf")

		err := runIntegration([]string{"md2pdf", "--config", configPath, inputPath})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify PDF was created with cover and signature (internal behavior)
		assertValidPDFFile(t, expectedOutput)
	})

	t.Run("CLI author flags override config in both cover and signature", func(t *testing.T) {
		t.Parallel()

		tempDir := setupTestDir(t, map[string]string{
			"doc.md": "# Test Document\n\nSome content here.",
		})

		// Create config with author info
		configContent := `author:
  name: "Config Author"
  title: "Config Title"
cover:
  enabled: true
signature:
  enabled: true
`
		configPath := filepath.Join(tempDir, "test.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		inputPath := filepath.Join(tempDir, "doc.md")
		expectedOutput := filepath.Join(tempDir, "doc.pdf")

		err := runIntegration([]string{
			"md2pdf",
			"--config", configPath,
			"--author-name", "CLI Author",
			"--author-title", "CLI Title",
			inputPath,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify PDF was created (CLI flags override internal behavior)
		assertValidPDFFile(t, expectedOutput)
	})
}

func TestIntegration_DocumentInfoDRY(t *testing.T) {
	t.Parallel()

	t.Run("document config flows to cover and footer", func(t *testing.T) {
		t.Parallel()

		tempDir := setupTestDir(t, map[string]string{
			"doc.md": "# Test Document\n\nSome content here.",
		})

		// Create config with document info
		configContent := `document:
  title: "Document Title"
  subtitle: "A Comprehensive Guide"
  version: "v1.0.0"
  date: "2025-06-15"
cover:
  enabled: true
footer:
  enabled: true
`
		configPath := filepath.Join(tempDir, "test.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		inputPath := filepath.Join(tempDir, "doc.md")
		expectedOutput := filepath.Join(tempDir, "doc.pdf")

		err := runIntegration([]string{"md2pdf", "--config", configPath, inputPath})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify PDF was created with document info in cover and footer
		assertValidPDFFile(t, expectedOutput)
	})

	t.Run("CLI document flags override config in cover and footer", func(t *testing.T) {
		t.Parallel()

		tempDir := setupTestDir(t, map[string]string{
			"doc.md": "# Test Document\n\nSome content here.",
		})

		// Create config with document info
		configContent := `document:
  version: "v1.0"
  date: "2025-01-01"
cover:
  enabled: true
footer:
  enabled: true
`
		configPath := filepath.Join(tempDir, "test.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		inputPath := filepath.Join(tempDir, "doc.md")
		expectedOutput := filepath.Join(tempDir, "doc.pdf")

		err := runIntegration([]string{
			"md2pdf",
			"--config", configPath,
			"--doc-version", "v2.0",
			"--doc-date", "2025-12-31",
			inputPath,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify PDF was created (CLI flags override internal behavior)
		assertValidPDFFile(t, expectedOutput)
	})
}

func TestIntegration_NewCLIFlags(t *testing.T) {
	t.Parallel()

	t.Run("watermark shorthand flags work", func(t *testing.T) {
		t.Parallel()

		tempDir := setupTestDir(t, map[string]string{
			"doc.md": "# Test Document\n\nSome content here.",
		})

		inputPath := filepath.Join(tempDir, "doc.md")
		expectedOutput := filepath.Join(tempDir, "doc.pdf")

		err := runIntegration([]string{
			"md2pdf",
			"--wm-text", "DRAFT",
			"--wm-color", "#ff0000",
			"--wm-opacity", "0.3",
			"--wm-angle", "-30",
			inputPath,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify PDF was created with watermark
		assertValidPDFFile(t, expectedOutput)
	})

	t.Run("footer flags work", func(t *testing.T) {
		t.Parallel()

		tempDir := setupTestDir(t, map[string]string{
			"doc.md": "# Test Document\n\nSome content here.",
		})

		inputPath := filepath.Join(tempDir, "doc.md")
		expectedOutput := filepath.Join(tempDir, "doc.pdf")

		err := runIntegration([]string{
			"md2pdf",
			"--footer-position", "left",
			"--footer-text", "Confidential",
			"--footer-page-number",
			inputPath,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify PDF was created with footer
		assertValidPDFFile(t, expectedOutput)
	})

	t.Run("toc flags work", func(t *testing.T) {
		t.Parallel()

		tempDir := setupTestDir(t, map[string]string{
			"doc.md": "# Chapter 1\n\nContent.\n\n## Section 1.1\n\nMore content.\n\n# Chapter 2\n\nEven more.",
		})

		// Create config with TOC enabled
		configContent := `toc:
  enabled: true
`
		configPath := filepath.Join(tempDir, "test.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		inputPath := filepath.Join(tempDir, "doc.md")
		expectedOutput := filepath.Join(tempDir, "doc.pdf")

		err := runIntegration([]string{
			"md2pdf",
			"--config", configPath,
			"--toc-title", "Contents",
			"--toc-depth", "4",
			inputPath,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify PDF was created with TOC
		assertValidPDFFile(t, expectedOutput)
	})
}
