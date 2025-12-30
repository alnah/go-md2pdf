package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// PDFConverter abstracts HTML to PDF conversion to allow different backends.
type PDFConverter interface {
	ToPDF(htmlContent, outputPath string) error
}

// PDFRenderer abstracts PDF rendering from an HTML file to enable testing without a browser.
type PDFRenderer interface {
	RenderFromFile(filePath string) ([]byte, error)
}

// Sentinel errors for PDF conversion failures.
var (
	ErrBrowserConnect = errors.New("failed to connect to browser")
	ErrPageCreate     = errors.New("failed to create browser page")
	ErrPageLoad       = errors.New("failed to load page")
	ErrPDFGeneration  = errors.New("PDF generation failed")
	ErrWritePDF       = errors.New("failed to write PDF file")
)

// PDF page dimensions in inches (US Letter format).
const (
	paperWidthInches  = 8.5
	paperHeightInches = 11
	marginInches      = 0.5
	defaultTimeout    = 30 * time.Second
)

// RodRenderer implements PDFRenderer using go-rod.
// Rod automatically downloads Chromium on first run if not found.
type RodRenderer struct {
	Timeout time.Duration
}

// RenderFromFile opens a local HTML file in headless Chrome and renders it to PDF.
// Returns explicit errors instead of panicking when browser operations fail.
func (r *RodRenderer) RenderFromFile(filePath string) ([]byte, error) {
	browser := rod.New()
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{URL: "file://" + filePath})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPageCreate, err)
	}
	defer page.Close()

	// Wait for page to load with timeout
	if err := page.Timeout(r.Timeout).WaitLoad(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPageLoad, err)
	}

	// Generate PDF with US Letter format and margins
	reader, err := page.PDF(&proto.PagePrintToPDF{
		PaperWidth:      floatPtr(paperWidthInches),
		PaperHeight:     floatPtr(paperHeightInches),
		MarginTop:       floatPtr(marginInches),
		MarginBottom:    floatPtr(marginInches),
		MarginLeft:      floatPtr(marginInches),
		MarginRight:     floatPtr(marginInches),
		PrintBackground: true,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPDFGeneration, err)
	}

	pdfBuf, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: reading PDF stream: %v", ErrPDFGeneration, err)
	}

	return pdfBuf, nil
}

// floatPtr returns a pointer to a float64 value.
func floatPtr(v float64) *float64 {
	return &v
}

// RodConverter converts HTML to PDF using headless Chrome via go-rod.
type RodConverter struct {
	Renderer PDFRenderer
}

// NewRodConverter creates a RodConverter with production renderer.
func NewRodConverter() *RodConverter {
	return &RodConverter{
		Renderer: &RodRenderer{Timeout: defaultTimeout},
	}
}

// NewRodConverterWith creates a RodConverter with custom renderer (for testing).
func NewRodConverterWith(renderer PDFRenderer) *RodConverter {
	if renderer == nil {
		panic("nil PDFRenderer in NewRodConverterWith")
	}
	return &RodConverter{Renderer: renderer}
}

// ToPDF converts HTML content to a PDF file using headless Chrome.
// Uses US Letter format (8.5x11 inches) with 0.5 inch margins.
func (c *RodConverter) ToPDF(htmlContent, outputPath string) error {
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
