package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/assets"
	"github.com/alnah/go-md2pdf/internal/config"
)

// Sentinel errors for CLI operations.
var (
	ErrNoInput            = errors.New("no input specified")
	ErrReadCSS            = errors.New("failed to read CSS file")
	ErrReadMarkdown       = errors.New("failed to read markdown file")
	ErrWritePDF           = errors.New("failed to write PDF file")
	ErrInvalidExtension   = errors.New("file must have .md or .markdown extension")
	ErrSignatureImagePath = errors.New("signature image not found")
	ErrInvalidWorkerCount = errors.New("invalid worker count")
)

// File permission constants.
const (
	dirPermissions  = 0o750 // rwxr-x---: owner full, group read+execute
	filePermissions = 0o644 // rw-r--r--: owner read+write, others read
)

// Converter is the interface for the conversion service.
type Converter interface {
	Convert(ctx context.Context, input md2pdf.Input) (*md2pdf.ConvertResult, error)
}

// Compile-time interface implementation check.
var _ Converter = (*md2pdf.Service)(nil)

// Pool abstracts service pool operations for testability.
type Pool interface {
	Acquire() Converter
	Release(Converter)
	Size() int
}

// FileToConvert represents a single file to process.
type FileToConvert struct {
	InputPath  string
	OutputPath string
}

// ConversionResult holds the outcome of a single conversion.
type ConversionResult struct {
	InputPath  string
	OutputPath string
	Err        error
	Duration   time.Duration
}

// conversionParams groups parameters shared across batch/file conversion.
type conversionParams struct {
	css        string
	footer     *md2pdf.Footer
	signature  *md2pdf.Signature
	page       *md2pdf.PageSettings
	watermark  *md2pdf.Watermark
	toc        *md2pdf.TOC
	pageBreaks *md2pdf.PageBreaks
	cfg        *config.Config
	htmlOnly   bool // Output HTML only, skip PDF
	htmlOutput bool // Output HTML alongside PDF
}

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
	pageData, err := buildPageSettings(flags, cfg)
	if err != nil {
		return err
	}

	// Build watermark data
	watermarkData, err := buildWatermarkData(flags, cfg)
	if err != nil {
		return err
	}

	// Build TOC data
	tocData := buildTOCData(cfg, flags.toc)

	// Build page breaks data
	pageBreaksData := buildPageBreaksData(flags, cfg)

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
	if flags.toc.depth > 0 {
		cfg.TOC.MaxDepth = flags.toc.depth
		cfg.TOC.Enabled = true
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
}

// resolveDateWithTime resolves "auto" and "auto:FORMAT" to formatted date.
func resolveDateWithTime(date string, now func() time.Time) (string, error) {
	return md2pdf.ResolveDate(date, now())
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
func resolveTemplateSet(templateFlag string, loader assets.AssetLoader) (*assets.TemplateSet, error) {
	// Use default if not specified
	if templateFlag == "" {
		return loader.LoadTemplateSet(assets.DefaultTemplateSetName)
	}

	// If it looks like a path, load from filesystem directory
	if md2pdf.IsFilePath(templateFlag) {
		return loadTemplateSetFromDir(templateFlag)
	}

	// Otherwise, treat as a template set name and use the loader
	return loader.LoadTemplateSet(templateFlag)
}

// loadTemplateSetFromDir loads cover.html and signature.html from a directory.
func loadTemplateSetFromDir(dirPath string) (*assets.TemplateSet, error) {
	coverPath := filepath.Join(dirPath, "cover.html")
	sigPath := filepath.Join(dirPath, "signature.html")

	cover, coverErr := os.ReadFile(coverPath) // #nosec G304 -- user-provided path
	signature, sigErr := os.ReadFile(sigPath) // #nosec G304 -- user-provided path

	// If both files are missing, the directory is not a valid template set
	if os.IsNotExist(coverErr) && os.IsNotExist(sigErr) {
		return nil, fmt.Errorf("%w: %q (directory has no templates)", assets.ErrTemplateSetNotFound, dirPath)
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
		return nil, fmt.Errorf("%w: %q missing cover.html", assets.ErrIncompleteTemplateSet, dirPath)
	}
	if os.IsNotExist(sigErr) {
		return nil, fmt.Errorf("%w: %q missing signature.html", assets.ErrIncompleteTemplateSet, dirPath)
	}

	return &assets.TemplateSet{
		Name:      dirPath,
		Cover:     string(cover),
		Signature: string(signature),
	}, nil
}

// resolveCSSContent resolves CSS content from CLI flag, config, or asset loader.
// Priority: CLI flag > config style > default style.
// If the style value looks like a path (contains / or \), read it directly.
// Otherwise, treat it as a style name and use the asset loader.
func resolveCSSContent(styleFlag string, cfg *config.Config, noStyle bool, loader assets.AssetLoader) (string, error) {
	if noStyle {
		return "", nil
	}

	// Determine which style to use: CLI flag > config > default
	style := styleFlag
	if style == "" && cfg != nil {
		style = cfg.Style
	}
	if style == "" {
		style = assets.DefaultStyleName
	}

	// If it looks like a path, read the file directly
	if md2pdf.IsFilePath(style) {
		content, err := os.ReadFile(style) // #nosec G304 -- user-provided path
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrReadCSS, err)
		}
		return string(content), nil
	}

	// Otherwise, treat as a style name and use the loader
	return loader.LoadStyle(style)
}

// discoverFiles finds all markdown files to convert.
func discoverFiles(inputPath, outputDir string) ([]FileToConvert, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		if err := validateMarkdownExtension(inputPath); err != nil {
			return nil, err
		}
		outPath := resolveOutputPath(inputPath, outputDir, "")
		return []FileToConvert{{InputPath: inputPath, OutputPath: outPath}}, nil
	}

	var files []FileToConvert
	err = filepath.WalkDir(inputPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".markdown" {
			return nil
		}
		outPath := resolveOutputPath(path, outputDir, inputPath)
		files = append(files, FileToConvert{InputPath: path, OutputPath: outPath})
		return nil
	})

	return files, err
}

// resolveOutputPath determines the PDF output path for a markdown file.
func resolveOutputPath(inputPath, outputDir, baseInputDir string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	if outputDir == "" {
		return filepath.Join(filepath.Dir(inputPath), base+".pdf")
	}

	if strings.HasSuffix(outputDir, ".pdf") {
		return outputDir
	}

	if baseInputDir != "" {
		relPath, err := filepath.Rel(baseInputDir, inputPath)
		if err == nil {
			relDir := filepath.Dir(relPath)
			return filepath.Join(outputDir, relDir, base+".pdf")
		}
	}

	return filepath.Join(outputDir, base+".pdf")
}

// validateMarkdownExtension checks that the file has a .md or .markdown extension.
func validateMarkdownExtension(path string) error {
	ext := filepath.Ext(path)
	if ext != ".md" && ext != ".markdown" {
		return fmt.Errorf("%w: got %q", ErrInvalidExtension, ext)
	}
	return nil
}

// validateWorkers checks that the worker count is within valid bounds.
func validateWorkers(n int) error {
	if n < 0 {
		return fmt.Errorf("%w: %d (must be >= 0, 0 means auto)", ErrInvalidWorkerCount, n)
	}
	if n > md2pdf.MaxPoolSize {
		return fmt.Errorf("%w: %d (maximum is %d)", ErrInvalidWorkerCount, n, md2pdf.MaxPoolSize)
	}
	return nil
}

// buildSignatureData creates md2pdf.Signature from config.
// Uses cfg.Author.* for author information.
// Department is always shown if defined (signature always displays it).
func buildSignatureData(cfg *config.Config, noSignature bool) (*md2pdf.Signature, error) {
	if noSignature || !cfg.Signature.Enabled {
		return nil, nil
	}

	// Validate image path if set (and not a URL)
	if cfg.Signature.ImagePath != "" && !md2pdf.IsURL(cfg.Signature.ImagePath) {
		if !md2pdf.FileExists(cfg.Signature.ImagePath) {
			return nil, fmt.Errorf("%w: %s", ErrSignatureImagePath, cfg.Signature.ImagePath)
		}
	}

	// Convert config links to md2pdf.Link
	links := make([]md2pdf.Link, len(cfg.Signature.Links))
	for i, l := range cfg.Signature.Links {
		links[i] = md2pdf.Link{Label: l.Label, URL: l.URL}
	}

	return &md2pdf.Signature{
		Name:         cfg.Author.Name,
		Title:        cfg.Author.Title,
		Email:        cfg.Author.Email,
		Organization: cfg.Author.Organization,
		ImagePath:    cfg.Signature.ImagePath,
		Links:        links,
		Phone:        cfg.Author.Phone,
		Address:      cfg.Author.Address,
		Department:   cfg.Author.Department,
	}, nil
}

// buildFooterData creates md2pdf.Footer from config.
// Uses cfg.Document.Date and cfg.Document.Version for date/status.
// DocumentID is only shown if cfg.Footer.ShowDocumentID is true.
func buildFooterData(cfg *config.Config, noFooter bool) *md2pdf.Footer {
	if noFooter || !cfg.Footer.Enabled {
		return nil
	}

	var docID string
	if cfg.Footer.ShowDocumentID {
		docID = cfg.Document.DocumentID
	}

	return &md2pdf.Footer{
		Position:       cfg.Footer.Position,
		ShowPageNumber: cfg.Footer.ShowPageNumber,
		Date:           cfg.Document.Date,
		Status:         cfg.Document.Version,
		Text:           cfg.Footer.Text,
		DocumentID:     docID,
	}
}

// buildWatermarkData creates md2pdf.Watermark from flags and config.
func buildWatermarkData(flags *convertFlags, cfg *config.Config) (*md2pdf.Watermark, error) {
	if flags.watermark.disabled {
		return nil, nil
	}

	hasFlags := flags.watermark.text != ""
	hasConfig := cfg.Watermark.Enabled

	if !hasFlags && !hasConfig {
		return nil, nil
	}

	w := &md2pdf.Watermark{}
	if cfg.Watermark.Enabled {
		w.Text = cfg.Watermark.Text
		w.Color = cfg.Watermark.Color
		w.Opacity = cfg.Watermark.Opacity
		w.Angle = cfg.Watermark.Angle
	}

	// CLI flags override config
	if flags.watermark.text != "" {
		w.Text = flags.watermark.text
	}
	if flags.watermark.color != "" {
		w.Color = flags.watermark.color
	}
	if flags.watermark.opacity != 0 {
		w.Opacity = flags.watermark.opacity
	}
	if flags.watermark.angle != watermarkAngleSentinel {
		w.Angle = flags.watermark.angle
	}

	// Apply defaults
	if w.Color == "" {
		w.Color = md2pdf.DefaultWatermarkColor
	}
	if w.Opacity == 0 {
		w.Opacity = md2pdf.DefaultWatermarkOpacity
	}
	if shouldApplyDefaultAngle(flags.watermark.angle, cfg) {
		w.Angle = md2pdf.DefaultWatermarkAngle
	}

	// Validate
	if w.Text == "" {
		return nil, fmt.Errorf("watermark text is required when watermark is enabled")
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}

	return w, nil
}

// shouldApplyDefaultAngle returns true if the watermark angle should use default.
func shouldApplyDefaultAngle(flagAngle float64, cfg *config.Config) bool {
	flagNotSet := flagAngle == watermarkAngleSentinel
	configNotSet := cfg.Watermark.Angle == 0 && !cfg.Watermark.Enabled
	return flagNotSet && configNotSet
}

// buildPageSettings creates md2pdf.PageSettings from flags and config.
func buildPageSettings(flags *convertFlags, cfg *config.Config) (*md2pdf.PageSettings, error) {
	hasFlags := flags.page.size != "" || flags.page.orientation != "" || flags.page.margin > 0
	hasConfig := cfg.Page.Size != "" || cfg.Page.Orientation != "" || cfg.Page.Margin > 0

	if !hasFlags && !hasConfig {
		return nil, nil
	}

	ps := &md2pdf.PageSettings{
		Size:        cfg.Page.Size,
		Orientation: cfg.Page.Orientation,
		Margin:      cfg.Page.Margin,
	}

	// CLI flags override config
	if flags.page.size != "" {
		ps.Size = flags.page.size
	}
	if flags.page.orientation != "" {
		ps.Orientation = flags.page.orientation
	}
	if flags.page.margin > 0 {
		ps.Margin = flags.page.margin
	}

	// Apply defaults
	if ps.Size == "" {
		ps.Size = md2pdf.PageSizeLetter
	}
	if ps.Orientation == "" {
		ps.Orientation = md2pdf.OrientationPortrait
	}
	if ps.Margin == 0 {
		ps.Margin = md2pdf.DefaultMargin
	}

	if err := ps.Validate(); err != nil {
		return nil, err
	}

	return ps, nil
}

// firstHeadingPattern matches the first # heading in markdown content.
var firstHeadingPattern = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// extractFirstHeading extracts the first # heading from markdown content.
func extractFirstHeading(markdown string) string {
	matches := firstHeadingPattern.FindStringSubmatch(markdown)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// buildCoverData creates md2pdf.Cover from config and markdown content.
// Uses cfg.Author.* and cfg.Document.* for metadata.
// Department is only shown if cfg.Cover.ShowDepartment is true.
func buildCoverData(cfg *config.Config, markdownContent, filename string) (*md2pdf.Cover, error) {
	if !cfg.Cover.Enabled {
		return nil, nil
	}

	c := &md2pdf.Cover{
		Logo: cfg.Cover.Logo,
	}

	// Title: config → H1 → filename
	if cfg.Document.Title != "" {
		c.Title = cfg.Document.Title
	} else {
		c.Title = extractFirstHeading(markdownContent)
		if c.Title == "" {
			c.Title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
		}
	}

	c.Subtitle = cfg.Document.Subtitle
	c.Author = cfg.Author.Name
	c.AuthorTitle = cfg.Author.Title
	c.Organization = cfg.Author.Organization
	c.Date = cfg.Document.Date // Already resolved
	c.Version = cfg.Document.Version

	// Extended metadata fields
	c.ClientName = cfg.Document.ClientName
	c.ProjectName = cfg.Document.ProjectName
	c.DocumentType = cfg.Document.DocumentType
	c.DocumentID = cfg.Document.DocumentID
	c.Description = cfg.Document.Description

	// Department only if explicitly enabled on cover
	if cfg.Cover.ShowDepartment {
		c.Department = cfg.Author.Department
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// buildTOCData creates md2pdf.TOC from config.
func buildTOCData(cfg *config.Config, tocFlags tocFlags) *md2pdf.TOC {
	if tocFlags.disabled || !cfg.TOC.Enabled {
		return nil
	}

	maxDepth := cfg.TOC.MaxDepth
	if maxDepth == 0 {
		maxDepth = md2pdf.DefaultTOCMaxDepth
	}

	return &md2pdf.TOC{
		Title:    cfg.TOC.Title,
		MinDepth: cfg.TOC.MinDepth, // 0 = library defaults to 2
		MaxDepth: maxDepth,
	}
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

// buildPageBreaksData creates md2pdf.PageBreaks from flags and config.
func buildPageBreaksData(flags *convertFlags, cfg *config.Config) *md2pdf.PageBreaks {
	if flags.pageBreaks.disabled {
		return nil
	}

	pb := &md2pdf.PageBreaks{
		Orphans: md2pdf.DefaultOrphans,
		Widows:  md2pdf.DefaultWidows,
	}

	if cfg.PageBreaks.Enabled {
		pb.BeforeH1 = cfg.PageBreaks.BeforeH1
		pb.BeforeH2 = cfg.PageBreaks.BeforeH2
		pb.BeforeH3 = cfg.PageBreaks.BeforeH3
		if cfg.PageBreaks.Orphans > 0 {
			pb.Orphans = cfg.PageBreaks.Orphans
		}
		if cfg.PageBreaks.Widows > 0 {
			pb.Widows = cfg.PageBreaks.Widows
		}
	}

	// CLI flags override config
	if flags.pageBreaks.breakBefore != "" {
		h1, h2, h3 := parseBreakBefore(flags.pageBreaks.breakBefore)
		pb.BeforeH1 = h1
		pb.BeforeH2 = h2
		pb.BeforeH3 = h3
	}
	if flags.pageBreaks.orphans > 0 {
		pb.Orphans = flags.pageBreaks.orphans
	}
	if flags.pageBreaks.widows > 0 {
		pb.Widows = flags.pageBreaks.widows
	}

	return pb
}

// ErrServiceInit indicates the conversion service failed to initialize.
var ErrServiceInit = errors.New("failed to initialize conversion service")

// convertBatch processes files concurrently using the service pool.
func convertBatch(ctx context.Context, pool Pool, files []FileToConvert, params *conversionParams) []ConversionResult {
	if len(files) == 0 {
		return nil
	}

	concurrency := pool.Size()
	if concurrency > len(files) {
		concurrency = len(files)
	}

	results := make([]ConversionResult, len(files))
	var wg sync.WaitGroup
	jobs := make(chan int, len(files))

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			svc := pool.Acquire()
			if svc == nil {
				// Service creation failed, mark remaining jobs as failed
				for idx := range jobs {
					results[idx] = ConversionResult{
						InputPath: files[idx].InputPath,
						Err:       ErrServiceInit,
					}
				}
				return
			}
			defer pool.Release(svc)

			for idx := range jobs {
				if ctx.Err() != nil {
					results[idx] = ConversionResult{
						InputPath: files[idx].InputPath,
						Err:       ctx.Err(),
					}
					continue
				}
				results[idx] = convertFile(ctx, svc, files[idx], params)
			}
		}()
	}

	for i := range files {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	return results
}

// convertFile processes a single file and returns the result.
func convertFile(ctx context.Context, service Converter, f FileToConvert, params *conversionParams) ConversionResult {
	start := time.Now()
	result := ConversionResult{
		InputPath:  f.InputPath,
		OutputPath: f.OutputPath,
	}

	content, err := os.ReadFile(f.InputPath) // #nosec G304 -- discovered path
	if err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrReadMarkdown, err)
		result.Duration = time.Since(start)
		return result
	}

	// Build cover data (depends on markdown content for H1 extraction)
	coverData, err := buildCoverData(params.cfg, string(content), f.InputPath)
	if err != nil {
		result.Err = fmt.Errorf("building cover data: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	outDir := filepath.Dir(f.OutputPath)
	if err := os.MkdirAll(outDir, dirPermissions); err != nil {
		result.Err = fmt.Errorf("creating output directory: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	convResult, err := service.Convert(ctx, md2pdf.Input{
		Markdown:   string(content),
		CSS:        params.css,
		Footer:     params.footer,
		Signature:  params.signature,
		Page:       params.page,
		Watermark:  params.watermark,
		Cover:      coverData,
		TOC:        params.toc,
		PageBreaks: params.pageBreaks,
		HTMLOnly:   params.htmlOnly,
	})
	if err != nil {
		result.Err = err
		result.Duration = time.Since(start)
		return result
	}

	// Write HTML output if requested (--html or --html-only)
	if params.htmlOnly || params.htmlOutput {
		htmlPath := htmlOutputPath(f.OutputPath)
		// #nosec G306 -- HTML files are meant to be readable
		if err := os.WriteFile(htmlPath, convResult.HTML, filePermissions); err != nil {
			result.Err = fmt.Errorf("failed to write HTML file: %w", err)
			result.Duration = time.Since(start)
			return result
		}
		// For --html-only, update output path to HTML file
		if params.htmlOnly {
			result.OutputPath = htmlPath
			result.Duration = time.Since(start)
			return result
		}
	}

	// Write PDF (unless --html-only)
	// #nosec G306 -- PDFs are meant to be readable
	if err := os.WriteFile(f.OutputPath, convResult.PDF, filePermissions); err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrWritePDF, err)
		result.Duration = time.Since(start)
		return result
	}

	result.Duration = time.Since(start)
	return result
}

// htmlOutputPath returns the HTML path corresponding to a PDF path.
func htmlOutputPath(pdfPath string) string {
	return strings.TrimSuffix(pdfPath, ".pdf") + ".html"
}

// ResultSummary holds the count of succeeded and failed conversions.
type ResultSummary struct {
	Succeeded int
	Failed    int
}

// countResults tallies succeeded and failed conversions.
func countResults(results []ConversionResult) ResultSummary {
	var summary ResultSummary
	for _, r := range results {
		if r.Err != nil {
			summary.Failed++
		} else {
			summary.Succeeded++
		}
	}
	return summary
}

// printResultsWithWriter outputs conversion results using the provided writers.
func printResultsWithWriter(results []ConversionResult, quiet, verbose bool, env *Environment) int {
	summary := countResults(results)

	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(env.Stderr, "FAILED %s: %v\n", r.InputPath, r.Err)
			continue
		}

		if quiet {
			continue
		}

		if verbose {
			fmt.Fprintf(env.Stdout, "%s -> %s (%v)\n", r.InputPath, r.OutputPath, r.Duration.Round(time.Millisecond))
		} else {
			fmt.Fprintf(env.Stdout, "Created %s\n", r.OutputPath)
		}
	}

	if !quiet && len(results) > 1 {
		fmt.Fprintf(env.Stdout, "\n%d succeeded, %d failed\n", summary.Succeeded, summary.Failed)
	}

	return summary.Failed
}
