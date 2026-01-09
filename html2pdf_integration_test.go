//go:build integration

package md2pdf

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func assertValidPDF(t *testing.T, data []byte) {
	t.Helper()

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Errorf("data does not have PDF magic bytes, got prefix: %q", data[:min(10, len(data))])
	}

	if len(data) < 100 {
		t.Errorf("PDF data suspiciously small: %d bytes", len(data))
	}
}

func assertValidPDFFile(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read PDF file: %v", err)
	}

	assertValidPDF(t, data)
}

// TestRodConverter_ToPDF_Integration tests PDF generation using go-rod.
// Rod automatically downloads Chromium on first run if not found.
func TestRodConverter_ToPDF_Integration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("valid HTML produces PDF", func(t *testing.T) {
		t.Parallel()
		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello, World!</h1><p>This is a test document.</p></body>
</html>`

		converter := newRodConverter(defaultTimeout)
		data, err := converter.ToPDF(ctx, html, nil)
		if err != nil {
			t.Fatalf("ToPDF() error = %v", err)
		}

		assertValidPDF(t, data)
	})

	t.Run("HTML with CSS produces PDF", func(t *testing.T) {
		t.Parallel()

		// CSS is now injected before calling ToPDF
		injector := &cssInjection{}
		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Styled Document</h1></body>
</html>`
		css := "h1 { color: blue; font-size: 24px; }"
		htmlWithCSS := injector.InjectCSS(ctx, html, css)

		converter := newRodConverter(defaultTimeout)
		data, err := converter.ToPDF(ctx, htmlWithCSS, nil)
		if err != nil {
			t.Fatalf("ToPDF() error = %v", err)
		}

		assertValidPDF(t, data)
	})

	t.Run("HTML with footer produces PDF", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Document with Footer</h1></body>
</html>`

		converter := newRodConverter(defaultTimeout)
		opts := &pdfOptions{
			Footer: &footerData{
				ShowPageNumber: true,
				Date:           "2025-01-15",
				Status:         "DRAFT",
			},
		}
		data, err := converter.ToPDF(ctx, html, opts)
		if err != nil {
			t.Fatalf("ToPDF() error = %v", err)
		}

		assertValidPDF(t, data)
	})
}

// TestService_Integration tests the full conversion pipeline through the public API.
func TestService_Integration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("basic markdown to PDF", func(t *testing.T) {
		t.Parallel()

		service := acquireService(t)
		input := Input{
			Markdown: "# Hello\n\nWorld",
		}

		data, err := service.Convert(ctx, input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		assertValidPDF(t, data)
	})

	t.Run("markdown with CSS", func(t *testing.T) {
		t.Parallel()

		service := acquireService(t)
		input := Input{
			Markdown: "# Styled\n\nContent",
			CSS:      "h1 { color: blue; }",
		}

		data, err := service.Convert(ctx, input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		assertValidPDF(t, data)
	})

	t.Run("markdown with footer", func(t *testing.T) {
		t.Parallel()

		service := acquireService(t)
		input := Input{
			Markdown: "# Document\n\nWith footer",
			Footer: &Footer{
				Position:       "center",
				ShowPageNumber: true,
				Date:           "2025-01-15",
				Status:         "DRAFT",
			},
		}

		data, err := service.Convert(ctx, input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		assertValidPDF(t, data)
	})

	t.Run("markdown with signature", func(t *testing.T) {
		t.Parallel()

		service := acquireService(t)
		input := Input{
			Markdown: "# Document\n\nWith signature",
			Signature: &Signature{
				Name:  "John Doe",
				Title: "Developer",
				Email: "john@example.com",
				Links: []Link{
					{Label: "GitHub", URL: "https://github.com/johndoe"},
				},
			},
		}

		data, err := service.Convert(ctx, input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		assertValidPDF(t, data)
	})

	t.Run("write to file", func(t *testing.T) {
		t.Parallel()

		service := acquireService(t)
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.pdf")

		input := Input{
			Markdown: "# Test\n\nWriting to file",
		}

		data, err := service.Convert(ctx, input)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}

		err = os.WriteFile(outputPath, data, 0644)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		assertValidPDFFile(t, outputPath)
	})
}

// TestRodRenderer_EnsureBrowser_CI tests browser launch with CI environment variable.
func TestRodRenderer_EnsureBrowser_CI(t *testing.T) {
	t.Setenv("CI", "true")

	renderer := newRodRenderer(testTimeout)
	defer renderer.Close()

	err := renderer.ensureBrowser()
	if err != nil {
		t.Fatalf("ensureBrowser() with CI=true error = %v", err)
	}

	if renderer.browser == nil {
		t.Error("browser should not be nil after ensureBrowser()")
	}
}

// TestRodRenderer_RenderFromFile_ContextCancelled tests early exit on cancelled context.
func TestRodRenderer_RenderFromFile_ContextCancelled(t *testing.T) {
	t.Parallel()

	renderer := newRodRenderer(testTimeout)
	defer renderer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := renderer.RenderFromFile(ctx, "/tmp/nonexistent.html", nil)

	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestRodRenderer_RenderFromFile_ContextDeadlineExceeded tests early exit on expired deadline.
func TestRodRenderer_RenderFromFile_ContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	renderer := newRodRenderer(testTimeout)
	defer renderer.Close()

	// Context with already-passed deadline
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, err := renderer.RenderFromFile(ctx, "/tmp/nonexistent.html", nil)

	if err == nil {
		t.Fatal("expected error for expired deadline, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}
