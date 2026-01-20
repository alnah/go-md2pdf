package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	md2pdf "github.com/alnah/go-md2pdf"
)

// Sentinel errors for file discovery.
var (
	ErrInvalidExtension   = errors.New("file must have .md or .markdown extension")
	ErrInvalidWorkerCount = errors.New("invalid worker count")
)

// FileToConvert represents a single file to process.
type FileToConvert struct {
	InputPath  string
	OutputPath string
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
			return fmt.Errorf("scanning %s: %w", path, err)
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

// htmlOutputPath returns the HTML path corresponding to a PDF path.
func htmlOutputPath(pdfPath string) string {
	return strings.TrimSuffix(pdfPath, ".pdf") + ".html"
}
