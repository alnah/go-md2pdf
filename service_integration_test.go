//go:build integration

package md2pdf

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewConversionService(t *testing.T) {
	service := New()
	defer service.Close()

	if service.preprocessor == nil {
		t.Error("preprocessor is nil")
	}
	if _, ok := service.preprocessor.(*commonMarkPreprocessor); !ok {
		t.Errorf("preprocessor type = %T, want *commonMarkPreprocessor", service.preprocessor)
	}

	if service.htmlConverter == nil {
		t.Error("htmlConverter is nil")
	}
	if _, ok := service.htmlConverter.(*goldmarkConverter); !ok {
		t.Errorf("htmlConverter type = %T, want *goldmarkConverter", service.htmlConverter)
	}

	if service.cssInjector == nil {
		t.Error("cssInjector is nil")
	}
	if _, ok := service.cssInjector.(*cssInjection); !ok {
		t.Errorf("cssInjector type = %T, want *cssInjection", service.cssInjector)
	}

	if service.pdfConverter == nil {
		t.Error("pdfConverter is nil")
	}
	// pdfConverter is already *rodConverter (concrete type), type assertion not needed
}

func TestConversionService_Convert_Integration(t *testing.T) {
	service := New()
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: "# Hello\n\nWorld",
	}

	data, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	// Verify PDF bytes
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Error("output does not have PDF magic bytes")
	}

	if len(data) < 100 {
		t.Error("PDF data suspiciously small")
	}
}

func TestConversionService_WriteToFile_Integration(t *testing.T) {
	service := New()
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: "# Hello\n\nWorld",
	}

	data, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "out.pdf")
	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("PDF not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("PDF file is empty")
	}
}

func TestConversionService_PageSettings_Integration(t *testing.T) {
	// Test various page settings combinations to ensure they don't crash
	// and produce valid PDF output
	tests := []struct {
		name string
		page *PageSettings
	}{
		{
			name: "nil uses defaults",
			page: nil,
		},
		{
			name: "letter portrait",
			page: &PageSettings{Size: PageSizeLetter, Orientation: OrientationPortrait, Margin: DefaultMargin},
		},
		{
			name: "a4 portrait",
			page: &PageSettings{Size: PageSizeA4, Orientation: OrientationPortrait, Margin: 0.5},
		},
		{
			name: "a4 landscape",
			page: &PageSettings{Size: PageSizeA4, Orientation: OrientationLandscape, Margin: 0.5},
		},
		{
			name: "legal portrait",
			page: &PageSettings{Size: PageSizeLegal, Orientation: OrientationPortrait, Margin: 0.5},
		},
		{
			name: "legal landscape",
			page: &PageSettings{Size: PageSizeLegal, Orientation: OrientationLandscape, Margin: 1.0},
		},
		{
			name: "letter landscape custom margin",
			page: &PageSettings{Size: PageSizeLetter, Orientation: OrientationLandscape, Margin: 1.5},
		},
		{
			name: "minimum margin",
			page: &PageSettings{Size: PageSizeLetter, Orientation: OrientationPortrait, Margin: MinMargin},
		},
		{
			name: "maximum margin",
			page: &PageSettings{Size: PageSizeLetter, Orientation: OrientationPortrait, Margin: MaxMargin},
		},
	}

	service := New()
	defer service.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			input := Input{
				Markdown: "# Page Settings Test\n\nThis is a test document.",
				Page:     tt.page,
			}

			data, err := service.Convert(ctx, input)
			if err != nil {
				t.Fatalf("Convert() failed: %v", err)
			}

			// Verify PDF magic bytes
			if !bytes.HasPrefix(data, []byte("%PDF-")) {
				t.Error("output does not have PDF magic bytes")
			}

			// Ensure PDF is not suspiciously small
			if len(data) < 100 {
				t.Errorf("PDF data suspiciously small: %d bytes", len(data))
			}
		})
	}
}

func TestConversionService_PageSettingsWithFooter_Integration(t *testing.T) {
	service := New()
	defer service.Close()

	ctx := context.Background()
	input := Input{
		Markdown: "# Test with Footer\n\nContent here.",
		Page:     &PageSettings{Size: PageSizeA4, Orientation: OrientationLandscape, Margin: 1.0},
		Footer: &Footer{
			Position:       "center",
			ShowPageNumber: true,
			Text:           "Footer Text",
		},
	}

	data, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Error("output does not have PDF magic bytes")
	}
}
