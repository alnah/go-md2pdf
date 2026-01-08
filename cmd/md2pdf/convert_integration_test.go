package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	md2pdf "github.com/alnah/go-md2pdf"
)

// mockConverter is a test double for the Converter interface.
type mockConverter struct {
	mu          sync.Mutex
	calls       []md2pdf.Input
	convertFunc func(ctx context.Context, input md2pdf.Input) ([]byte, error)
}

func newMockConverter() *mockConverter {
	return &mockConverter{}
}

func (m *mockConverter) Convert(ctx context.Context, input md2pdf.Input) ([]byte, error) {
	m.mu.Lock()
	m.calls = append(m.calls, input)
	m.mu.Unlock()

	if m.convertFunc != nil {
		return m.convertFunc(ctx, input)
	}

	// Default: return mock PDF bytes
	return []byte("%PDF-1.4 mock"), nil
}

func (m *mockConverter) Close() error {
	return nil
}

func (m *mockConverter) getCalls() []md2pdf.Input {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]md2pdf.Input{}, m.calls...)
}

// testPool creates a ServicePool for testing that uses a mock converter.
// It wraps the mock in a way that convertBatch can use it.
type testPool struct {
	mock   *mockConverter
	sem    chan Converter
	size   int
	mu     sync.Mutex
	closed bool
}

func newTestPool(mock *mockConverter, size int) *testPool {
	if size < 1 {
		size = 1
	}
	p := &testPool{
		mock: mock,
		sem:  make(chan Converter, size),
		size: size,
	}
	// Fill the semaphore with the mock converter
	for i := 0; i < size; i++ {
		p.sem <- mock
	}
	return p
}

func (p *testPool) Acquire() Converter {
	return <-p.sem
}

func (p *testPool) Release(c Converter) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()
	// Send outside lock to avoid deadlock: if channel is full,
	// holding the lock would prevent Close() from running.
	p.sem <- c
}

func (p *testPool) Close() error {
	p.mu.Lock()
	p.closed = true
	close(p.sem)
	p.mu.Unlock()
	return nil
}

func (p *testPool) Size() int {
	return p.size
}

// runWithTestPool is a test helper that runs with a test pool.
func runWithTestPool(args []string, mock *mockConverter) error {
	pool := newTestPool(mock, 2)
	defer pool.Close()
	return run(context.Background(), args, pool)
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

	err := runWithTestPool([]string{"md2pdf", inputPath}, mock)
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
	if calls[0].Markdown != "# Hello World" {
		t.Errorf("Markdown = %q, want %q", calls[0].Markdown, "# Hello World")
	}
}

func TestBatchConversion_SingleFileWithOutputFile(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test",
	})

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")
	outputPath := filepath.Join(tempDir, "custom.pdf")

	err := runWithTestPool([]string{"md2pdf", "-o", outputPath, inputPath}, mock)
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
}

func TestBatchConversion_SingleFileWithOutputDir(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test",
	})

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")
	outputDir := filepath.Join(tempDir, "out")
	expectedOutput := filepath.Join(outputDir, "doc.pdf")

	err := runWithTestPool([]string{"md2pdf", "-o", outputDir, inputPath}, mock)
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
}

func TestBatchConversion_Directory(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc1.md":       "# Doc 1",
		"doc2.md":       "# Doc 2",
		"doc3.markdown": "# Doc 3",
		"ignored.txt":   "ignored",
	})

	mock := newMockConverter()

	err := runWithTestPool([]string{"md2pdf", tempDir}, mock)
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

	err := runWithTestPool([]string{"md2pdf", "-o", outputDir, inputDir}, mock)
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
	mock.convertFunc = func(ctx context.Context, input md2pdf.Input) ([]byte, error) {
		if input.Markdown == "# Bad" {
			return nil, errors.New("simulated conversion failure")
		}
		return []byte("%PDF-1.4 mock"), nil
	}

	err := runWithTestPool([]string{"md2pdf", tempDir}, mock)

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

	err := runWithTestPool([]string{"md2pdf", tempDir}, mock)

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
	err := runWithTestPool([]string{"md2pdf", "--config", configPath}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify conversion happened
	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Markdown != "# From Config" {
		t.Errorf("Markdown = %q, want %q", calls[0].Markdown, "# From Config")
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

	err := runWithTestPool([]string{"md2pdf", "--css", cssPath, inputPath}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].CSS != "body { color: blue; }" {
		t.Errorf("CSS = %q, want %q", calls[0].CSS, "body { color: blue; }")
	}
}

func TestBatchConversion_NoInput(t *testing.T) {
	mock := newMockConverter()

	err := runWithTestPool([]string{"md2pdf"}, mock)

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

	err := runWithTestPool([]string{"md2pdf", tempDir}, mock)
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

func TestBatchConversion_PageBreaksFlags(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Chapter 1\n\n## Section 1\n\nContent here.\n\n# Chapter 2\n\nMore content.",
	})

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")

	// Test with page break flags
	err := runWithTestPool([]string{
		"md2pdf",
		"--break-before", "h1,h2",
		"--orphans", "3",
		"--widows", "4",
		inputPath,
	}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	// Verify page breaks settings were passed to converter
	if calls[0].PageBreaks == nil {
		t.Fatal("expected PageBreaks to be set")
	}
	if !calls[0].PageBreaks.BeforeH1 {
		t.Error("expected BeforeH1 = true")
	}
	if !calls[0].PageBreaks.BeforeH2 {
		t.Error("expected BeforeH2 = true")
	}
	if calls[0].PageBreaks.BeforeH3 {
		t.Error("expected BeforeH3 = false")
	}
	if calls[0].PageBreaks.Orphans != 3 {
		t.Errorf("Orphans = %d, want 3", calls[0].PageBreaks.Orphans)
	}
	if calls[0].PageBreaks.Widows != 4 {
		t.Errorf("Widows = %d, want 4", calls[0].PageBreaks.Widows)
	}
}

func TestBatchConversion_NoPageBreaksFlag(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test",
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

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")

	// Test --no-page-breaks overrides config
	err := runWithTestPool([]string{
		"md2pdf",
		"--config", configPath,
		"--no-page-breaks",
		inputPath,
	}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	// Verify page breaks were disabled (nil)
	if calls[0].PageBreaks != nil {
		t.Errorf("expected PageBreaks to be nil when --no-page-breaks used, got %+v", calls[0].PageBreaks)
	}
}

func TestBatchConversion_PageBreaksFromConfig(t *testing.T) {
	tempDir := setupTestDir(t, map[string]string{
		"doc.md": "# Test",
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

	mock := newMockConverter()
	inputPath := filepath.Join(tempDir, "doc.md")

	err := runWithTestPool([]string{"md2pdf", "--config", configPath, inputPath}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	// Verify config settings were applied
	if calls[0].PageBreaks == nil {
		t.Fatal("expected PageBreaks to be set from config")
	}
	if !calls[0].PageBreaks.BeforeH1 {
		t.Error("expected BeforeH1 = true from config")
	}
	if calls[0].PageBreaks.BeforeH2 {
		t.Error("expected BeforeH2 = false from config")
	}
	if !calls[0].PageBreaks.BeforeH3 {
		t.Error("expected BeforeH3 = true from config")
	}
	if calls[0].PageBreaks.Orphans != 4 {
		t.Errorf("Orphans = %d, want 4 from config", calls[0].PageBreaks.Orphans)
	}
	if calls[0].PageBreaks.Widows != 5 {
		t.Errorf("Widows = %d, want 5 from config", calls[0].PageBreaks.Widows)
	}
}
