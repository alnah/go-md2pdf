//go:build integration

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConversionService(t *testing.T) {
	service := NewConversionService()

	if service.preprocessor == nil {
		t.Error("preprocessor is nil")
	}
	if _, ok := service.preprocessor.(*CommonMarkToPandocPreprocessor); !ok {
		t.Errorf("preprocessor type = %T, want *CommonMarkToPandocPreprocessor", service.preprocessor)
	}

	if service.htmlConverter == nil {
		t.Error("htmlConverter is nil")
	}
	if _, ok := service.htmlConverter.(*PandocConverter); !ok {
		t.Errorf("htmlConverter type = %T, want *PandocConverter", service.htmlConverter)
	}

	if service.cssInjector == nil {
		t.Error("cssInjector is nil")
	}
	if _, ok := service.cssInjector.(*CSSInjection); !ok {
		t.Errorf("cssInjector type = %T, want *CSSInjection", service.cssInjector)
	}

	if service.pdfConverter == nil {
		t.Error("pdfConverter is nil")
	}
	if _, ok := service.pdfConverter.(*ChromeConverter); !ok {
		t.Errorf("pdfConverter type = %T, want *ChromeConverter", service.pdfConverter)
	}
}

func TestConversionService_Convert_Integration(t *testing.T) {
	service := NewConversionService()

	opts := ConversionOptions{
		MarkdownContent: "# Hello\n\nWorld",
		OutputPath:      filepath.Join(t.TempDir(), "out.pdf"),
	}

	err := service.Convert(opts)
	if err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	info, err := os.Stat(opts.OutputPath)
	if err != nil {
		t.Fatalf("PDF not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("PDF file is empty")
	}
}
