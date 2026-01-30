package md2pdf

import (
	"context"
	"fmt"
	"os"

	"github.com/alnah/go-md2pdf/internal/assets"
	"github.com/alnah/go-md2pdf/internal/fileutil"
	"github.com/alnah/go-md2pdf/internal/pipeline"
)

// Compile-time interface implementation checks.
// These ensure implementations satisfy their interfaces at compile time,
// catching signature mismatches before runtime.
var (
	_ pipeline.MarkdownPreprocessor = (*pipeline.CommonMarkPreprocessor)(nil)
	_ pipeline.HTMLConverter        = (*pipeline.GoldmarkConverter)(nil)
	_ pipeline.CSSInjector          = (*pipeline.CSSInjection)(nil)
	_ pipeline.CoverInjector        = (*pipeline.CoverInjection)(nil)
	_ pipeline.TOCInjector          = (*pipeline.TOCInjection)(nil)
	_ pipeline.SignatureInjector    = (*pipeline.SignatureInjection)(nil)
	_ pdfConverter                  = (*rodConverter)(nil)
	_ pdfRenderer                   = (*rodRenderer)(nil)
)

// Converter orchestrates the markdown-to-PDF conversion pipeline.
// Create with New() or NewConverter(), use Convert() for conversion, and Close() when done.
type Converter struct {
	cfg               converterConfig
	assetLoader       assets.AssetLoader // internal loader (for backward compat)
	publicAssetLoader AssetLoader        // public loader (from WithAssetLoader)
	preprocessor      pipeline.MarkdownPreprocessor
	htmlConverter     pipeline.HTMLConverter
	cssInjector       pipeline.CSSInjector
	coverInjector     pipeline.CoverInjector
	tocInjector       pipeline.TOCInjector
	signatureInjector pipeline.SignatureInjector
	pdfConverter      pdfConverter
}

// Service is an alias for Converter for backward compatibility.
//
// Deprecated: Use Converter instead. This alias will be removed in v2.
type Service = Converter

// publicToInternalAdapter wraps public AssetLoader to internal assets.AssetLoader.
type publicToInternalAdapter struct {
	pub AssetLoader
}

func (a *publicToInternalAdapter) LoadStyle(name string) (string, error) {
	return a.pub.LoadStyle(name)
}

func (a *publicToInternalAdapter) LoadTemplateSet(name string) (*assets.TemplateSet, error) {
	ts, err := a.pub.LoadTemplateSet(name)
	if err != nil {
		return nil, err
	}
	return &assets.TemplateSet{
		Name:      ts.Name,
		Cover:     ts.Cover,
		Signature: ts.Signature,
	}, nil
}

// New creates a Converter with default configuration.
// Use options to customize behavior (e.g., WithTimeout, WithAssetLoader, WithTemplateSet).
// Returns error if asset loading or template parsing fails.
//
// Deprecated: Use NewConverter instead. New will be removed in v2.
func New(opts ...Option) (*Converter, error) {
	return NewConverter(opts...)
}

// NewConverter creates a Converter with default configuration.
// Use options to customize behavior (e.g., WithTimeout, WithAssetLoader, WithTemplateSet).
// Returns error if asset loading or template parsing fails.
func NewConverter(opts ...Option) (*Converter, error) {
	c := &Converter{
		cfg:           converterConfig{timeout: defaultTimeout},
		assetLoader:   assets.NewEmbeddedLoader(),
		preprocessor:  &pipeline.CommonMarkPreprocessor{},
		htmlConverter: pipeline.NewGoldmarkConverter(),
		cssInjector:   &pipeline.CSSInjection{},
		tocInjector:   pipeline.NewTOCInjection(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Handle WithAssetPath: resolve to internal loader
	if c.cfg.assetPath != "" {
		resolver, err := assets.NewAssetResolver(c.cfg.assetPath)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAssetPath, err)
		}
		c.assetLoader = resolver
	}

	// Handle WithAssetLoader (public interface): wrap to internal interface
	if c.publicAssetLoader != nil {
		c.assetLoader = &publicToInternalAdapter{pub: c.publicAssetLoader}
	}

	// Resolve style input (name, path, or CSS content) to CSS content
	if err := c.resolveStyle(); err != nil {
		return nil, err
	}

	// Load template set if not already configured via WithTemplateSet
	var templateSet *assets.TemplateSet
	if c.cfg.templateSet != nil {
		templateSet = c.cfg.templateSet
	} else {
		// Load default template set
		var err error
		templateSet, err = c.assetLoader.LoadTemplateSet(assets.DefaultTemplateSetName)
		if err != nil {
			return nil, fmt.Errorf("loading default template set: %w", err)
		}
	}

	// Create injectors using template content (if not injected by tests)
	var err error
	if c.coverInjector == nil {
		c.coverInjector, err = pipeline.NewCoverInjection(templateSet.Cover)
		if err != nil {
			return nil, fmt.Errorf("initializing cover injector: %w", err)
		}
	}

	if c.signatureInjector == nil {
		c.signatureInjector, err = pipeline.NewSignatureInjection(templateSet.Signature)
		if err != nil {
			return nil, fmt.Errorf("initializing signature injector: %w", err)
		}
	}

	// Create PDF converter if not injected (e.g., by tests)
	if c.pdfConverter == nil {
		c.pdfConverter = newRodConverter(c.cfg.timeout)
	}

	return c, nil
}

// Convert runs the full pipeline and returns the result containing HTML and PDF.
// The context is used for cancellation and timeout.
// If input.HTMLOnly is true, PDF generation is skipped (for debugging).
// Recovers from internal panics to prevent crashes from propagating to callers.
func (c *Converter) Convert(ctx context.Context, input Input) (result *ConvertResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
		}
	}()

	if err := c.validateInput(input); err != nil {
		return nil, err
	}

	// Preprocess markdown
	mdContent := c.preprocessor.PreprocessMarkdown(ctx, input.Markdown)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Convert to HTML
	htmlContent, err := c.htmlConverter.ToHTML(ctx, mdContent)
	if err != nil {
		return nil, fmt.Errorf("converting to HTML: %w", err)
	}

	// Rewrite relative paths to absolute file:// URLs (if source directory provided)
	if input.SourceDir != "" {
		htmlContent, err = pipeline.RewriteRelativePaths(htmlContent, input.SourceDir)
		if err != nil {
			return nil, fmt.Errorf("rewriting relative paths: %w", err)
		}
	}

	// Convert highlight placeholders to <mark> tags.
	// This completes the ==text== feature started in preprocessing.
	// Done after Goldmark to avoid needing html.WithUnsafe().
	htmlContent = pipeline.ConvertMarkPlaceholders(htmlContent)

	// Build combined CSS (converter style + page breaks + watermark + user CSS)
	// Order matters: converter style first (base), user CSS last (can override)
	cssContent := c.cfg.resolvedStyle
	if input.CSS != "" {
		cssContent += "\n" + input.CSS
	}
	if input.Watermark != nil {
		watermarkCSS := buildWatermarkCSS(input.Watermark)
		cssContent = watermarkCSS + cssContent
	}
	// Page breaks CSS always generated (includes hardcoded rules + configurable)
	pageBreaksCSS := buildPageBreaksCSS(input.PageBreaks)
	cssContent = pageBreaksCSS + cssContent

	// Inject CSS
	htmlContent = c.cssInjector.InjectCSS(ctx, htmlContent, cssContent)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Inject cover (if provided)
	var cvData *pipeline.CoverData
	if input.Cover != nil {
		cvData = toCoverData(input.Cover)
	}
	htmlContent, err = c.coverInjector.InjectCover(ctx, htmlContent, cvData)
	if err != nil {
		return nil, fmt.Errorf("injecting cover: %w", err)
	}

	// Inject TOC (if provided) - must be after cover
	var tData *pipeline.TOCData
	if input.TOC != nil {
		tData = toTOCData(input.TOC)
	}
	htmlContent, err = c.tocInjector.InjectTOC(ctx, htmlContent, tData)
	if err != nil {
		return nil, fmt.Errorf("injecting TOC: %w", err)
	}

	// Inject signature (if provided)
	var sigData *pipeline.SignatureData
	if input.Signature != nil {
		sigData = toSignatureData(input.Signature)
	}
	htmlContent, err = c.signatureInjector.InjectSignature(ctx, htmlContent, sigData)
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
	var footData *pipeline.FooterData
	if input.Footer != nil {
		footData = toFooterData(input.Footer)
	}
	pdfOpts := &pdfOptions{
		Footer: footData,
		Page:   input.Page,
	}

	// Convert to PDF
	pdfBytes, err := c.pdfConverter.ToPDF(ctx, htmlContent, pdfOpts)
	if err != nil {
		return nil, fmt.Errorf("converting to PDF: %w", err)
	}

	res.PDF = pdfBytes
	return res, nil
}

// Close releases resources (headless Chrome browser).
func (c *Converter) Close() error {
	if c.pdfConverter != nil {
		return c.pdfConverter.Close()
	}
	return nil
}

// resolveStyle resolves the style input (name, path, or CSS content) to CSS content.
// Called during New() after options are applied and asset loader is configured.
func (c *Converter) resolveStyle() error {
	input := c.cfg.styleInput
	if input == "" {
		return nil // no style specified, use default from loader if needed
	}

	// File path? (contains / or \)
	if fileutil.IsFilePath(input) {
		content, err := os.ReadFile(input) // #nosec G304 -- user-provided path
		if err != nil {
			return fmt.Errorf("loading style file %q: %w", input, err)
		}
		c.cfg.resolvedStyle = string(content)
		return nil
	}

	// CSS content? (contains {)
	if fileutil.IsCSS(input) {
		c.cfg.resolvedStyle = input
		return nil
	}

	// Style name -> use asset loader
	css, err := c.assetLoader.LoadStyle(input)
	if err != nil {
		return fmt.Errorf("loading style %q: %w", input, err)
	}
	c.cfg.resolvedStyle = css
	return nil
}

// validateInput checks that required fields are present and valid.
//
// This is a TRUST BOUNDARY for direct library users who build Input manually.
// CLI users have their input validated earlier by Config.Validate() at config load time.
// Both paths converge here, ensuring all inputs are validated before processing.
func (c *Converter) validateInput(input Input) error {
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
	if err := input.Signature.Validate(); err != nil {
		return err
	}
	return nil
}

// toSignatureData converts the public Signature type to internal pipeline.SignatureData.
func toSignatureData(sig *Signature) *pipeline.SignatureData {
	if sig == nil {
		return nil
	}
	links := make([]pipeline.SignatureLink, len(sig.Links))
	for i, l := range sig.Links {
		links[i] = pipeline.SignatureLink(l)
	}
	return &pipeline.SignatureData{
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

// toFooterData converts the public Footer type to internal pipeline.FooterData.
func toFooterData(f *Footer) *pipeline.FooterData {
	if f == nil {
		return nil
	}
	return &pipeline.FooterData{
		Position:       f.Position,
		ShowPageNumber: f.ShowPageNumber,
		Date:           f.Date,
		Status:         f.Status,
		Text:           f.Text,
		DocumentID:     f.DocumentID,
	}
}

// toCoverData converts the public Cover type to internal pipeline.CoverData.
func toCoverData(c *Cover) *pipeline.CoverData {
	if c == nil {
		return nil
	}
	return &pipeline.CoverData{
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

// toTOCData converts the public TOC type to internal pipeline.TOCData.
func toTOCData(t *TOC) *pipeline.TOCData {
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
	return &pipeline.TOCData{
		Title:    t.Title,
		MinDepth: minDepth,
		MaxDepth: maxDepth,
	}
}
