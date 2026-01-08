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
	flag "github.com/spf13/pflag"
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

// maxWorkers limits parallel browser instances to prevent resource exhaustion.
// Each Chrome instance uses ~100-200MB RAM; 32 workers cap memory at ~6GB.
const maxWorkers = 32

// watermarkAngleSentinel is used to detect if --watermark-angle was explicitly set.
// We need a sentinel because 0 is a valid angle value.
const watermarkAngleSentinel = -999.0

// Converter is the interface for the conversion service.
type Converter interface {
	Convert(ctx context.Context, input md2pdf.Input) ([]byte, error)
}

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

// cliFlags holds parsed command-line flags.
type cliFlags struct {
	configName       string
	outputPath       string
	cssFile          string
	quiet            bool
	verbose          bool
	noSignature      bool
	noStyle          bool
	noFooter         bool
	noWatermark      bool
	noCover          bool
	coverTitle       string
	version          bool
	workers          int
	pageSize         string
	orientation      string
	margin           float64
	watermarkText    string
	watermarkColor   string
	watermarkOpacity float64
	watermarkAngle   float64
}

// run parses arguments, discovers files, and orchestrates batch conversion.
// The context is used for cancellation (e.g., on SIGINT/SIGTERM).
func run(ctx context.Context, args []string, pool Pool) error {
	flags, positionalArgs, err := parseFlags(args)
	if err != nil {
		return err
	}

	// Validate worker count early for CLI feedback
	if err := validateWorkers(flags.workers); err != nil {
		return err
	}

	// Load configuration
	cfg := config.DefaultConfig()
	if flags.configName != "" {
		cfg, err = config.LoadConfig(flags.configName)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}

	// Resolve input path
	inputPath, err := resolveInputPath(positionalArgs, cfg)
	if err != nil {
		return err
	}

	// Resolve output directory
	outputDir := resolveOutputDir(flags.outputPath, cfg)

	// Discover files to convert
	files, err := discoverFiles(inputPath, outputDir)
	if err != nil {
		return fmt.Errorf("discovering files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no markdown files found in %s", inputPath)
	}

	// Resolve CSS content
	cssContent, err := resolveCSSContent(flags.cssFile, cfg, flags.noStyle)
	if err != nil {
		return err
	}

	// Build signature data
	sigData, err := buildSignatureData(cfg, flags.noSignature)
	if err != nil {
		return err
	}

	// Build footer data
	footerData := buildFooterData(cfg, flags.noFooter)

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

	// Convert files
	results := convertBatch(ctx, pool, files, cssContent, footerData, sigData, pageData, watermarkData, flags, cfg)

	// Print results and return appropriate exit code
	failedCount := printResults(results, flags.quiet, flags.verbose)
	if failedCount > 0 {
		return fmt.Errorf("%d conversion(s) failed", failedCount)
	}

	return nil
}

// parseFlags parses command-line flags and returns remaining positional arguments.
// Supports GNU-style flags (--flag, -f) and flags after positional arguments.
func parseFlags(args []string) (*cliFlags, []string, error) {
	flagSet := flag.NewFlagSet("md2pdf", flag.ContinueOnError)

	flags := &cliFlags{}
	flagSet.StringVarP(&flags.configName, "config", "c", "", "config name or path")
	flagSet.StringVarP(&flags.outputPath, "output", "o", "", "output file or directory")
	flagSet.StringVar(&flags.cssFile, "css", "", "CSS file for styling")
	flagSet.BoolVarP(&flags.quiet, "quiet", "q", false, "only show errors")
	flagSet.BoolVarP(&flags.verbose, "verbose", "v", false, "show detailed timing")
	flagSet.BoolVar(&flags.noSignature, "no-signature", false, "disable signature injection")
	flagSet.BoolVar(&flags.noStyle, "no-style", false, "disable CSS styling")
	flagSet.BoolVar(&flags.noFooter, "no-footer", false, "disable page footer")
	flagSet.BoolVar(&flags.noWatermark, "no-watermark", false, "disable watermark")
	flagSet.BoolVar(&flags.noCover, "no-cover", false, "disable cover page")
	flagSet.StringVar(&flags.coverTitle, "cover-title", "", "override cover page title")
	flagSet.BoolVar(&flags.version, "version", false, "show version and exit")
	flagSet.IntVarP(&flags.workers, "workers", "w", 0, "number of parallel workers (default: auto)")
	flagSet.StringVarP(&flags.pageSize, "page-size", "p", "", "page size: letter, a4, legal")
	flagSet.StringVar(&flags.orientation, "orientation", "", "page orientation: portrait, landscape")
	flagSet.Float64Var(&flags.margin, "margin", 0, "page margin in inches (0.25-3.0)")
	flagSet.StringVar(&flags.watermarkText, "watermark-text", "", "watermark text (e.g., DRAFT, CONFIDENTIAL)")
	flagSet.StringVar(&flags.watermarkColor, "watermark-color", "", "watermark color in hex (default: #888888)")
	flagSet.Float64Var(&flags.watermarkOpacity, "watermark-opacity", 0, "watermark opacity 0.0-1.0 (default: 0.1)")
	flagSet.Float64Var(&flags.watermarkAngle, "watermark-angle", watermarkAngleSentinel, "watermark rotation in degrees (default: -45)")

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: md2pdf [flags] <input> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Converts Markdown files to PDF.\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  input    Markdown file or directory (optional if config has input.defaultDir)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flagSet.PrintDefaults()
	}

	// Skip program name (args[0])
	if len(args) > 1 {
		if err := flagSet.Parse(args[1:]); err != nil {
			return nil, nil, err
		}
	}

	return flags, flagSet.Args(), nil
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

// resolveCSSContent resolves CSS content from CLI flag or config.
// Priority: 1) --no-style disables all, 2) --css flag (external file), 3) config.CSS.Style (embedded), 4) none.
func resolveCSSContent(cssFile string, cfg *config.Config, noStyle bool) (string, error) {
	if noStyle {
		return "", nil
	}

	// 1. CLI flag overrides everything (for dev/debug)
	if cssFile != "" {
		content, err := os.ReadFile(cssFile) // #nosec G304 -- user-provided path
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrReadCSS, err)
		}
		return string(content), nil
	}

	// 2. Config style reference loads from embedded assets
	if cfg != nil && cfg.CSS.Style != "" {
		return assets.LoadStyle(cfg.CSS.Style)
	}

	// 3. No CSS
	return "", nil
}

// discoverFiles finds all markdown files to convert.
func discoverFiles(inputPath, outputDir string) ([]FileToConvert, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		// Single file
		if err := validateMarkdownExtension(inputPath); err != nil {
			return nil, err
		}
		outPath := resolveOutputPath(inputPath, outputDir, "")
		return []FileToConvert{{InputPath: inputPath, OutputPath: outPath}}, nil
	}

	// Directory: walk recursively
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
// baseInputDir is used for mirroring directory structure (empty for single file).
func resolveOutputPath(inputPath, outputDir, baseInputDir string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	// No output dir specified: put PDF next to source
	if outputDir == "" {
		return filepath.Join(filepath.Dir(inputPath), base+".pdf")
	}

	// Output looks like a file (has .pdf extension).
	// Note: a directory named "foo.pdf/" would be misdetected as a file,
	// but this is an unlikely edge case in practice.
	if strings.HasSuffix(outputDir, ".pdf") {
		return outputDir
	}

	// Mirror directory structure if we have a base input dir.
	// If filepath.Rel fails (e.g., paths on different drives on Windows),
	// fall through to flat output in outputDir.
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
	if n > maxWorkers {
		return fmt.Errorf("%w: %d (maximum is %d)", ErrInvalidWorkerCount, n, maxWorkers)
	}
	return nil
}

// buildSignatureData creates md2pdf.Signature from config if signature is enabled.
// Returns nil if signature is disabled (via config or --no-signature flag).
// Returns error if image path is set but file doesn't exist.
func buildSignatureData(cfg *config.Config, noSignature bool) (*md2pdf.Signature, error) {
	if noSignature || !cfg.Signature.Enabled {
		return nil, nil
	}

	// Validate image path if set (and not a URL)
	if cfg.Signature.ImagePath != "" && !isURL(cfg.Signature.ImagePath) {
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
		Name:      cfg.Signature.Name,
		Title:     cfg.Signature.Title,
		Email:     cfg.Signature.Email,
		ImagePath: cfg.Signature.ImagePath,
		Links:     links,
	}, nil
}

// isURL returns true if the string looks like a URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// buildFooterData creates md2pdf.Footer from config if footer is enabled.
// Returns nil if footer is disabled (via config or --no-footer flag).
func buildFooterData(cfg *config.Config, noFooter bool) *md2pdf.Footer {
	if noFooter || !cfg.Footer.Enabled {
		return nil
	}

	return &md2pdf.Footer{
		Position:       cfg.Footer.Position,
		ShowPageNumber: cfg.Footer.ShowPageNumber,
		Date:           cfg.Footer.Date,
		Status:         cfg.Footer.Status,
		Text:           cfg.Footer.Text,
	}
}

// buildWatermarkData creates md2pdf.Watermark from flags and config.
// Priority: CLI flags > config > defaults.
// Returns nil if watermark is disabled (via config or --no-watermark flag).
// Validates settings early for user-friendly CLI feedback.
func buildWatermarkData(flags *cliFlags, cfg *config.Config) (*md2pdf.Watermark, error) {
	if flags.noWatermark {
		return nil, nil
	}

	// Check if any watermark settings are specified
	hasFlags := flags.watermarkText != ""
	hasConfig := cfg.Watermark.Enabled

	if !hasFlags && !hasConfig {
		return nil, nil
	}

	// Start with config values if enabled
	w := &md2pdf.Watermark{}
	if cfg.Watermark.Enabled {
		w.Text = cfg.Watermark.Text
		w.Color = cfg.Watermark.Color
		w.Opacity = cfg.Watermark.Opacity
		w.Angle = cfg.Watermark.Angle
	}

	// CLI flags override config
	if flags.watermarkText != "" {
		w.Text = flags.watermarkText
	}
	if flags.watermarkColor != "" {
		w.Color = flags.watermarkColor
	}
	// Note: 0 is not a valid opacity (invisible), so we use > 0 to detect "set"
	// Negative values are caught by validation below
	if flags.watermarkOpacity != 0 {
		w.Opacity = flags.watermarkOpacity
	}
	if flags.watermarkAngle != watermarkAngleSentinel {
		w.Angle = flags.watermarkAngle
	}

	// Apply defaults for any remaining zero values
	if w.Color == "" {
		w.Color = "#888888"
	}
	if w.Opacity == 0 {
		w.Opacity = 0.1
	}
	// Angle defaults to -45, but 0 is a valid value so we use a sentinel
	if flags.watermarkAngle == watermarkAngleSentinel && cfg.Watermark.Angle == 0 && !cfg.Watermark.Enabled {
		w.Angle = -45
	}

	// Validate at boundary
	if w.Text == "" {
		return nil, fmt.Errorf("watermark text is required when watermark is enabled")
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	if w.Opacity < 0 || w.Opacity > 1 {
		return nil, fmt.Errorf("watermark opacity must be between 0 and 1, got %.2f", w.Opacity)
	}
	if w.Angle < -90 || w.Angle > 90 {
		return nil, fmt.Errorf("watermark angle must be between -90 and 90, got %.2f", w.Angle)
	}

	return w, nil
}

// buildPageSettings creates md2pdf.PageSettings from flags and config.
// Priority: CLI flags > config > defaults (handled by library).
// Returns nil if no page settings specified (library uses defaults).
// Validates settings early for user-friendly CLI feedback.
func buildPageSettings(flags *cliFlags, cfg *config.Config) (*md2pdf.PageSettings, error) {
	// Check if any page settings are specified
	hasFlags := flags.pageSize != "" || flags.orientation != "" || flags.margin > 0
	hasConfig := cfg.Page.Size != "" || cfg.Page.Orientation != "" || cfg.Page.Margin > 0

	if !hasFlags && !hasConfig {
		return nil, nil // Use library defaults
	}

	// Start with config values
	ps := &md2pdf.PageSettings{
		Size:        cfg.Page.Size,
		Orientation: cfg.Page.Orientation,
		Margin:      cfg.Page.Margin,
	}

	// CLI flags override config
	if flags.pageSize != "" {
		ps.Size = flags.pageSize
	}
	if flags.orientation != "" {
		ps.Orientation = flags.orientation
	}
	if flags.margin > 0 {
		ps.Margin = flags.margin
	}

	// Apply defaults for any remaining zero values
	if ps.Size == "" {
		ps.Size = md2pdf.PageSizeLetter
	}
	if ps.Orientation == "" {
		ps.Orientation = md2pdf.OrientationPortrait
	}
	if ps.Margin == 0 {
		ps.Margin = md2pdf.DefaultMargin
	}

	// Validate early for CLI feedback
	if err := ps.Validate(); err != nil {
		return nil, err
	}

	return ps, nil
}

// headingPattern matches the first # heading in markdown content.
var headingPattern = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// extractFirstHeading extracts the first # heading from markdown content.
// Returns empty string if no heading found.
func extractFirstHeading(markdown string) string {
	matches := headingPattern.FindStringSubmatch(markdown)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// resolveDate resolves "auto" to current date in YYYY-MM-DD format.
func resolveDate(date string) string {
	if strings.ToLower(date) == "auto" {
		return time.Now().Format("2006-01-02")
	}
	return date
}

// buildCoverData creates md2pdf.Cover from flags, config, and markdown content.
// Priority: CLI flags > config > auto extraction (H1 → filename).
// Returns nil if cover is disabled (via config or --no-cover flag).
// Validates settings early for user-friendly CLI feedback.
func buildCoverData(flags *cliFlags, cfg *config.Config, markdownContent, filename string) (*md2pdf.Cover, error) {
	if flags.noCover {
		return nil, nil
	}

	if !cfg.Cover.Enabled {
		return nil, nil
	}

	c := &md2pdf.Cover{}

	// Title resolution: CLI → config → H1 → filename
	if flags.coverTitle != "" {
		c.Title = flags.coverTitle
	} else if cfg.Cover.Title != "" {
		c.Title = cfg.Cover.Title
	} else {
		// Extract from markdown H1
		c.Title = extractFirstHeading(markdownContent)
		if c.Title == "" {
			// Fallback to filename without extension
			c.Title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
		}
	}

	// Subtitle: config only
	c.Subtitle = cfg.Cover.Subtitle

	// Logo: config only
	c.Logo = cfg.Cover.Logo

	// Author: config → signature.name fallback
	if cfg.Cover.Author != "" {
		c.Author = cfg.Cover.Author
	} else if cfg.Signature.Enabled && cfg.Signature.Name != "" {
		c.Author = cfg.Signature.Name
	}

	// AuthorTitle: config → signature.title fallback
	if cfg.Cover.AuthorTitle != "" {
		c.AuthorTitle = cfg.Cover.AuthorTitle
	} else if cfg.Signature.Enabled && cfg.Signature.Title != "" {
		c.AuthorTitle = cfg.Signature.Title
	}

	// Organization: config only
	c.Organization = cfg.Cover.Organization

	// Date: config ("auto" → today) → footer.date fallback
	if cfg.Cover.Date != "" {
		c.Date = resolveDate(cfg.Cover.Date)
	} else if cfg.Footer.Enabled && cfg.Footer.Date != "" {
		c.Date = resolveDate(cfg.Footer.Date)
	}

	// Version: config → footer.status fallback
	if cfg.Cover.Version != "" {
		c.Version = cfg.Cover.Version
	} else if cfg.Footer.Enabled && cfg.Footer.Status != "" {
		c.Version = cfg.Footer.Status
	}

	// Validate at boundary
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// convertBatch processes files concurrently using the service pool.
// Each worker acquires its own service (browser) for true parallelism.
// The context is checked for cancellation between file conversions.
func convertBatch(ctx context.Context, pool Pool, files []FileToConvert, cssContent string, footerData *md2pdf.Footer, sigData *md2pdf.Signature, pageData *md2pdf.PageSettings, watermarkData *md2pdf.Watermark, flags *cliFlags, cfg *config.Config) []ConversionResult {
	if len(files) == 0 {
		return nil
	}

	// Concurrency limited by pool size (each worker gets its own browser)
	concurrency := pool.Size()
	if concurrency > len(files) {
		concurrency = len(files)
	}

	results := make([]ConversionResult, len(files))
	var wg sync.WaitGroup
	jobs := make(chan int, len(files))

	// Start workers
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Each worker acquires its own service
			svc := pool.Acquire()
			defer pool.Release(svc)

			for idx := range jobs {
				// Check for cancellation before processing
				if ctx.Err() != nil {
					results[idx] = ConversionResult{
						InputPath: files[idx].InputPath,
						Err:       ctx.Err(),
					}
					continue
				}
				results[idx] = convertFile(ctx, svc, files[idx], cssContent, footerData, sigData, pageData, watermarkData, flags, cfg)
			}
		}()
	}

	// Send jobs
	for i := range files {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	return results
}

// convertFile processes a single file and returns the result.
// The context is passed to the conversion service for cancellation support.
func convertFile(ctx context.Context, service Converter, f FileToConvert, cssContent string, footerData *md2pdf.Footer, sigData *md2pdf.Signature, pageData *md2pdf.PageSettings, watermarkData *md2pdf.Watermark, flags *cliFlags, cfg *config.Config) ConversionResult {
	start := time.Now()
	result := ConversionResult{
		InputPath:  f.InputPath,
		OutputPath: f.OutputPath,
	}

	// Read the markdown file
	content, err := os.ReadFile(f.InputPath) // #nosec G304 -- discovered path
	if err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrReadMarkdown, err)
		result.Duration = time.Since(start)
		return result
	}

	// Build cover data (depends on markdown content for H1 extraction)
	coverData, err := buildCoverData(flags, cfg, string(content), f.InputPath)
	if err != nil {
		result.Err = fmt.Errorf("building cover data: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	// Ensure output directory exists
	outDir := filepath.Dir(f.OutputPath)
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		result.Err = fmt.Errorf("creating output directory: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	// Convert via service (returns []byte)
	pdfBytes, err := service.Convert(ctx, md2pdf.Input{
		Markdown:  string(content),
		CSS:       cssContent,
		Footer:    footerData,
		Signature: sigData,
		Page:      pageData,
		Watermark: watermarkData,
		Cover:     coverData,
	})
	if err != nil {
		result.Err = err
		result.Duration = time.Since(start)
		return result
	}

	// Write PDF to file (0644 is appropriate for shareable documents)
	// #nosec G306 -- PDFs are meant to be readable
	if err := os.WriteFile(f.OutputPath, pdfBytes, 0o644); err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrWritePDF, err)
		result.Duration = time.Since(start)
		return result
	}

	result.Duration = time.Since(start)
	return result
}

// printResults outputs conversion results and returns the number of failures.
func printResults(results []ConversionResult, quiet, verbose bool) int {
	var succeeded, failed int

	for _, r := range results {
		if r.Err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "FAILED %s: %v\n", r.InputPath, r.Err)
			continue
		}

		succeeded++
		if quiet {
			continue
		}

		if verbose {
			fmt.Printf("%s -> %s (%v)\n", r.InputPath, r.OutputPath, r.Duration.Round(time.Millisecond))
		} else {
			fmt.Printf("Created %s\n", r.OutputPath)
		}
	}

	// Summary for batch operations
	if !quiet && len(results) > 1 {
		fmt.Printf("\n%d succeeded, %d failed\n", succeeded, failed)
	}

	return failed
}
