package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
)

// File permission constants.
const (
	dirPermissions  = 0o750 // rwxr-x---: owner full, group read+execute
	filePermissions = 0o644 // rw-r--r--: owner read+write, others read
)

// Sentinel errors for batch operations.
var (
	ErrNoInput      = errors.New("no input specified")
	ErrReadCSS      = errors.New("failed to read CSS file")
	ErrReadMarkdown = errors.New("failed to read markdown file")
	ErrWritePDF     = errors.New("failed to write PDF file")
	ErrServiceInit  = errors.New("failed to initialize conversion service")
)

// CLIConverter is the interface for the conversion service.
type CLIConverter interface {
	Convert(ctx context.Context, input md2pdf.Input) (*md2pdf.ConvertResult, error)
}

// Compile-time interface implementation check.
var _ CLIConverter = (*md2pdf.Converter)(nil)

// Pool abstracts service pool operations for testability.
type Pool interface {
	Acquire() CLIConverter
	Release(CLIConverter)
	Size() int
}

// ConversionResult holds the outcome of a single conversion.
type ConversionResult struct {
	InputPath  string
	OutputPath string
	Err        error
	Duration   time.Duration
}

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
func convertFile(ctx context.Context, service CLIConverter, f FileToConvert, params *conversionParams) ConversionResult {
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
