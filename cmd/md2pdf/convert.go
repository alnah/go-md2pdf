package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
	"github.com/alnah/go-md2pdf/internal/dateutil"
	"github.com/alnah/go-md2pdf/internal/fileutil"
)

// runConvert orchestrates the conversion process.
// Config is accessed via env.Config (loaded once in runConvertCmd).
func runConvert(ctx context.Context, positionalArgs []string, flags *convertFlags, pool Pool, env *Environment) error {
	cfg := env.Config

	// Merge CLI flags into config (CLI wins)
	mergeFlags(flags, cfg)

	// Resolve "auto" date once for entire batch
	resolvedDate, err := resolveDateWithTime(cfg.Document.Date, env.Now)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}
	cfg.Document.Date = resolvedDate

	// Resolve input path
	inputPath, err := resolveInputPath(positionalArgs, cfg)
	if err != nil {
		return err
	}

	// Resolve output directory
	outputDir := resolveOutputDir(flags.output, cfg)

	// Discover files to convert
	files, err := discoverFiles(inputPath, outputDir)
	if err != nil {
		return fmt.Errorf("discovering files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no markdown files found in %s", inputPath)
	}

	// Resolve CSS content using the asset loader
	cssContent, err := resolveCSSContent(flags.assets.style, cfg, flags.assets.noStyle, env.AssetLoader)
	if err != nil {
		return err
	}

	// Build signature data (uses cfg.Author.*)
	sigData, err := buildSignatureData(cfg, flags.signature.disabled)
	if err != nil {
		return err
	}

	// Build footer data (uses cfg.Document.Date, cfg.Document.Version)
	footerData := buildFooterData(cfg, flags.footer.disabled)

	// Build page settings
	pageData, err := buildPageSettings(cfg)
	if err != nil {
		return err
	}

	// Build watermark data
	watermarkData, err := buildWatermarkData(cfg)
	if err != nil {
		return err
	}

	// Build TOC data
	tocData, err := buildTOCData(cfg, flags.toc)
	if err != nil {
		return err
	}

	// Build page breaks data
	pageBreaksData := buildPageBreaksData(cfg)

	// Bundle conversion parameters
	params := &conversionParams{
		css:        cssContent,
		footer:     footerData,
		signature:  sigData,
		page:       pageData,
		watermark:  watermarkData,
		toc:        tocData,
		pageBreaks: pageBreaksData,
		cfg:        cfg,
		htmlOnly:   flags.outputMode.htmlOnly,
		htmlOutput: flags.outputMode.html,
	}

	// Convert files
	results := convertBatch(ctx, pool, files, params)

	// Print results
	failedCount := printResultsWithWriter(results, flags.common.quiet, flags.common.verbose, env)
	if failedCount > 0 {
		return fmt.Errorf("%d conversion(s) failed", failedCount)
	}

	return nil
}

// parseBreakBefore parses "--break-before=h1,h2,h3" into individual bools.
func parseBreakBefore(value string) (h1, h2, h3 bool) {
	if value == "" {
		return false, false, false
	}
	parts := strings.Split(strings.ToLower(value), ",")
	for _, p := range parts {
		switch strings.TrimSpace(p) {
		case "h1":
			h1 = true
		case "h2":
			h2 = true
		case "h3":
			h3 = true
		}
	}
	return h1, h2, h3
}

// mergeFlags merges CLI flags into config. CLI values override config values.
func mergeFlags(flags *convertFlags, cfg *config.Config) {
	// Author flags
	if flags.author.name != "" {
		cfg.Author.Name = flags.author.name
	}
	if flags.author.title != "" {
		cfg.Author.Title = flags.author.title
	}
	if flags.author.email != "" {
		cfg.Author.Email = flags.author.email
	}
	if flags.author.org != "" {
		cfg.Author.Organization = flags.author.org
	}
	if flags.author.phone != "" {
		cfg.Author.Phone = flags.author.phone
	}
	if flags.author.address != "" {
		cfg.Author.Address = flags.author.address
	}
	if flags.author.department != "" {
		cfg.Author.Department = flags.author.department
	}

	// Document flags
	if flags.document.title != "" {
		cfg.Document.Title = flags.document.title
	}
	if flags.document.subtitle != "" {
		cfg.Document.Subtitle = flags.document.subtitle
	}
	if flags.document.version != "" {
		cfg.Document.Version = flags.document.version
	}
	if flags.document.date != "" {
		cfg.Document.Date = flags.document.date
	}
	if flags.document.clientName != "" {
		cfg.Document.ClientName = flags.document.clientName
	}
	if flags.document.projectName != "" {
		cfg.Document.ProjectName = flags.document.projectName
	}
	if flags.document.documentType != "" {
		cfg.Document.DocumentType = flags.document.documentType
	}
	if flags.document.documentID != "" {
		cfg.Document.DocumentID = flags.document.documentID
	}
	if flags.document.description != "" {
		cfg.Document.Description = flags.document.description
	}

	// Footer flags - auto-enable when flags are provided
	if flags.footer.position != "" {
		cfg.Footer.Position = flags.footer.position
		cfg.Footer.Enabled = true
	}
	if flags.footer.text != "" {
		cfg.Footer.Text = flags.footer.text
		cfg.Footer.Enabled = true
	}
	if flags.footer.pageNumber {
		cfg.Footer.ShowPageNumber = true
		cfg.Footer.Enabled = true
	}
	if flags.footer.showDocumentID {
		cfg.Footer.ShowDocumentID = true
		cfg.Footer.Enabled = true
	}

	// Cover flags - auto-enable when flags are provided
	if flags.cover.logo != "" {
		cfg.Cover.Logo = flags.cover.logo
		cfg.Cover.Enabled = true
	}
	if flags.cover.showDepartment {
		cfg.Cover.ShowDepartment = true
		cfg.Cover.Enabled = true
	}

	// Signature flags - auto-enable when flags are provided
	if flags.signature.image != "" {
		cfg.Signature.ImagePath = flags.signature.image
		cfg.Signature.Enabled = true
	}

	// TOC flags - auto-enable when flags are provided
	if flags.toc.title != "" {
		cfg.TOC.Title = flags.toc.title
		cfg.TOC.Enabled = true
	}
	if flags.toc.minDepth > 0 {
		cfg.TOC.MinDepth = flags.toc.minDepth
		cfg.TOC.Enabled = true
	}
	if flags.toc.maxDepth > 0 {
		cfg.TOC.MaxDepth = flags.toc.maxDepth
		cfg.TOC.Enabled = true
	}

	// Watermark flags
	// Track if watermark was configured via config file (vs just CLI flags)
	configuredViaFile := cfg.Watermark.Enabled
	if flags.watermark.text != "" {
		cfg.Watermark.Text = flags.watermark.text
		cfg.Watermark.Enabled = true
	}
	if flags.watermark.color != "" {
		cfg.Watermark.Color = flags.watermark.color
	}
	if flags.watermark.opacity != 0 {
		cfg.Watermark.Opacity = flags.watermark.opacity
	}
	if flags.watermark.angle != watermarkAngleSentinel {
		cfg.Watermark.Angle = flags.watermark.angle
	} else if !configuredViaFile && cfg.Watermark.Enabled {
		// Only apply default angle if enabled purely by CLI flags (not config file)
		// Config file with Angle: 0 is considered intentional
		cfg.Watermark.Angle = md2pdf.DefaultWatermarkAngle
	}

	// Page flags
	if flags.page.size != "" {
		cfg.Page.Size = flags.page.size
	}
	if flags.page.orientation != "" {
		cfg.Page.Orientation = flags.page.orientation
	}
	if flags.page.margin > 0 {
		cfg.Page.Margin = flags.page.margin
	}

	// PageBreaks flags
	if flags.pageBreaks.breakBefore != "" {
		h1, h2, h3 := parseBreakBefore(flags.pageBreaks.breakBefore)
		cfg.PageBreaks.BeforeH1 = h1
		cfg.PageBreaks.BeforeH2 = h2
		cfg.PageBreaks.BeforeH3 = h3
		cfg.PageBreaks.Enabled = true
	}
	if flags.pageBreaks.orphans > 0 {
		cfg.PageBreaks.Orphans = flags.pageBreaks.orphans
	}
	if flags.pageBreaks.widows > 0 {
		cfg.PageBreaks.Widows = flags.pageBreaks.widows
	}

	// Disable flags
	if flags.footer.disabled {
		cfg.Footer.Enabled = false
	}
	if flags.cover.disabled {
		cfg.Cover.Enabled = false
	}
	if flags.signature.disabled {
		cfg.Signature.Enabled = false
	}
	if flags.toc.disabled {
		cfg.TOC.Enabled = false
	}
	if flags.watermark.disabled {
		cfg.Watermark.Enabled = false
	}
	if flags.pageBreaks.disabled {
		cfg.PageBreaks.Enabled = false
	}
}

// resolveDateWithTime resolves "auto" and "auto:FORMAT" to formatted date.
func resolveDateWithTime(date string, now func() time.Time) (string, error) {
	return dateutil.ResolveDate(date, now())
}

// resolveInputPath determines the input path from args or config.
func resolveInputPath(args []string, cfg *config.Config) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if cfg.Input.DefaultDir != "" {
		return cfg.Input.DefaultDir, nil
	}
	return "", ErrNoInput
}

// resolveOutputDir determines the output directory from flag or config.
func resolveOutputDir(flagOutput string, cfg *config.Config) string {
	if flagOutput != "" {
		return flagOutput
	}
	return cfg.Output.DefaultDir
}

// resolveTemplateSet resolves a template set from a name or path.
// If templateFlag is empty, loads the default template set.
// If templateFlag looks like a path, loads from the filesystem directory.
// Otherwise, treats it as a template set name and uses the loader.
func resolveTemplateSet(templateFlag string, loader md2pdf.AssetLoader) (*md2pdf.TemplateSet, error) {
	// Use default if not specified
	if templateFlag == "" {
		return loader.LoadTemplateSet(md2pdf.DefaultTemplateSet)
	}

	// If it looks like a path, load from filesystem directory
	if fileutil.IsFilePath(templateFlag) {
		return loadTemplateSetFromDir(templateFlag)
	}

	// Otherwise, treat as a template set name and use the loader
	return loader.LoadTemplateSet(templateFlag)
}

// loadTemplateSetFromDir loads cover.html and signature.html from a directory.
func loadTemplateSetFromDir(dirPath string) (*md2pdf.TemplateSet, error) {
	coverPath := filepath.Join(dirPath, "cover.html")
	sigPath := filepath.Join(dirPath, "signature.html")

	cover, coverErr := os.ReadFile(coverPath) // #nosec G304 -- user-provided path
	signature, sigErr := os.ReadFile(sigPath) // #nosec G304 -- user-provided path

	// If both files are missing, the directory is not a valid template set
	if os.IsNotExist(coverErr) && os.IsNotExist(sigErr) {
		return nil, fmt.Errorf("%w: %q (directory has no templates)", md2pdf.ErrTemplateSetNotFound, dirPath)
	}

	// Handle read errors (not just not-exist)
	if coverErr != nil && !os.IsNotExist(coverErr) {
		return nil, fmt.Errorf("reading cover.html: %w", coverErr)
	}
	if sigErr != nil && !os.IsNotExist(sigErr) {
		return nil, fmt.Errorf("reading signature.html: %w", sigErr)
	}

	// If only one file is missing, the template set is incomplete
	if os.IsNotExist(coverErr) {
		return nil, fmt.Errorf("%w: %q missing cover.html", md2pdf.ErrIncompleteTemplateSet, dirPath)
	}
	if os.IsNotExist(sigErr) {
		return nil, fmt.Errorf("%w: %q missing signature.html", md2pdf.ErrIncompleteTemplateSet, dirPath)
	}

	return md2pdf.NewTemplateSet(dirPath, string(cover), string(signature)), nil
}

// resolveCSSContent resolves CSS content from CLI flag, config, or asset loader.
// Priority: CLI flag > config style > default style.
// If the style value looks like a path (contains / or \), read it directly.
// Otherwise, treat it as a style name and use the asset loader.
func resolveCSSContent(styleFlag string, cfg *config.Config, noStyle bool, loader md2pdf.AssetLoader) (string, error) {
	if noStyle {
		return "", nil
	}

	// Determine which style to use: CLI flag > config > default
	style := styleFlag
	if style == "" && cfg != nil {
		style = cfg.Style
	}
	if style == "" {
		style = md2pdf.DefaultStyle
	}

	// If it looks like a path, read the file directly
	if fileutil.IsFilePath(style) {
		content, err := os.ReadFile(style) // #nosec G304 -- user-provided path
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrReadCSS, err)
		}
		return string(content), nil
	}

	// Otherwise, treat as a style name and use the loader
	return loader.LoadStyle(style)
}
