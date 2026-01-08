package md2pdf

import (
	"context"
	"fmt"
	"html"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// browserCloseTimeout is the maximum time to wait for browser.Close() before force-killing.
const browserCloseTimeout = 5 * time.Second

// pdfConverter abstracts HTML to PDF conversion to allow different backends.
type pdfConverter interface {
	ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error)
	Close() error
}

// pdfRenderer abstracts PDF rendering from an HTML file to enable testing without a browser.
type pdfRenderer interface {
	RenderFromFile(ctx context.Context, filePath string, opts *pdfOptions) ([]byte, error)
}

// pdfOptions holds options for PDF generation.
type pdfOptions struct {
	Footer *footerData
	Page   *PageSettings
}

// footerMarginExtra is added to bottom margin when footer is active.
const footerMarginExtra = 0.25

// pageDimensions maps page size to (width, height) in inches.
var pageDimensions = map[string]struct{ width, height float64 }{
	PageSizeLetter: {8.5, 11.0},
	PageSizeA4:     {8.27, 11.69},
	PageSizeLegal:  {8.5, 14.0},
}

// rodRenderer implements pdfRenderer using go-rod.
// Rod automatically downloads Chromium on first run if not found.
type rodRenderer struct {
	browser   *rod.Browser
	launcher  *launcher.Launcher
	timeout   time.Duration
	closeOnce sync.Once
}

// newRodRenderer creates a rodRenderer with the given timeout.
func newRodRenderer(timeout time.Duration) *rodRenderer {
	return &rodRenderer{timeout: timeout}
}

// findSystemChrome returns the path to system Chrome/Chromium if found.
// Returns empty string if not found (rod will download its own).
func findSystemChrome() string {
	// macOS paths
	paths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
	}

	// Linux paths
	paths = append(paths,
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	)

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// ensureBrowser lazily connects to the browser.
func (r *rodRenderer) ensureBrowser() error {
	if r.browser != nil {
		return nil
	}

	// Configure launcher
	// Leakless(false) prevents hanging on macOS - see github.com/go-rod/rod/issues/210
	// We compensate by explicitly calling Kill() and Cleanup() in Close().
	l := launcher.New().Headless(true).Leakless(false).Set("disable-gpu")

	// Use pre-installed browser if specified, or auto-detect on macOS/Linux
	bin := os.Getenv("ROD_BROWSER_BIN")
	if bin == "" {
		bin = findSystemChrome()
	}
	if bin != "" {
		l = l.Bin(bin).NoSandbox(true)
	}

	u, err := l.Launch()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}

	// Store launcher reference for cleanup in Close()
	r.launcher = l

	r.browser = rod.New().ControlURL(u)
	if err := r.browser.Connect(); err != nil {
		r.launcher.Kill()
		r.launcher.Cleanup()
		r.browser = nil
		r.launcher = nil
		return fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	return nil
}

// Close releases browser resources.
// Safe to call multiple times (idempotent via sync.Once).
// Uses a timeout to avoid hanging indefinitely if browser.Close() blocks.
func (r *rodRenderer) Close() error {
	var closeErr error
	r.closeOnce.Do(func() {
		// Get PID before any cleanup - we'll need it to kill the process group
		var pid int
		if r.launcher != nil {
			pid = r.launcher.PID()
		}

		// Try graceful close first with timeout
		if r.browser != nil {
			done := make(chan error, 1)
			go func() {
				done <- r.browser.Close()
			}()

			select {
			case closeErr = <-done:
				// Browser closed normally
			case <-time.After(browserCloseTimeout):
				// Timeout - will be force-killed below
			}
			r.browser = nil
		}

		// Force kill the Chrome process group (kills all child processes too)
		if pid > 0 {
			// Kill the entire process group to ensure GPU, renderer,
			// and other Chrome child processes are terminated.
			killProcessGroup(pid)
		}

		// Also call launcher.Kill() as fallback and cleanup user-data-dir
		if r.launcher != nil {
			r.launcher.Kill()
			r.launcher.Cleanup()
			r.launcher = nil
		}
	})
	return closeErr
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

// resolvePageDimensions returns width, height, margin, and bottom margin.
// Applies defaults for nil/zero values, swaps for landscape, adds footer space.
func resolvePageDimensions(page *PageSettings, hasFooter bool) (w, h, margin, bottomMargin float64) {
	// Apply defaults
	size := PageSizeLetter
	orientation := OrientationPortrait
	margin = DefaultMargin

	if page != nil {
		if page.Size != "" {
			size = strings.ToLower(page.Size)
		}
		if page.Orientation != "" {
			orientation = strings.ToLower(page.Orientation)
		}
		if page.Margin > 0 {
			margin = page.Margin
		}
	}

	// Get dimensions for size
	dims, ok := pageDimensions[size]
	if !ok {
		dims = pageDimensions[PageSizeLetter] // fallback
	}
	w, h = dims.width, dims.height

	// Swap for landscape
	if orientation == OrientationLandscape {
		w, h = h, w
	}

	// Bottom margin: add extra space for footer
	bottomMargin = margin
	if hasFooter {
		bottomMargin = margin + footerMarginExtra
	}

	return w, h, margin, bottomMargin
}

// buildPDFOptions constructs proto.PagePrintToPDF with page settings and optional footer.
func (r *rodRenderer) buildPDFOptions(opts *pdfOptions) *proto.PagePrintToPDF {
	hasFooter := opts != nil && opts.Footer != nil
	var page *PageSettings
	if opts != nil {
		page = opts.Page
	}

	w, h, margin, bottomMargin := resolvePageDimensions(page, hasFooter)

	pdfOpts := &proto.PagePrintToPDF{
		PaperWidth:      floatPtr(w),
		PaperHeight:     floatPtr(h),
		MarginTop:       floatPtr(margin),
		MarginBottom:    floatPtr(bottomMargin),
		MarginLeft:      floatPtr(margin),
		MarginRight:     floatPtr(margin),
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
// Page dimensions are configured via opts.Page (defaults to US Letter, portrait, 0.5in margins).
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
