package md2pdf

import (
	"context"
	"fmt"
	"html"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// pdfConverter abstracts HTML to PDF conversion to allow different backends.
type pdfConverter interface {
	ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error)
	Close() error
}

// pdfRenderer abstracts PDF rendering from an HTML file to enable testing without a browser.
type pdfRenderer interface {
	RenderFromFile(ctx context.Context, filePath string, opts *pdfOptions) ([]byte, error)
}

// Compile-time interface checks
var (
	_ pdfConverter = (*rodConverter)(nil)
	_ pdfRenderer  = (*rodRenderer)(nil)
)

// pdfOptions holds options for PDF generation.
type pdfOptions struct {
	Footer *footerData
}

// PDF page dimensions in inches (US Letter format).
const (
	paperWidthInches       = 8.5
	paperHeightInches      = 11
	marginInches           = 0.5
	marginBottomWithFooter = 0.75 // Extra space for footer
)

// rodRenderer implements pdfRenderer using go-rod.
// Rod automatically downloads Chromium on first run if not found.
type rodRenderer struct {
	browser *rod.Browser
	timeout time.Duration
}

// newRodRenderer creates a rodRenderer with the given timeout.
func newRodRenderer(timeout time.Duration) *rodRenderer {
	return &rodRenderer{timeout: timeout}
}

// ensureBrowser lazily connects to the browser.
func (r *rodRenderer) ensureBrowser() error {
	if r.browser != nil {
		return nil
	}

	// Configure launcher
	l := launcher.New()

	// Use pre-installed browser if specified (Docker/containerized environments)
	if bin := os.Getenv("ROD_BROWSER_BIN"); bin != "" {
		l = l.Bin(bin)
	}

	// NoSandbox required for CI and containerized environments
	if os.Getenv("CI") == "true" || os.Getenv("ROD_BROWSER_BIN") != "" {
		l = l.NoSandbox(true)
	}
	u, err := l.Launch()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}

	r.browser = rod.New().ControlURL(u)
	if err := r.browser.Connect(); err != nil {
		r.browser = nil
		return fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	return nil
}

// Close releases browser resources.
func (r *rodRenderer) Close() error {
	if r.browser != nil {
		err := r.browser.Close()
		r.browser = nil
		return err
	}
	return nil
}

// RenderFromFile opens a local HTML file in headless Chrome and renders it to PDF.
// Returns explicit errors instead of panicking when browser operations fail.
func (r *rodRenderer) RenderFromFile(ctx context.Context, filePath string, opts *pdfOptions) ([]byte, error) {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := r.ensureBrowser(); err != nil {
		return nil, err
	}

	page, err := r.browser.Page(proto.TargetCreateTarget{URL: "file://" + filePath})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPageCreate, err)
	}
	defer page.Close()

	// Wait for page to load with timeout from context or default
	timeout := r.timeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			return nil, context.DeadlineExceeded
		}
	}

	if err := page.Timeout(timeout).WaitLoad(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPageLoad, err)
	}

	// Check context after page load
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Build PDF options
	pdfOpts := r.buildPDFOptions(opts)

	// Generate PDF
	reader, err := page.PDF(pdfOpts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPDFGeneration, err)
	}

	pdfBuf, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: reading PDF stream: %v", ErrPDFGeneration, err)
	}

	return pdfBuf, nil
}

// buildPDFOptions constructs proto.PagePrintToPDF with optional footer.
func (r *rodRenderer) buildPDFOptions(opts *pdfOptions) *proto.PagePrintToPDF {
	marginBottom := marginInches
	hasFooter := opts != nil && opts.Footer != nil

	if hasFooter {
		marginBottom = marginBottomWithFooter
	}

	pdfOpts := &proto.PagePrintToPDF{
		PaperWidth:      floatPtr(paperWidthInches),
		PaperHeight:     floatPtr(paperHeightInches),
		MarginTop:       floatPtr(marginInches),
		MarginBottom:    floatPtr(marginBottom),
		MarginLeft:      floatPtr(marginInches),
		MarginRight:     floatPtr(marginInches),
		PrintBackground: true,
	}

	if hasFooter {
		pdfOpts.DisplayHeaderFooter = true
		pdfOpts.HeaderTemplate = "<span></span>" // Empty header
		pdfOpts.FooterTemplate = buildFooterTemplate(opts.Footer)
	}

	return pdfOpts
}

// buildFooterTemplate generates an HTML template for Chrome's native footer.
// Supports pageNumber, totalPages, date placeholders via CSS classes.
func buildFooterTemplate(data *footerData) string {
	if data == nil {
		return "<span></span>"
	}

	var parts []string

	if data.ShowPageNumber {
		parts = append(parts, `<span class="pageNumber"></span>/<span class="totalPages"></span>`)
	}
	if data.Date != "" {
		parts = append(parts, html.EscapeString(data.Date))
	}
	if data.Status != "" {
		parts = append(parts, html.EscapeString(data.Status))
	}
	if data.Text != "" {
		parts = append(parts, html.EscapeString(data.Text))
	}

	if len(parts) == 0 {
		return "<span></span>"
	}

	content := strings.Join(parts, " - ")

	// Position: left, center, or right (default)
	textAlign := "right"
	switch data.Position {
	case "left":
		textAlign = "left"
	case "center":
		textAlign = "center"
	}

	return fmt.Sprintf(`<div style="font-size: 10px; font-family: %s; color: #aaa; width: 100%%; text-align: %s; padding: 0 0.5in;">%s</div>`, defaultFontFamily, textAlign, content)
}

// floatPtr returns a pointer to a float64 value.
func floatPtr(v float64) *float64 {
	return &v
}

// rodConverter converts HTML to PDF using headless Chrome via go-rod.
type rodConverter struct {
	renderer *rodRenderer
}

// newRodConverter creates a rodConverter with production renderer.
func newRodConverter(timeout time.Duration) *rodConverter {
	return &rodConverter{
		renderer: newRodRenderer(timeout),
	}
}

// ToPDF converts HTML content to PDF bytes using headless Chrome.
// Uses US Letter format (8.5x11 inches) with 0.5 inch margins.
func (c *rodConverter) ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error) {
	tmpPath, cleanup, err := writeTempFile(htmlContent, "html")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return c.renderer.RenderFromFile(ctx, tmpPath, opts)
}

// Close releases browser resources.
func (c *rodConverter) Close() error {
	if c.renderer != nil {
		return c.renderer.Close()
	}
	return nil
}
