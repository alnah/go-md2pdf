//go:build integration

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func assertValidPDF(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read PDF file: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Errorf("file does not have PDF magic bytes, got prefix: %q", data[:min(10, len(data))])
	}

	if len(data) < 100 {
		t.Errorf("PDF file suspiciously small: %d bytes", len(data))
	}
}

// TestRodConverter_ToPDF_Integration tests PDF generation using go-rod.
// Rod automatically downloads Chromium on first run if not found.
func TestRodConverter_ToPDF_Integration(t *testing.T) {
	t.Run("valid HTML produces PDF", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.pdf")

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello, World!</h1><p>This is a test document.</p></body>
</html>`

		converter := NewRodConverter()
		err := converter.ToPDF(html, outputPath, nil)
		if err != nil {
			t.Fatalf("ToPDF() error = %v", err)
		}

		assertValidPDF(t, outputPath)
	})

	t.Run("HTML with CSS produces PDF", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.pdf")

		// CSS is now injected before calling ToPDF
		injector := &CSSInjection{}
		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Styled Document</h1></body>
</html>`
		css := "h1 { color: blue; font-size: 24px; }"
		htmlWithCSS := injector.InjectCSS(html, css)

		converter := NewRodConverter()
		err := converter.ToPDF(htmlWithCSS, outputPath, nil)
		if err != nil {
			t.Fatalf("ToPDF() error = %v", err)
		}

		assertValidPDF(t, outputPath)
	})

	t.Run("invalid output directory returns error", func(t *testing.T) {
		converter := NewRodConverter()
		err := converter.ToPDF("<html></html>", "/nonexistent/directory/output.pdf", nil)
		if !errors.Is(err, ErrWritePDF) {
			t.Errorf("ToPDF() error = %v, want wrapped %v", err, ErrWritePDF)
		}
	})
}
