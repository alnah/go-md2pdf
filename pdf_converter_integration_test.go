//go:build integration

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireChrome(t *testing.T) {
	t.Helper()

	// chromedp looks for these binaries in order
	chromePaths := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	}

	for _, p := range chromePaths {
		if _, err := exec.LookPath(p); err == nil {
			return
		}
	}

	// Check macOS app bundle directly
	if _, err := os.Stat("/Applications/Google Chrome.app"); err == nil {
		return
	}

	t.Fatal("Chrome not found. Install Chrome or Chromium to run integration tests.")
}

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

func TestChromeConverter_ToPDF_Integration(t *testing.T) {
	requireChrome(t)

	t.Run("valid HTML produces PDF", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.pdf")

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello, World!</h1><p>This is a test document.</p></body>
</html>`

		converter := NewChromeConverter()
		err := converter.ToPDF(html, outputPath)
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

		converter := NewChromeConverter()
		err := converter.ToPDF(htmlWithCSS, outputPath)
		if err != nil {
			t.Fatalf("ToPDF() error = %v", err)
		}

		assertValidPDF(t, outputPath)
	})

	t.Run("empty HTML returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.pdf")

		converter := NewChromeConverter()
		err := converter.ToPDF("", outputPath)
		if !errors.Is(err, ErrEmptyHTML) {
			t.Errorf("ToPDF() error = %v, want %v", err, ErrEmptyHTML)
		}
	})

	t.Run("empty output path returns error", func(t *testing.T) {
		converter := NewChromeConverter()
		err := converter.ToPDF("<html></html>", "")
		if !errors.Is(err, ErrEmptyOutputPath) {
			t.Errorf("ToPDF() error = %v, want %v", err, ErrEmptyOutputPath)
		}
	})

	t.Run("invalid output directory returns error", func(t *testing.T) {
		converter := NewChromeConverter()
		err := converter.ToPDF("<html></html>", "/nonexistent/directory/output.pdf")
		if !errors.Is(err, ErrWritePDF) {
			t.Errorf("ToPDF() error = %v, want wrapped %v", err, ErrWritePDF)
		}
	})
}
