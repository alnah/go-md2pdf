package md2pdf

import (
	"context"
	"fmt"
)

// Service orchestrates the markdown-to-PDF pipeline.
type Service struct {
	cfg               serviceConfig
	preprocessor      markdownPreprocessor
	htmlConverter     htmlConverter
	cssInjector       cssInjector
	coverInjector     coverInjector
	signatureInjector signatureInjector
	pdfConverter      pdfConverter
}

// New creates a Service with default configuration.
// Use options to customize behavior (e.g., WithTimeout).
func New(opts ...Option) *Service {
	s := &Service{
		cfg:               serviceConfig{timeout: defaultTimeout},
		preprocessor:      &commonMarkPreprocessor{},
		htmlConverter:     newGoldmarkConverter(),
		cssInjector:       &cssInjection{},
		coverInjector:     newCoverInjection(),
		signatureInjector: newSignatureInjection(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Create PDF converter if not injected (e.g., by tests)
	if s.pdfConverter == nil {
		s.pdfConverter = newRodConverter(s.cfg.timeout)
	}

	return s
}

// Convert runs the full pipeline and returns the PDF as bytes.
// The context is used for cancellation and timeout.
func (s *Service) Convert(ctx context.Context, input Input) ([]byte, error) {
	if err := s.validateInput(input); err != nil {
		return nil, err
	}

	// Preprocess markdown
	mdContent := s.preprocessor.PreprocessMarkdown(ctx, input.Markdown)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Convert to HTML
	htmlContent, err := s.htmlConverter.ToHTML(ctx, mdContent)
	if err != nil {
		return nil, fmt.Errorf("converting to HTML: %w", err)
	}

	// Build combined CSS (watermark + user CSS)
	cssContent := input.CSS
	if input.Watermark != nil {
		watermarkCSS := buildWatermarkCSS(input.Watermark)
		cssContent = watermarkCSS + cssContent
	}

	// Inject CSS
	htmlContent = s.cssInjector.InjectCSS(ctx, htmlContent, cssContent)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Inject cover (if provided)
	var cvData *coverData
	if input.Cover != nil {
		cvData = toCoverData(input.Cover)
	}
	htmlContent, err = s.coverInjector.InjectCover(ctx, htmlContent, cvData)
	if err != nil {
		return nil, fmt.Errorf("injecting cover: %w", err)
	}

	// Inject signature (if provided)
	var sigData *signatureData
	if input.Signature != nil {
		sigData = toSignatureData(input.Signature)
	}
	htmlContent, err = s.signatureInjector.InjectSignature(ctx, htmlContent, sigData)
	if err != nil {
		return nil, fmt.Errorf("injecting signature: %w", err)
	}

	// Build PDF options with footer and page settings
	var footData *footerData
	if input.Footer != nil {
		footData = toFooterData(input.Footer)
	}
	pdfOpts := &pdfOptions{
		Footer: footData,
		Page:   input.Page,
	}

	// Convert to PDF
	pdfBytes, err := s.pdfConverter.ToPDF(ctx, htmlContent, pdfOpts)
	if err != nil {
		return nil, fmt.Errorf("converting to PDF: %w", err)
	}

	return pdfBytes, nil
}

// Close releases resources (headless Chrome browser).
func (s *Service) Close() error {
	if s.pdfConverter != nil {
		return s.pdfConverter.Close()
	}
	return nil
}

// validateInput checks that required fields are present and valid.
func (s *Service) validateInput(input Input) error {
	if input.Markdown == "" {
		return ErrEmptyMarkdown
	}
	if err := input.Page.Validate(); err != nil {
		return err
	}
	if err := input.Footer.Validate(); err != nil {
		return err
	}
	if err := input.Watermark.Validate(); err != nil {
		return err
	}
	if err := input.Cover.Validate(); err != nil {
		return err
	}
	return nil
}

// toSignatureData converts the public Signature type to internal signatureData.
func toSignatureData(sig *Signature) *signatureData {
	if sig == nil {
		return nil
	}
	links := make([]signatureLink, len(sig.Links))
	for i, l := range sig.Links {
		links[i] = signatureLink(l)
	}
	return &signatureData{
		Name:      sig.Name,
		Title:     sig.Title,
		Email:     sig.Email,
		ImagePath: sig.ImagePath,
		Links:     links,
	}
}

// toFooterData converts the public Footer type to internal footerData.
func toFooterData(f *Footer) *footerData {
	if f == nil {
		return nil
	}
	return &footerData{
		Position:       f.Position,
		ShowPageNumber: f.ShowPageNumber,
		Date:           f.Date,
		Status:         f.Status,
		Text:           f.Text,
	}
}

// toCoverData converts the public Cover type to internal coverData.
func toCoverData(c *Cover) *coverData {
	if c == nil {
		return nil
	}
	return &coverData{
		Title:        c.Title,
		Subtitle:     c.Subtitle,
		Logo:         c.Logo,
		Author:       c.Author,
		AuthorTitle:  c.AuthorTitle,
		Organization: c.Organization,
		Date:         c.Date,
		Version:      c.Version,
	}
}
