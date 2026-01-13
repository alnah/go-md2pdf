package md2pdf

import (
	"context"
	"fmt"

	"github.com/alnah/go-md2pdf/internal/assets"
)

// Compile-time interface implementation checks.
// These ensure implementations satisfy their interfaces at compile time,
// catching signature mismatches before runtime.
var (
	_ markdownPreprocessor = (*commonMarkPreprocessor)(nil)
	_ htmlConverter        = (*goldmarkConverter)(nil)
	_ cssInjector          = (*cssInjection)(nil)
	_ coverInjector        = (*coverInjection)(nil)
	_ tocInjector          = (*tocInjection)(nil)
	_ signatureInjector    = (*signatureInjection)(nil)
	_ pdfConverter         = (*rodConverter)(nil)
	_ pdfRenderer          = (*rodRenderer)(nil)
)

// Service orchestrates the markdown-to-PDF pipeline.
type Service struct {
	cfg               serviceConfig
	assetLoader       assets.AssetLoader
	preprocessor      markdownPreprocessor
	htmlConverter     htmlConverter
	cssInjector       cssInjector
	coverInjector     coverInjector
	tocInjector       tocInjector
	signatureInjector signatureInjector
	pdfConverter      pdfConverter
}

// New creates a Service with default configuration.
// Use options to customize behavior (e.g., WithTimeout, WithAssetLoader, WithTemplateSet).
// Returns error if asset loading or template parsing fails.
func New(opts ...Option) (*Service, error) {
	s := &Service{
		cfg:           serviceConfig{timeout: defaultTimeout},
		assetLoader:   assets.NewEmbeddedLoader(),
		preprocessor:  &commonMarkPreprocessor{},
		htmlConverter: newGoldmarkConverter(),
		cssInjector:   &cssInjection{},
		tocInjector:   newTOCInjection(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Load template set if not already configured via WithTemplateSet
	var templateSet *assets.TemplateSet
	if s.cfg.templateSet != nil {
		templateSet = s.cfg.templateSet
	} else {
		// Load default template set
		var err error
		templateSet, err = s.assetLoader.LoadTemplateSet(assets.DefaultTemplateSetName)
		if err != nil {
			return nil, fmt.Errorf("loading default template set: %w", err)
		}
	}

	// Create injectors using template content (if not injected by tests)
	var err error
	if s.coverInjector == nil {
		s.coverInjector, err = newCoverInjection(templateSet.Cover)
		if err != nil {
			return nil, fmt.Errorf("initializing cover injector: %w", err)
		}
	}

	if s.signatureInjector == nil {
		s.signatureInjector, err = newSignatureInjection(templateSet.Signature)
		if err != nil {
			return nil, fmt.Errorf("initializing signature injector: %w", err)
		}
	}

	// Create PDF converter if not injected (e.g., by tests)
	if s.pdfConverter == nil {
		s.pdfConverter = newRodConverter(s.cfg.timeout)
	}

	return s, nil
}

// Convert runs the full pipeline and returns the result containing HTML and PDF.
// The context is used for cancellation and timeout.
// If input.HTMLOnly is true, PDF generation is skipped (for debugging).
// Recovers from internal panics to prevent crashes from propagating to callers.
func (s *Service) Convert(ctx context.Context, input Input) (result *ConvertResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
		}
	}()

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

	// Convert highlight placeholders to <mark> tags.
	// This completes the ==text== feature started in preprocessing.
	// Done after Goldmark to avoid needing html.WithUnsafe().
	htmlContent = convertMarkPlaceholders(htmlContent)

	// Build combined CSS (page breaks + watermark + user CSS)
	// Order matters: page breaks first (lowest priority), user CSS last (can override)
	cssContent := input.CSS
	if input.Watermark != nil {
		watermarkCSS := buildWatermarkCSS(input.Watermark)
		cssContent = watermarkCSS + cssContent
	}
	// Page breaks CSS always generated (includes hardcoded rules + configurable)
	pageBreaksCSS := buildPageBreaksCSS(input.PageBreaks)
	cssContent = pageBreaksCSS + cssContent

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

	// Inject TOC (if provided) - must be after cover
	var tData *tocData
	if input.TOC != nil {
		tData = toTOCData(input.TOC)
	}
	htmlContent, err = s.tocInjector.InjectTOC(ctx, htmlContent, tData)
	if err != nil {
		return nil, fmt.Errorf("injecting TOC: %w", err)
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

	// Prepare result with HTML
	res := &ConvertResult{
		HTML: []byte(htmlContent),
	}

	// Skip PDF generation if HTMLOnly mode
	if input.HTMLOnly {
		return res, nil
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

	res.PDF = pdfBytes
	return res, nil
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
	if err := input.TOC.Validate(); err != nil {
		return err
	}
	if err := input.PageBreaks.Validate(); err != nil {
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
		Name:         sig.Name,
		Title:        sig.Title,
		Email:        sig.Email,
		Organization: sig.Organization,
		ImagePath:    sig.ImagePath,
		Links:        links,
		Phone:        sig.Phone,
		Address:      sig.Address,
		Department:   sig.Department,
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
		DocumentID:     f.DocumentID,
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
		ClientName:   c.ClientName,
		ProjectName:  c.ProjectName,
		DocumentType: c.DocumentType,
		DocumentID:   c.DocumentID,
		Description:  c.Description,
		Department:   c.Department,
	}
}

// toTOCData converts the public TOC type to internal tocData.
func toTOCData(t *TOC) *tocData {
	if t == nil {
		return nil
	}
	minDepth := t.MinDepth
	if minDepth == 0 {
		minDepth = DefaultTOCMinDepth
	}
	maxDepth := t.MaxDepth
	if maxDepth == 0 {
		maxDepth = DefaultTOCMaxDepth
	}
	return &tocData{
		Title:    t.Title,
		MinDepth: minDepth,
		MaxDepth: maxDepth,
	}
}
