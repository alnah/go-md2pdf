package md2pdf

// Notes:
// - Tests rodConverter and rodRenderer with mock implementations
// - Tests buildFooterTemplate with various footer configurations
// - Tests resolvePageDimensions for all page sizes and orientations
// - Tests buildPDFOptions for margin calculations with footer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alnah/go-md2pdf/internal/fileutil"
	"github.com/alnah/go-md2pdf/internal/pipeline"
)

// ---------------------------------------------------------------------------
// Compile-Time Interface Checks
// ---------------------------------------------------------------------------

var (
	_ pdfConverter = (*rodConverter)(nil)
	_ pdfRenderer  = (*rodRenderer)(nil)
)

// ---------------------------------------------------------------------------
// Mock Implementations
// ---------------------------------------------------------------------------

type mockRenderer struct {
	Result     []byte
	Err        error
	CalledWith string
	CalledOpts *pdfOptions
}

func (m *mockRenderer) RenderFromFile(ctx context.Context, filePath string, opts *pdfOptions) ([]byte, error) {
	m.CalledWith = filePath
	m.CalledOpts = opts
	return m.Result, m.Err
}

type testableRodConverter struct {
	mock *mockRenderer
}

func (c *testableRodConverter) ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error) {
	tmpPath, cleanup, err := fileutil.WriteTempFile(htmlContent, "html")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return c.mock.RenderFromFile(ctx, tmpPath, opts)
}

// ---------------------------------------------------------------------------
// TestRodConverter_ToPDF - PDF Conversion with Mock Renderer
// ---------------------------------------------------------------------------

func TestRodConverter_ToPDF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		html       string
		mock       *mockRenderer
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "successful render returns PDF bytes",
			html: "<html><body>Test</body></html>",
			mock: &mockRenderer{
				Result: []byte("%PDF-1.4 fake pdf content"),
			},
		},
		{
			name: "renderer error propagates",
			html: "<html></html>",
			mock: &mockRenderer{
				Err: errors.New("browser crashed"),
			},
			wantAnyErr: true,
		},
		{
			name: "empty HTML is valid",
			html: "",
			mock: &mockRenderer{
				Result: []byte("%PDF-1.4"),
			},
		},
		{
			name: "unicode content succeeds",
			html: "<html><body>Hello World</body></html>",
			mock: &mockRenderer{
				Result: []byte("%PDF-1.4 unicode"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter := &testableRodConverter{mock: tt.mock}
			ctx := context.Background()

			result, err := converter.ToPDF(ctx, tt.html, nil)

			if tt.wantAnyErr || tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify PDF bytes returned
			if string(result) != string(tt.mock.Result) {
				t.Errorf("expected result %q, got %q", tt.mock.Result, result)
			}

			// Verify renderer was called with temp file
			if !strings.Contains(tt.mock.CalledWith, "md2pdf-") {
				t.Errorf("expected temp file path with 'md2pdf-', got %q", tt.mock.CalledWith)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRodConverter_ToPDF_ContextCancellation - Context Handling
// ---------------------------------------------------------------------------

func TestRodConverter_ToPDF_ContextCancellation(t *testing.T) {
	t.Parallel()

	mock := &mockRenderer{
		Result: []byte("%PDF-1.4"),
	}
	converter := &testableRodConverter{mock: mock}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The mock doesn't check context, but real renderer would
	// This test verifies the converter accepts context parameter
	_, err := converter.ToPDF(ctx, "<html></html>", nil)
	// Mock doesn't check context, so it succeeds
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestNewRodConverter - Converter Creation
// ---------------------------------------------------------------------------

func TestNewRodConverter(t *testing.T) {
	t.Parallel()

	converter := newRodConverter(defaultTimeout)

	if converter.renderer == nil {
		t.Fatal("expected non-nil renderer")
	}

	if converter.renderer.timeout != defaultTimeout {
		t.Errorf("expected timeout %v, got %v", defaultTimeout, converter.renderer.timeout)
	}
}

// ---------------------------------------------------------------------------
// TestBuildFooterTemplate - Footer Template Generation
// ---------------------------------------------------------------------------

func TestBuildFooterTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     *pipeline.FooterData
		wantPart string // Substring that should appear
		wantNot  string // Substring that should NOT appear
	}{
		{
			name:     "nil data returns empty span",
			data:     nil,
			wantPart: "<span></span>",
		},
		{
			name:     "page number only",
			data:     &pipeline.FooterData{ShowPageNumber: true},
			wantPart: `class="pageNumber"`,
		},
		{
			name:     "date only",
			data:     &pipeline.FooterData{Date: "2025-01-15"},
			wantPart: "2025-01-15",
		},
		{
			name:     "status only",
			data:     &pipeline.FooterData{Status: "DRAFT"},
			wantPart: "DRAFT",
		},
		{
			name:     "text only",
			data:     &pipeline.FooterData{Text: "Footer Text"},
			wantPart: "Footer Text",
		},
		{
			name: "all fields",
			data: &pipeline.FooterData{
				ShowPageNumber: true,
				Date:           "2025-01-15",
				Status:         "DRAFT",
				Text:           "Custom",
			},
			wantPart: "pageNumber",
		},
		{
			name:     "left position",
			data:     &pipeline.FooterData{Text: "Test", Position: "left"},
			wantPart: "text-align: left",
		},
		{
			name:     "center position",
			data:     &pipeline.FooterData{Text: "Test", Position: "center"},
			wantPart: "text-align: center",
		},
		{
			name:     "right position (default)",
			data:     &pipeline.FooterData{Text: "Test", Position: "right"},
			wantPart: "text-align: right",
		},
		{
			name:     "empty position defaults to right",
			data:     &pipeline.FooterData{Text: "Test"},
			wantPart: "text-align: right",
		},
		{
			name:    "HTML escapes special chars",
			data:    &pipeline.FooterData{Text: "<script>alert('xss')</script>"},
			wantNot: "<script>",
		},
		{
			name:     "DocumentID only",
			data:     &pipeline.FooterData{DocumentID: "DOC-2024-001"},
			wantPart: "DOC-2024-001",
		},
		{
			name: "DocumentID with other fields",
			data: &pipeline.FooterData{
				Date:       "2025-01-15",
				Status:     "FINAL",
				DocumentID: "REF-001",
			},
			wantPart: "REF-001",
		},
		{
			name:    "DocumentID HTML escapes special chars",
			data:    &pipeline.FooterData{DocumentID: "<doc>&test</doc>"},
			wantNot: "<doc>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := buildFooterTemplate(tt.data)

			if tt.wantPart != "" && !strings.Contains(result, tt.wantPart) {
				t.Errorf("expected %q in result, got: %s", tt.wantPart, result)
			}
			if tt.wantNot != "" && strings.Contains(result, tt.wantNot) {
				t.Errorf("expected %q NOT in result, got: %s", tt.wantNot, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolvePageDimensions - Page Dimension Calculation
// ---------------------------------------------------------------------------

func TestResolvePageDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		page             *PageSettings
		hasFooter        bool
		wantW            float64
		wantH            float64
		wantMargin       float64
		wantBottomMargin float64
	}{
		{
			name:             "nil uses defaults (letter portrait)",
			page:             nil,
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       DefaultMargin,
			wantBottomMargin: DefaultMargin,
		},
		{
			name:             "nil with footer adds extra bottom margin",
			page:             nil,
			hasFooter:        true,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       DefaultMargin,
			wantBottomMargin: DefaultMargin + footerMarginExtra,
		},
		{
			name:             "letter portrait explicit",
			page:             &PageSettings{Size: "letter", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "letter landscape swaps dimensions",
			page:             &PageSettings{Size: "letter", Orientation: "landscape", Margin: 0.5},
			hasFooter:        false,
			wantW:            11.0,
			wantH:            8.5,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "a4 portrait",
			page:             &PageSettings{Size: "a4", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.27,
			wantH:            11.69,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "a4 landscape",
			page:             &PageSettings{Size: "a4", Orientation: "landscape", Margin: 0.5},
			hasFooter:        false,
			wantW:            11.69,
			wantH:            8.27,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "legal portrait",
			page:             &PageSettings{Size: "legal", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            14.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "legal landscape",
			page:             &PageSettings{Size: "legal", Orientation: "landscape", Margin: 0.5},
			hasFooter:        false,
			wantW:            14.0,
			wantH:            8.5,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "custom margin",
			page:             &PageSettings{Size: "letter", Orientation: "portrait", Margin: 1.0},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       1.0,
			wantBottomMargin: 1.0,
		},
		{
			name:             "custom margin with footer",
			page:             &PageSettings{Size: "letter", Orientation: "portrait", Margin: 1.0},
			hasFooter:        true,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       1.0,
			wantBottomMargin: 1.0 + footerMarginExtra,
		},
		{
			name:             "case insensitive size",
			page:             &PageSettings{Size: "A4", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.27,
			wantH:            11.69,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "case insensitive orientation",
			page:             &PageSettings{Size: "letter", Orientation: "LANDSCAPE", Margin: 0.5},
			hasFooter:        false,
			wantW:            11.0,
			wantH:            8.5,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "unknown size falls back to letter",
			page:             &PageSettings{Size: "tabloid", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "empty size uses default letter",
			page:             &PageSettings{Size: "", Orientation: "portrait", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "empty orientation uses portrait",
			page:             &PageSettings{Size: "letter", Orientation: "", Margin: 0.5},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       0.5,
			wantBottomMargin: 0.5,
		},
		{
			name:             "zero margin uses default",
			page:             &PageSettings{Size: "letter", Orientation: "portrait", Margin: 0},
			hasFooter:        false,
			wantW:            8.5,
			wantH:            11.0,
			wantMargin:       DefaultMargin,
			wantBottomMargin: DefaultMargin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w, h, margin, bottomMargin := resolvePageDimensions(tt.page, tt.hasFooter)

			if w != tt.wantW {
				t.Errorf("width = %v, want %v", w, tt.wantW)
			}
			if h != tt.wantH {
				t.Errorf("height = %v, want %v", h, tt.wantH)
			}
			if margin != tt.wantMargin {
				t.Errorf("margin = %v, want %v", margin, tt.wantMargin)
			}
			if bottomMargin != tt.wantBottomMargin {
				t.Errorf("bottomMargin = %v, want %v", bottomMargin, tt.wantBottomMargin)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRodRenderer_Close_Idempotent - Close Idempotency
// ---------------------------------------------------------------------------

func TestRodRenderer_Close_Idempotent(t *testing.T) {
	t.Parallel()

	renderer := newRodRenderer(defaultTimeout)

	// Multiple calls should not panic and all should succeed
	err1 := renderer.Close()
	err2 := renderer.Close()
	err3 := renderer.Close()

	if err1 != nil {
		t.Errorf("first Close() error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("second Close() error = %v", err2)
	}
	if err3 != nil {
		t.Errorf("third Close() error = %v", err3)
	}
}

// ---------------------------------------------------------------------------
// TestRodConverter_Close_NilRenderer - Close with Nil Renderer
// ---------------------------------------------------------------------------

func TestRodConverter_Close_NilRenderer(t *testing.T) {
	t.Parallel()

	converter := &rodConverter{renderer: nil}

	err := converter.Close()
	if err != nil {
		t.Errorf("Close() with nil renderer should not error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestBuildPDFOptions - PDF Options Construction
// ---------------------------------------------------------------------------

func TestBuildPDFOptions(t *testing.T) {
	t.Parallel()

	renderer := &rodRenderer{timeout: defaultTimeout}

	t.Run("nil opts uses default margins", func(t *testing.T) {
		t.Parallel()

		pdfOpts := renderer.buildPDFOptions(nil)

		if *pdfOpts.MarginBottom != DefaultMargin {
			t.Errorf("expected margin %v, got %v", DefaultMargin, *pdfOpts.MarginBottom)
		}
		if pdfOpts.DisplayHeaderFooter {
			t.Error("expected no header/footer by default")
		}
	})

	t.Run("with footer increases bottom margin", func(t *testing.T) {
		t.Parallel()

		opts := &pdfOptions{Footer: &pipeline.FooterData{Text: "Footer"}}
		pdfOpts := renderer.buildPDFOptions(opts)

		expectedMargin := DefaultMargin + footerMarginExtra
		if *pdfOpts.MarginBottom != expectedMargin {
			t.Errorf("expected margin %v, got %v", expectedMargin, *pdfOpts.MarginBottom)
		}
		if !pdfOpts.DisplayHeaderFooter {
			t.Error("expected header/footer enabled")
		}
	})

	t.Run("with page settings uses custom dimensions", func(t *testing.T) {
		t.Parallel()

		opts := &pdfOptions{
			Page: &PageSettings{Size: "a4", Orientation: "landscape", Margin: 1.0},
		}
		pdfOpts := renderer.buildPDFOptions(opts)

		if *pdfOpts.PaperWidth != 11.69 {
			t.Errorf("PaperWidth = %v, want 11.69", *pdfOpts.PaperWidth)
		}
		if *pdfOpts.PaperHeight != 8.27 {
			t.Errorf("PaperHeight = %v, want 8.27", *pdfOpts.PaperHeight)
		}
		if *pdfOpts.MarginTop != 1.0 {
			t.Errorf("MarginTop = %v, want 1.0", *pdfOpts.MarginTop)
		}
		if *pdfOpts.MarginBottom != 1.0 {
			t.Errorf("MarginBottom = %v, want 1.0", *pdfOpts.MarginBottom)
		}
	})

	t.Run("with page settings and footer", func(t *testing.T) {
		t.Parallel()

		opts := &pdfOptions{
			Page:   &PageSettings{Size: "letter", Orientation: "portrait", Margin: 0.75},
			Footer: &pipeline.FooterData{Text: "Footer"},
		}
		pdfOpts := renderer.buildPDFOptions(opts)

		if *pdfOpts.MarginTop != 0.75 {
			t.Errorf("MarginTop = %v, want 0.75", *pdfOpts.MarginTop)
		}
		expectedBottom := 0.75 + footerMarginExtra
		if *pdfOpts.MarginBottom != expectedBottom {
			t.Errorf("MarginBottom = %v, want %v", *pdfOpts.MarginBottom, expectedBottom)
		}
		if !pdfOpts.DisplayHeaderFooter {
			t.Error("expected header/footer enabled")
		}
	})
}

// ---------------------------------------------------------------------------
// TestPageDimensions_AllSizesPresent - Page Dimensions Map Completeness
// ---------------------------------------------------------------------------

func TestPageDimensions_AllSizesPresent(t *testing.T) {
	t.Parallel()

	requiredSizes := []string{PageSizeLetter, PageSizeA4, PageSizeLegal}

	for _, size := range requiredSizes {
		if _, ok := pageDimensions[size]; !ok {
			t.Errorf("missing page dimensions for size %q", size)
		}
	}
}

// ---------------------------------------------------------------------------
// TestPageDimensions_ValidValues - Page Dimensions Value Validity
// ---------------------------------------------------------------------------

func TestPageDimensions_ValidValues(t *testing.T) {
	t.Parallel()

	for size, dims := range pageDimensions {
		t.Run(size, func(t *testing.T) {
			t.Parallel()

			if dims.width <= 0 {
				t.Errorf("width must be positive, got %v", dims.width)
			}
			if dims.height <= 0 {
				t.Errorf("height must be positive, got %v", dims.height)
			}
			// Portrait dimensions: height > width
			if dims.height <= dims.width {
				t.Errorf("expected portrait dimensions (height > width), got %v x %v", dims.width, dims.height)
			}
		})
	}
}
