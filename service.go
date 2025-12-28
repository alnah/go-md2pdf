package main

import (
	"errors"
	"fmt"
)

// Sentinel errors for service operations.
var (
	ErrConflictingCSSOptions = errors.New("cannot use both NoStyle and CSSContent")
	ErrEmptyMarkdown         = errors.New("markdown content cannot be empty")
	ErrEmptyOutput           = errors.New("output path cannot be empty")
)

// ConversionOptions holds all inputs for a conversion.
type ConversionOptions struct {
	MarkdownContent string // Raw markdown content (required)
	OutputPath      string // Path for output PDF (required)
	CSSContent      string // Custom CSS (empty = use default)
	NoStyle         bool   // Skip CSS injection entirely
}

// ConversionService orchestrates the markdown-to-PDF pipeline.
type ConversionService struct {
	preprocessor  MarkdownPreprocessor
	htmlConverter HTMLConverter
	cssInjector   CSSInjector
	pdfConverter  PDFConverter
}

// NewConversionService creates a service with production dependencies.
func NewConversionService() *ConversionService {
	return &ConversionService{
		preprocessor:  &CommonMarkToPandocPreprocessor{},
		htmlConverter: NewPandocConverter(),
		cssInjector:   &CSSInjection{},
		pdfConverter:  NewChromeConverter(),
	}
}

// NewConversionServiceWith creates a service with custom dependencies (for testing).
// Panics if any dependency is nil.
func NewConversionServiceWith(
	preprocessor MarkdownPreprocessor,
	htmlConverter HTMLConverter,
	cssInjector CSSInjector,
	pdfConverter PDFConverter,
) *ConversionService {
	if preprocessor == nil {
		panic("nil preprocessor provided to ConversionService")
	}
	if htmlConverter == nil {
		panic("nil htmlConverter provided to ConversionService")
	}
	if cssInjector == nil {
		panic("nil cssInjector provided to ConversionService")
	}
	if pdfConverter == nil {
		panic("nil pdfConverter provided to ConversionService")
	}
	return &ConversionService{
		preprocessor:  preprocessor,
		htmlConverter: htmlConverter,
		cssInjector:   cssInjector,
		pdfConverter:  pdfConverter,
	}
}

// Convert executes the full markdown-to-PDF pipeline.
func (s *ConversionService) Convert(opts ConversionOptions) error {
	if err := s.validateOptions(opts); err != nil {
		return err
	}

	css := s.resolveCSS(opts)

	// Preprocess markdown
	mdContent := s.preprocessor.PreprocessMarkdown(opts.MarkdownContent)

	// Convert to HTML
	htmlContent, err := s.htmlConverter.ToHTML(mdContent)
	if err != nil {
		return fmt.Errorf("converting to HTML: %w", err)
	}

	// Inject CSS
	htmlContent = s.cssInjector.InjectCSS(htmlContent, css)

	// Convert to PDF
	if err := s.pdfConverter.ToPDF(htmlContent, opts.OutputPath); err != nil {
		return fmt.Errorf("converting to PDF: %w", err)
	}

	return nil
}

// validateOptions checks that required fields are present and options are consistent.
func (s *ConversionService) validateOptions(opts ConversionOptions) error {
	if opts.MarkdownContent == "" {
		return ErrEmptyMarkdown
	}
	if opts.OutputPath == "" {
		return ErrEmptyOutput
	}
	if opts.NoStyle && opts.CSSContent != "" {
		return ErrConflictingCSSOptions
	}
	return nil
}

// resolveCSS determines which CSS to use based on options.
// Assumes validation has already been performed.
func (s *ConversionService) resolveCSS(opts ConversionOptions) string {
	if opts.NoStyle {
		return ""
	}
	if opts.CSSContent != "" {
		return opts.CSSContent
	}
	return defaultCSS
}
