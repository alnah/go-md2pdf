package md2pdf

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockRenderer implements pdfRenderer for testing.
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

// testableRodConverter wraps rodConverter for testing with mock renderer.
type testableRodConverter struct {
	mock *mockRenderer
}

func (c *testableRodConverter) ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error) {
	tmpPath, cleanup, err := writeTempFile(htmlContent, "html")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return c.mock.RenderFromFile(ctx, tmpPath, opts)
}

func TestRodConverter_ToPDF(t *testing.T) {
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
			html: "<html><body>Bonjour le monde</body></html>",
			mock: &mockRenderer{
				Result: []byte("%PDF-1.4 unicode"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestRodConverter_ToPDF_ContextCancellation(t *testing.T) {
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

func TestNewRodConverter(t *testing.T) {
	converter := newRodConverter(defaultTimeout)

	if converter.renderer == nil {
		t.Fatal("expected non-nil renderer")
	}

	if converter.renderer.timeout != defaultTimeout {
		t.Errorf("expected timeout %v, got %v", defaultTimeout, converter.renderer.timeout)
	}
}

func TestBuildFooterTemplate(t *testing.T) {
	tests := []struct {
		name     string
		data     *footerData
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
			data:     &footerData{ShowPageNumber: true},
			wantPart: `class="pageNumber"`,
		},
		{
			name:     "date only",
			data:     &footerData{Date: "2025-01-15"},
			wantPart: "2025-01-15",
		},
		{
			name:     "status only",
			data:     &footerData{Status: "DRAFT"},
			wantPart: "DRAFT",
		},
		{
			name:     "text only",
			data:     &footerData{Text: "Footer Text"},
			wantPart: "Footer Text",
		},
		{
			name: "all fields",
			data: &footerData{
				ShowPageNumber: true,
				Date:           "2025-01-15",
				Status:         "DRAFT",
				Text:           "Custom",
			},
			wantPart: "pageNumber",
		},
		{
			name:     "left position",
			data:     &footerData{Text: "Test", Position: "left"},
			wantPart: "text-align: left",
		},
		{
			name:     "center position",
			data:     &footerData{Text: "Test", Position: "center"},
			wantPart: "text-align: center",
		},
		{
			name:     "right position (default)",
			data:     &footerData{Text: "Test", Position: "right"},
			wantPart: "text-align: right",
		},
		{
			name:     "empty position defaults to right",
			data:     &footerData{Text: "Test"},
			wantPart: "text-align: right",
		},
		{
			name:    "HTML escapes special chars",
			data:    &footerData{Text: "<script>alert('xss')</script>"},
			wantNot: "<script>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestBuildPDFOptions(t *testing.T) {
	renderer := &rodRenderer{timeout: defaultTimeout}

	t.Run("nil opts uses default margins", func(t *testing.T) {
		pdfOpts := renderer.buildPDFOptions(nil)

		if *pdfOpts.MarginBottom != marginInches {
			t.Errorf("expected margin %v, got %v", marginInches, *pdfOpts.MarginBottom)
		}
		if pdfOpts.DisplayHeaderFooter {
			t.Error("expected no header/footer by default")
		}
	})

	t.Run("with footer increases bottom margin", func(t *testing.T) {
		opts := &pdfOptions{Footer: &footerData{Text: "Footer"}}
		pdfOpts := renderer.buildPDFOptions(opts)

		if *pdfOpts.MarginBottom != marginBottomWithFooter {
			t.Errorf("expected margin %v, got %v", marginBottomWithFooter, *pdfOpts.MarginBottom)
		}
		if !pdfOpts.DisplayHeaderFooter {
			t.Error("expected header/footer enabled")
		}
	})
}
