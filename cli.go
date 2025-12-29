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

	flag "github.com/spf13/pflag"
)

// Sentinel errors for CLI operations.
var (
	ErrNoInput          = errors.New("no input specified")
	ErrReadCSS          = errors.New("failed to read CSS file")
	ErrReadMarkdown     = errors.New("failed to read markdown file")
	ErrInvalidExtension = errors.New("file must have .md or .markdown extension")
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
	configName string
	outputPath string
	cssFile    string
	quiet      bool
	verbose    bool
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
	cssContent, err := resolveCSSContent(flags.cssFile)
	if err != nil {
		return err
	}

	// Convert files
	results := convertBatch(service, files, cssContent)

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

// resolveCSSContent reads CSS from file if specified.
func resolveCSSContent(cssFile string) (string, error) {
	if cssFile == "" {
		return "", nil
	}
	content, err := os.ReadFile(cssFile) // #nosec G304 -- user-provided path
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReadCSS, err)
	}
	return string(content), nil
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

	// Output looks like a file (has .pdf extension)
	if strings.HasSuffix(outputDir, ".pdf") {
		return outputDir
	}

	// Mirror directory structure if we have a base input dir
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

// convertBatch processes files concurrently using the service.
func convertBatch(service Converter, files []FileToConvert, cssContent string) []ConversionResult {
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
				results[idx] = result
				return
			}

			// Ensure output directory exists
			outDir := filepath.Dir(f.OutputPath)
			if err := os.MkdirAll(outDir, 0o750); err != nil {
				result.Err = fmt.Errorf("creating output directory: %w", err)
				result.Duration = time.Since(start)
				results[idx] = result
				return
			}

			// Convert via service
			err = service.Convert(ConversionOptions{
				MarkdownContent: string(content),
				OutputPath:      f.OutputPath,
				CSSContent:      cssContent,
			})

			result.Err = err
			result.Duration = time.Since(start)
			results[idx] = result
		}(i, file)
	}

	wg.Wait()
	return results
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
