package main

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// mockConverter is a test double for the Converter interface.
type mockConverter struct {
	mu          sync.Mutex
	calls       []ConversionOptions
	convertFunc func(opts ConversionOptions) error
}

func newMockConverter() *mockConverter {
	return &mockConverter{}
}

func (m *mockConverter) Convert(opts ConversionOptions) error {
	m.mu.Lock()
	m.calls = append(m.calls, opts)
	m.mu.Unlock()

	if m.convertFunc != nil {
		return m.convertFunc(opts)
	}

	// Default: simulate success by creating a minimal PDF file
	return os.WriteFile(opts.OutputPath, []byte("%PDF-1.4 mock"), 0644)
}

func (m *mockConverter) getCalls() []ConversionOptions {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ConversionOptions{}, m.calls...)
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

func TestBatchConversion_SingleFile(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Hello World",
	})

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")
	expectedOutput := filepath.Join(tempDir, "doc.pdf")

	err := run([]string{"go-md2pdf", inputPath}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDF was created
	if _, err := os.Stat(expectedOutput); os.IsNotExist(err) {
		t.Error("expected PDF file was not created")
	}

	// Verify converter was called with correct content
	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].MarkdownContent != "# Hello World" {
		t.Errorf("MarkdownContent = %q, want %q", calls[0].MarkdownContent, "# Hello World")
	}
	if calls[0].OutputPath != expectedOutput {
		t.Errorf("OutputPath = %q, want %q", calls[0].OutputPath, expectedOutput)
	}
}

func TestBatchConversion_SingleFileWithOutputFile(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test",
	})

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")
	outputPath := filepath.Join(tempDir, "custom.pdf")

	err := run([]string{"go-md2pdf", "-o", outputPath, inputPath}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDF was created at custom path
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected PDF file was not created at custom path")
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].OutputPath != outputPath {
		t.Errorf("OutputPath = %q, want %q", calls[0].OutputPath, outputPath)
	}
}

func TestBatchConversion_SingleFileWithOutputDir(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test",
	})

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")
	outputDir := filepath.Join(tempDir, "out")
	expectedOutput := filepath.Join(outputDir, "doc.pdf")

	err := run([]string{"go-md2pdf", "-o", outputDir, inputPath}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PDF was created in output directory
	if _, err := os.Stat(expectedOutput); os.IsNotExist(err) {
		t.Error("expected PDF file was not created in output directory")
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].OutputPath != expectedOutput {
		t.Errorf("OutputPath = %q, want %q", calls[0].OutputPath, expectedOutput)
	}
}

func TestBatchConversion_Directory(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc1.md":       "# Doc 1",
		"doc2.md":       "# Doc 2",
		"doc3.markdown": "# Doc 3",
		"ignored.txt":   "ignored",
	})

	mock := newMockConverter()

	err := run([]string{"go-md2pdf", tempDir}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify 3 conversions happened (ignoring .txt)
	calls := mock.getCalls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(calls))
	}

	// Verify PDFs created next to sources
	expectedPDFs := []string{
		filepath.Join(tempDir, "doc1.pdf"),
		filepath.Join(tempDir, "doc2.pdf"),
		filepath.Join(tempDir, "doc3.pdf"),
	}
	for _, pdf := range expectedPDFs {
		if _, err := os.Stat(pdf); os.IsNotExist(err) {
			t.Errorf("expected PDF %s was not created", pdf)
		}
	}
}

func TestBatchConversion_DirectoryMirror(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc1.md":             "# Doc 1",
		"subdir/doc2.md":      "# Doc 2",
		"subdir/deep/doc3.md": "# Doc 3",
	})

	mock := newMockConverter()
	inputDir := tempDir
	outputDir := filepath.Join(tempDir, "output")

	err := run([]string{"go-md2pdf", "-o", outputDir, inputDir}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify 3 conversions happened
	calls := mock.getCalls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(calls))
	}

	// Verify mirrored structure
	expectedPDFs := []string{
		filepath.Join(outputDir, "doc1.pdf"),
		filepath.Join(outputDir, "subdir", "doc2.pdf"),
		filepath.Join(outputDir, "subdir", "deep", "doc3.pdf"),
	}
	for _, pdf := range expectedPDFs {
		if _, err := os.Stat(pdf); os.IsNotExist(err) {
			t.Errorf("expected mirrored PDF %s was not created", pdf)
		}
	}
}

func TestBatchConversion_MixedSuccessFailure(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"good.md": "# Good",
		"bad.md":  "# Bad",
	})

	mock := newMockConverter()

	// Make converter fail for bad.md
	mock.convertFunc = func(opts ConversionOptions) error {
		if opts.OutputPath == filepath.Join(tempDir, "bad.pdf") {
			return errors.New("simulated conversion failure")
		}
		return os.WriteFile(opts.OutputPath, []byte("%PDF-1.4 mock"), 0644)
	}

	err := run([]string{"go-md2pdf", tempDir}, mock)

	// Should return error indicating 1 failure
	if err == nil {
		t.Fatal("expected error for partial failure")
	}

	// Good file should still be converted
	goodPDF := filepath.Join(tempDir, "good.pdf")
	if _, err := os.Stat(goodPDF); os.IsNotExist(err) {
		t.Error("good.pdf should have been created despite bad.md failure")
	}

	// Bad file should not have PDF
	badPDF := filepath.Join(tempDir, "bad.pdf")
	if _, err := os.Stat(badPDF); !os.IsNotExist(err) {
		t.Error("bad.pdf should not exist")
	}

	// Verify both files were attempted
	calls := mock.getCalls()
	if len(calls) != 2 {
		t.Errorf("expected 2 conversion attempts, got %d", len(calls))
	}
}

func TestBatchConversion_EmptyDirectory(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"ignored.txt":  "ignored",
		"ignored.html": "ignored",
	})

	mock := newMockConverter()

	err := run([]string{"go-md2pdf", tempDir}, mock)

	// Should return error for no markdown files
	if err == nil {
		t.Fatal("expected error for empty directory")
	}

	// No conversions should have been attempted
	calls := mock.getCalls()
	if len(calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(calls))
	}
}

func TestBatchConversion_ConfigDefaultDir(t *testing.T) {
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

	mock := newMockConverter()

	// Run without specifying input, using config
	err := run([]string{"go-md2pdf", "--config", configPath}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify conversion happened
	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].MarkdownContent != "# From Config" {
		t.Errorf("MarkdownContent = %q, want %q", calls[0].MarkdownContent, "# From Config")
	}
}

func TestBatchConversion_CSSPassedToConverter(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md":    "# Test",
		"style.css": "body { color: blue; }",
	})

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")
	cssPath := filepath.Join(tempDir, "style.css")

	err := run([]string{"go-md2pdf", "--css", cssPath, inputPath}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].CSSContent != "body { color: blue; }" {
		t.Errorf("CSSContent = %q, want %q", calls[0].CSSContent, "body { color: blue; }")
	}
}

func TestBatchConversion_NoInput(t *testing.T) {
	mock := newMockConverter()

	err := run([]string{"go-md2pdf"}, mock)

	// Should return ErrNoInput
	if !errors.Is(err, ErrNoInput) {
		t.Errorf("expected ErrNoInput, got %v", err)
	}

	// No conversions should have been attempted
	calls := mock.getCalls()
	if len(calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(calls))
	}
}

func TestBatchConversion_ConcurrentExecution(t *testing.T) {
	// Create many files to test concurrent processing
	files := make(map[string]string)
	for i := 0; i < 20; i++ {
		files[filepath.Join("doc"+string(rune('A'+i))+".md")] = "# Doc"
	}
	tempDir := setupTestDir(t, files)

	mock := newMockConverter()

	err := run([]string{"go-md2pdf", tempDir}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All files should be converted
	calls := mock.getCalls()
	if len(calls) != 20 {
		t.Errorf("expected 20 calls, got %d", len(calls))
	}

	// All PDFs should exist
	for i := 0; i < 20; i++ {
		pdf := filepath.Join(tempDir, "doc"+string(rune('A'+i))+".pdf")
		if _, err := os.Stat(pdf); os.IsNotExist(err) {
			t.Errorf("expected PDF %s was not created", pdf)
		}
	}
}
