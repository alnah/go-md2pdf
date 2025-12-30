package main

import (
	"errors"
	"fmt"
)

// Sentinel errors for service operations.
var (
	ErrEmptyMarkdown = errors.New("markdown content cannot be empty")
	ErrEmptyOutput   = errors.New("output path cannot be empty")
)

// ConversionOptions holds all inputs for a conversion.
type ConversionOptions struct {
	MarkdownContent string         // Raw markdown content (required)
	OutputPath      string         // Path for output PDF (required)
	CSSContent      string         // Custom CSS (empty = no CSS)
	Footer          *FooterData    // Footer data (nil = no footer)
	Signature       *SignatureData // Signature data (nil = no signature)
}

// ConversionService orchestrates the markdown-to-PDF pipeline.
type ConversionService struct {
	preprocessor      MarkdownPreprocessor
	htmlConverter     HTMLConverter
	cssInjector       CSSInjector
	footerInjector    FooterInjector
	signatureInjector SignatureInjector
	pdfConverter      PDFConverter
}

// NewConversionService creates a service with production dependencies.
func NewConversionService() *ConversionService {
	return &ConversionService{
		preprocessor:      &CommonMarkPreprocessor{},
		htmlConverter:     NewGoldmarkConverter(),
		cssInjector:       &CSSInjection{},
		footerInjector:    &FooterInjection{},
		signatureInjector: NewSignatureInjection(),
		pdfConverter:      NewRodConverter(),
	}
}

// NewConversionServiceWith creates a service with custom dependencies (for testing).
// Panics if any dependency is nil.
func NewConversionServiceWith(
	preprocessor MarkdownPreprocessor,
	htmlConverter HTMLConverter,
	cssInjector CSSInjector,
	footerInjector FooterInjector,
	signatureInjector SignatureInjector,
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
	if footerInjector == nil {
		panic("nil footerInjector provided to ConversionService")
	}
	if signatureInjector == nil {
		panic("nil signatureInjector provided to ConversionService")
	}
	if pdfConverter == nil {
		panic("nil pdfConverter provided to ConversionService")
	}
	return &ConversionService{
		preprocessor:      preprocessor,
		htmlConverter:     htmlConverter,
		cssInjector:       cssInjector,
		footerInjector:    footerInjector,
		signatureInjector: signatureInjector,
		pdfConverter:      pdfConverter,
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

	// Inject footer (if provided)
	htmlContent = s.footerInjector.InjectFooter(htmlContent, opts.Footer)

	// Inject signature (if provided)
	htmlContent, err = s.signatureInjector.InjectSignature(htmlContent, opts.Signature)
	if err != nil {
		return fmt.Errorf("injecting signature: %w", err)
	}

	// Convert to PDF
	if err := s.pdfConverter.ToPDF(htmlContent, opts.OutputPath); err != nil {
		return fmt.Errorf("converting to PDF: %w", err)
	}

	return nil
}

// validateOptions checks that required fields are present.
func (s *ConversionService) validateOptions(opts ConversionOptions) error {
	if opts.MarkdownContent == "" {
		return ErrEmptyMarkdown
	}
	if opts.OutputPath == "" {
		return ErrEmptyOutput
	}
	return nil
}

// resolveCSS returns the CSS content from options.
func (s *ConversionService) resolveCSS(opts ConversionOptions) string {
	return opts.CSSContent
}
