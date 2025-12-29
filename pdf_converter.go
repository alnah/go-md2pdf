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
	ToPDF(htmlContent, outputPath string) error
}

// PDFRenderer abstracts PDF rendering from an HTML file to enable testing without Chrome.
type PDFRenderer interface {
	RenderFromFile(filePath string) ([]byte, error)
}

// Sentinel errors for PDF conversion failures.
var (
	ErrPDFGeneration = errors.New("PDF generation failed")
	ErrWritePDF      = errors.New("failed to write PDF file")
)

// PDF page dimensions in inches (US Letter format).
const (
	paperWidthInches  = 8.5
	paperHeightInches = 11
	marginInches      = 0.5
	defaultTimeout    = 30 * time.Second
)

// ChromeDPRenderer implements PDFRenderer using chromedp.
type ChromeDPRenderer struct {
	Timeout time.Duration
}

// RenderFromFile opens a local HTML file in headless Chrome and renders it to PDF.
func (r *ChromeDPRenderer) RenderFromFile(filePath string) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancelTimeout := context.WithTimeout(ctx, r.Timeout)
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

// ChromeConverter converts HTML to PDF using headless Chrome via chromedp.
type ChromeConverter struct {
	Renderer PDFRenderer
}

// NewChromeConverter creates a ChromeConverter with production renderer.
func NewChromeConverter() *ChromeConverter {
	return &ChromeConverter{
		Renderer: &ChromeDPRenderer{Timeout: defaultTimeout},
	}
}

// NewChromeConverterWith creates a ChromeConverter with custom renderer (for testing).
func NewChromeConverterWith(renderer PDFRenderer) *ChromeConverter {
	if renderer == nil {
		panic("nil PDFRenderer in NewChromeConverterWith")
	}
	return &ChromeConverter{Renderer: renderer}
}

// ToPDF converts HTML content to a PDF file using headless Chrome.
// Uses US Letter format (8.5x11 inches) with 0.5 inch margins.
func (c *ChromeConverter) ToPDF(htmlContent, outputPath string) error {
	tmpPath, cleanup, err := writeTempFile(htmlContent, "html")
	if err != nil {
		return err
	}
	defer cleanup()

	pdfBuf, err := c.Renderer.RenderFromFile(tmpPath)
	if err != nil {
		return err
	}

	// #nosec G306 -- PDF output files are intended to be readable
	if err := os.WriteFile(outputPath, pdfBuf, 0o644); err != nil {
		return fmt.Errorf("%w: %v", ErrWritePDF, err)
	}

	return nil
}
