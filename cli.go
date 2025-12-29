package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alnah/go-md2pdf/internal/assets"
	flag "github.com/spf13/pflag"
)

// Sentinel errors for CLI operations.
var (
	ErrNoInput            = errors.New("no input specified")
	ErrReadCSS            = errors.New("failed to read CSS file")
	ErrReadMarkdown       = errors.New("failed to read markdown file")
	ErrInvalidExtension   = errors.New("file must have .md or .markdown extension")
	ErrSignatureImagePath = errors.New("signature image not found")
)

// Converter is the interface for the conversion service.
type Converter interface {
	Convert(opts ConversionOptions) error
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
	configName  string
	outputPath  string
	cssFile     string
	quiet       bool
	verbose     bool
	noSignature bool
}

// run parses arguments, discovers files, and orchestrates batch conversion.
func run(args []string, service Converter) error {
	flags, positionalArgs, err := parseFlags(args)
	if err != nil {
		return err
	}

	// Load configuration
	cfg := DefaultConfig()
	if flags.configName != "" {
		cfg, err = LoadConfig(flags.configName)
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
	cssContent, err := resolveCSSContent(flags.cssFile, cfg)
	if err != nil {
		return err
	}

	// Build signature data
	sigData, err := buildSignatureData(cfg, flags.noSignature)
	if err != nil {
		return err
	}

	// Convert files
	results := convertBatch(service, files, cssContent, sigData)

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
	flagSet := flag.NewFlagSet("go-md2pdf", flag.ContinueOnError)

	flags := &cliFlags{}
	flagSet.StringVarP(&flags.configName, "config", "c", "", "config name or path")
	flagSet.StringVarP(&flags.outputPath, "output", "o", "", "output file or directory")
	flagSet.StringVar(&flags.cssFile, "css", "", "CSS file for styling")
	flagSet.BoolVarP(&flags.quiet, "quiet", "q", false, "only show errors")
	flagSet.BoolVarP(&flags.verbose, "verbose", "v", false, "show detailed timing")
	flagSet.BoolVar(&flags.noSignature, "no-signature", false, "disable signature injection")

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: go-md2pdf [flags] <input> [flags]\n\n")
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
func resolveInputPath(args []string, cfg *Config) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if cfg.Input.DefaultDir != "" {
		return cfg.Input.DefaultDir, nil
	}
	return "", ErrNoInput
}

// resolveOutputDir determines the output directory from flag or config.
func resolveOutputDir(flagOutput string, cfg *Config) string {
	if flagOutput != "" {
		return flagOutput
	}
	return cfg.Output.DefaultDir
}

// resolveCSSContent resolves CSS content from CLI flag or config.
// Priority: 1) --css flag (external file), 2) config.CSS.Style (embedded), 3) none.
func resolveCSSContent(cssFile string, cfg *Config) (string, error) {
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

// buildSignatureData creates SignatureData from config if signature is enabled.
// Returns nil if signature is disabled (via config or --no-signature flag).
// Returns error if image path is set but file doesn't exist.
func buildSignatureData(cfg *Config, noSignature bool) (*SignatureData, error) {
	if noSignature || !cfg.Signature.Enabled {
		return nil, nil
	}

	// Validate image path if set (and not a URL)
	if cfg.Signature.ImagePath != "" && !isURL(cfg.Signature.ImagePath) {
		if !fileExists(cfg.Signature.ImagePath) {
			return nil, fmt.Errorf("%w: %s", ErrSignatureImagePath, cfg.Signature.ImagePath)
		}
	}

	// Convert config links to SignatureLink
	links := make([]SignatureLink, len(cfg.Signature.Links))
	for i, l := range cfg.Signature.Links {
		links[i] = SignatureLink(l)
	}

	return &SignatureData{
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

// convertBatch processes files concurrently using the service.
func convertBatch(service Converter, files []FileToConvert, cssContent string, sigData *SignatureData) []ConversionResult {
	if len(files) == 0 {
		return nil
	}

	concurrency := runtime.NumCPU()
	if concurrency > len(files) {
		concurrency = len(files)
	}

	results := make([]ConversionResult, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i, file := range files {
		wg.Add(1)
		go func(idx int, f FileToConvert) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release
			results[idx] = convertFile(service, f, cssContent, sigData)
		}(i, file)
	}

	wg.Wait()
	return results
}

// convertFile processes a single file and returns the result.
func convertFile(service Converter, f FileToConvert, cssContent string, sigData *SignatureData) ConversionResult {
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

	// Ensure output directory exists
	outDir := filepath.Dir(f.OutputPath)
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		result.Err = fmt.Errorf("creating output directory: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	// Convert via service
	err = service.Convert(ConversionOptions{
		MarkdownContent: string(content),
		OutputPath:      f.OutputPath,
		CSSContent:      cssContent,
		Signature:       sigData,
	})

	result.Err = err
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
