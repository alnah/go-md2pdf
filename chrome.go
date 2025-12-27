package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// PDFConverter abstracts HTML to PDF conversion to allow different backends.
type PDFConverter interface {
	ToPDF(htmlContent, cssContent, outputPath string) error
}

// Sentinel errors for PDF conversion failures.
var (
	ErrEmptyHTML       = errors.New("HTML content cannot be empty")
	ErrEmptyOutputPath = errors.New("output path cannot be empty")
	ErrPDFGeneration   = errors.New("PDF generation failed")
	ErrWritePDF        = errors.New("failed to write PDF file")
)

// PDF page dimensions in inches (US Letter format).
const (
	paperWidthInches  = 8.5
	paperHeightInches = 11
	marginInches      = 0.5
	defaultTimeout    = 30 * time.Second
)

// ChromeConverter converts HTML to PDF using headless Chrome via chromedp.
type ChromeConverter struct {
	Timeout time.Duration
}

// NewChromeConverter creates a ChromeConverter with default settings.
func NewChromeConverter() *ChromeConverter {
	return &ChromeConverter{
		Timeout: defaultTimeout,
	}
}

// ToPDF converts HTML content to a PDF file using headless Chrome.
// Uses US Letter format (8.5x11 inches) with 0.5 inch margins.
func (c *ChromeConverter) ToPDF(htmlContent, cssContent, outputPath string) error {
	if err := validateToPDFInputs(htmlContent, outputPath); err != nil {
		return err
	}

	fullHTML := InjectCSS(htmlContent, cssContent)

	tmpPath, cleanup, err := writeHTMLToTempFile(fullHTML)
	if err != nil {
		return err
	}
	defer cleanup()

	pdfBuf, err := c.renderPDFFromFile(tmpPath)
	if err != nil {
		return err
	}

	// #nosec G306 -- PDF output files are intended to be readable
	if err := os.WriteFile(outputPath, pdfBuf, 0o644); err != nil {
		return fmt.Errorf("%w: %v", ErrWritePDF, err)
	}

	return nil
}

// writeHTMLToTempFile creates a temporary file with HTML content.
// Returns the file path and a cleanup function to remove the file.
func writeHTMLToTempFile(html string) (path string, cleanup func(), err error) {
	tmpFile, err := os.CreateTemp("", "go-md2pdf-*.html")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}

	path = tmpFile.Name()
	cleanup = func() { _ = os.Remove(path) }

	if _, err := tmpFile.WriteString(html); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("closing temp file: %w", err)
	}

	return path, cleanup, nil
}

// renderPDFFromFile opens a local HTML file in headless Chrome and renders it to PDF.
func (c *ChromeConverter) renderPDFFromFile(filePath string) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancelTimeout := context.WithTimeout(ctx, c.Timeout)
	defer cancelTimeout()

	var pdfBuf []byte

	err := chromedp.Run(ctx,
		chromedp.Navigate("file://"+filePath),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPaperWidth(paperWidthInches).
				WithPaperHeight(paperHeightInches).
				WithMarginTop(marginInches).
				WithMarginBottom(marginInches).
				WithMarginLeft(marginInches).
				WithMarginRight(marginInches).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPDFGeneration, err)
	}

	return pdfBuf, nil
}

// validateToPDFInputs checks that required inputs are non-empty.
func validateToPDFInputs(htmlContent, outputPath string) error {
	if htmlContent == "" {
		return ErrEmptyHTML
	}
	if outputPath == "" {
		return ErrEmptyOutputPath
	}
	return nil
}
