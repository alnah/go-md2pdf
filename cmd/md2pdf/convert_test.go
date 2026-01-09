package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

// Aliases for cleaner test code
type Config = config.Config
type InputConfig = config.InputConfig
type OutputConfig = config.OutputConfig
type CSSConfig = config.CSSConfig
type SignatureConfig = config.SignatureConfig
type FooterConfig = config.FooterConfig
type AuthorConfig = config.AuthorConfig
type DocumentConfig = config.DocumentConfig
type Link = config.Link

// cliFlags is an alias for convertFlags (backward compatibility for tests)
type cliFlags = convertFlags

// parseFlags is a compatibility wrapper for tests (maps to parseConvertFlags).
func parseFlags(args []string) (*convertFlags, []string, error) {
	// Skip program name if present (legacy behavior)
	if len(args) > 0 {
		return parseConvertFlags(args[1:])
	}
	return parseConvertFlags(args)
}

// printResults is a compatibility wrapper for tests.
func printResults(results []ConversionResult, quiet, verbose bool) int {
	deps := DefaultDeps()
	return printResultsWithWriter(results, quiet, verbose, deps)
}

func TestParseFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		args            []string
		wantConfig      string
		wantOutput      string
		wantCSS         string
		wantQuiet       bool
		wantVerbose     bool
		wantNoSignature bool
		wantNoStyle     bool
		wantNoFooter    bool
		wantVersion     bool
		wantPageSize    string
		wantOrientation string
		wantMargin      float64
		wantPositional  []string
		wantErr         bool
	}{
		{
			name:           "no args",
			args:           []string{"md2pdf"},
			wantPositional: []string{},
		},
		{
			name:           "single file",
			args:           []string{"md2pdf", "doc.md"},
			wantPositional: []string{"doc.md"},
		},
		{
			name:           "config flag",
			args:           []string{"md2pdf", "--config", "work"},
			wantConfig:     "work",
			wantPositional: []string{},
		},
		{
			name:           "output flag short",
			args:           []string{"md2pdf", "-o", "./out/"},
			wantOutput:     "./out/",
			wantPositional: []string{},
		},
		{
			name:           "css flag",
			args:           []string{"md2pdf", "--css", "style.css"},
			wantCSS:        "style.css",
			wantPositional: []string{},
		},
		{
			name:           "quiet flag",
			args:           []string{"md2pdf", "--quiet"},
			wantQuiet:      true,
			wantPositional: []string{},
		},
		{
			name:           "verbose flag",
			args:           []string{"md2pdf", "--verbose"},
			wantVerbose:    true,
			wantPositional: []string{},
		},
		{
			name:           "all flags with file",
			args:           []string{"md2pdf", "--config", "work", "-o", "out.pdf", "--css", "style.css", "--verbose", "doc.md"},
			wantConfig:     "work",
			wantOutput:     "out.pdf",
			wantCSS:        "style.css",
			wantVerbose:    true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:    "unknown flag returns error",
			args:    []string{"md2pdf", "--unknown"},
			wantErr: true,
		},
		{
			name:           "flags after positional argument",
			args:           []string{"md2pdf", "doc.md", "-o", "./out/", "--verbose"},
			wantOutput:     "./out/",
			wantVerbose:    true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:           "short flags",
			args:           []string{"md2pdf", "-c", "work", "-q", "-v", "doc.md"},
			wantConfig:     "work",
			wantQuiet:      true,
			wantVerbose:    true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:           "mixed long and short flags",
			args:           []string{"md2pdf", "--config", "work", "-o", "./out/", "doc.md", "-v"},
			wantConfig:     "work",
			wantOutput:     "./out/",
			wantVerbose:    true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:            "no-signature flag",
			args:            []string{"md2pdf", "--no-signature", "doc.md"},
			wantNoSignature: true,
			wantPositional:  []string{"doc.md"},
		},
		{
			name:           "no-style flag",
			args:           []string{"md2pdf", "--no-style", "doc.md"},
			wantNoStyle:    true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:           "no-footer flag",
			args:           []string{"md2pdf", "--no-footer", "doc.md"},
			wantNoFooter:   true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:            "all disable flags combined",
			args:            []string{"md2pdf", "--no-signature", "--no-style", "--no-footer", "doc.md"},
			wantNoSignature: true,
			wantNoStyle:     true,
			wantNoFooter:    true,
			wantPositional:  []string{"doc.md"},
		},
		{
			name:           "page-size flag",
			args:           []string{"md2pdf", "--page-size", "a4", "doc.md"},
			wantPageSize:   "a4",
			wantPositional: []string{"doc.md"},
		},
		{
			name:           "page-size short flag",
			args:           []string{"md2pdf", "-p", "legal", "doc.md"},
			wantPageSize:   "legal",
			wantPositional: []string{"doc.md"},
		},
		{
			name:            "orientation flag",
			args:            []string{"md2pdf", "--orientation", "landscape", "doc.md"},
			wantOrientation: "landscape",
			wantPositional:  []string{"doc.md"},
		},
		{
			name:           "margin flag",
			args:           []string{"md2pdf", "--margin", "1.5", "doc.md"},
			wantMargin:     1.5,
			wantPositional: []string{"doc.md"},
		},
		{
			name:            "all page flags combined",
			args:            []string{"md2pdf", "-p", "a4", "--orientation", "landscape", "--margin", "1.0", "doc.md"},
			wantPageSize:    "a4",
			wantOrientation: "landscape",
			wantMargin:      1.0,
			wantPositional:  []string{"doc.md"},
		},
		{
			name:           "version flag",
			args:           []string{"md2pdf", "--version"},
			wantVersion:    true,
			wantPositional: []string{},
		},
		{
			name:           "version flag with other args ignored",
			args:           []string{"md2pdf", "--version", "doc.md"},
			wantVersion:    true,
			wantPositional: []string{"doc.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flags, positional, err := parseFlags(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if flags.common.config != tt.wantConfig {
				t.Errorf("configName = %q, want %q", flags.common.config, tt.wantConfig)
			}
			if flags.output != tt.wantOutput {
				t.Errorf("outputPath = %q, want %q", flags.output, tt.wantOutput)
			}
			if flags.style.css != tt.wantCSS {
				t.Errorf("cssFile = %q, want %q", flags.style.css, tt.wantCSS)
			}
			if flags.common.quiet != tt.wantQuiet {
				t.Errorf("quiet = %v, want %v", flags.common.quiet, tt.wantQuiet)
			}
			if flags.common.verbose != tt.wantVerbose {
				t.Errorf("verbose = %v, want %v", flags.common.verbose, tt.wantVerbose)
			}
			if flags.signature.disabled != tt.wantNoSignature {
				t.Errorf("noSignature = %v, want %v", flags.signature.disabled, tt.wantNoSignature)
			}
			if flags.style.disabled != tt.wantNoStyle {
				t.Errorf("noStyle = %v, want %v", flags.style.disabled, tt.wantNoStyle)
			}
			if flags.footer.disabled != tt.wantNoFooter {
				t.Errorf("noFooter = %v, want %v", flags.footer.disabled, tt.wantNoFooter)
			}
			if flags.version != tt.wantVersion {
				t.Errorf("version = %v, want %v", flags.version, tt.wantVersion)
			}
			if flags.page.size != tt.wantPageSize {
				t.Errorf("pageSize = %q, want %q", flags.page.size, tt.wantPageSize)
			}
			if flags.page.orientation != tt.wantOrientation {
				t.Errorf("orientation = %q, want %q", flags.page.orientation, tt.wantOrientation)
			}
			if flags.page.margin != tt.wantMargin {
				t.Errorf("margin = %v, want %v", flags.page.margin, tt.wantMargin)
			}
			if len(positional) != len(tt.wantPositional) {
				t.Errorf("positional args = %v, want %v", positional, tt.wantPositional)
			}
			for i := range positional {
				if positional[i] != tt.wantPositional[i] {
					t.Errorf("positional[%d] = %q, want %q", i, positional[i], tt.wantPositional[i])
				}
			}
		})
	}
}

func TestResolveInputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		cfg     *Config
		want    string
		wantErr error
	}{
		{
			name: "args takes precedence over config",
			args: []string{"doc.md"},
			cfg:  &Config{Input: InputConfig{DefaultDir: "./default/"}},
			want: "doc.md",
		},
		{
			name: "config fallback when no args",
			args: []string{},
			cfg:  &Config{Input: InputConfig{DefaultDir: "./default/"}},
			want: "./default/",
		},
		{
			name:    "error when no args and no config",
			args:    []string{},
			cfg:     &Config{Input: InputConfig{DefaultDir: ""}},
			wantErr: ErrNoInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveInputPath(tt.args, tt.cfg)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("resolveInputPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveOutputDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		flagOutput string
		cfg        *Config
		want       string
	}{
		{
			name:       "flag takes precedence over config",
			flagOutput: "./out/",
			cfg:        &Config{Output: OutputConfig{DefaultDir: "./default/"}},
			want:       "./out/",
		},
		{
			name:       "config fallback when no flag",
			flagOutput: "",
			cfg:        &Config{Output: OutputConfig{DefaultDir: "./default/"}},
			want:       "./default/",
		},
		{
			name:       "empty when no flag and no config",
			flagOutput: "",
			cfg:        &Config{Output: OutputConfig{DefaultDir: ""}},
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveOutputDir(tt.flagOutput, tt.cfg)
			if got != tt.want {
				t.Errorf("resolveOutputDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveOutputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputPath    string
		outputDir    string
		baseInputDir string
		want         string
	}{
		{
			name:      "no output dir - PDF next to source",
			inputPath: "/docs/file.md",
			outputDir: "",
			want:      "/docs/file.pdf",
		},
		{
			name:      "output is PDF file",
			inputPath: "/docs/file.md",
			outputDir: "/out/result.pdf",
			want:      "/out/result.pdf",
		},
		{
			name:      "output is directory - single file",
			inputPath: "/docs/file.md",
			outputDir: "/out/",
			want:      "/out/file.pdf",
		},
		{
			name:         "output is directory - mirror structure",
			inputPath:    "/docs/subdir/file.md",
			outputDir:    "/out",
			baseInputDir: "/docs",
			want:         "/out/subdir/file.pdf",
		},
		{
			name:         "mirror structure with nested dirs",
			inputPath:    "/docs/a/b/c/file.md",
			outputDir:    "/out",
			baseInputDir: "/docs",
			want:         "/out/a/b/c/file.pdf",
		},
		{
			name:      "markdown extension",
			inputPath: "/docs/file.markdown",
			outputDir: "",
			want:      "/docs/file.pdf",
		},
		{
			// When filepath.Rel fails (e.g., different drives on Windows),
			// falls back to flat output in outputDir.
			name:         "filepath.Rel fallback - unrelated paths",
			inputPath:    "relative/file.md",
			outputDir:    "/out",
			baseInputDir: "/absolute/base",
			want:         "/out/file.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveOutputPath(tt.inputPath, tt.outputDir, tt.baseInputDir)
			if got != tt.want {
				t.Errorf("resolveOutputPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateMarkdownExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid .md extension",
			path:    "doc.md",
			wantErr: false,
		},
		{
			name:    "valid .markdown extension",
			path:    "doc.markdown",
			wantErr: false,
		},
		{
			name:    "invalid .txt extension",
			path:    "doc.txt",
			wantErr: true,
		},
		{
			name:    "invalid .pdf extension",
			path:    "doc.pdf",
			wantErr: true,
		},
		{
			name:    "no extension",
			path:    "doc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateMarkdownExtension(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMarkdownExtension() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiscoverFiles(t *testing.T) {
	t.Parallel()

	// Create temp directory structure
	tempDir := t.TempDir()

	// Create files
	files := map[string]string{
		"doc1.md":              "# Doc 1",
		"doc2.markdown":        "# Doc 2",
		"subdir/doc3.md":       "# Doc 3",
		"subdir/deep/doc4.md":  "# Doc 4",
		"ignored.txt":          "ignored",
		"subdir/ignored2.html": "ignored",
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	t.Run("single file", func(t *testing.T) {
		t.Parallel()

		inputPath := filepath.Join(tempDir, "doc1.md")
		got, err := discoverFiles(inputPath, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d files, want 1", len(got))
		}
		if got[0].InputPath != inputPath {
			t.Errorf("InputPath = %q, want %q", got[0].InputPath, inputPath)
		}
	})

	t.Run("directory recursive", func(t *testing.T) {
		t.Parallel()

		got, err := discoverFiles(tempDir, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 4 {
			t.Errorf("got %d files, want 4 (doc1.md, doc2.markdown, subdir/doc3.md, subdir/deep/doc4.md)", len(got))
		}
	})

	t.Run("directory with output dir mirrors structure", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "output")
		got, err := discoverFiles(tempDir, outputDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check that subdir structure is mirrored
		foundMirrored := false
		for _, f := range got {
			if filepath.Base(f.InputPath) == "doc3.md" {
				expectedOutput := filepath.Join(outputDir, "subdir", "doc3.pdf")
				if f.OutputPath != expectedOutput {
					t.Errorf("OutputPath = %q, want %q", f.OutputPath, expectedOutput)
				}
				foundMirrored = true
			}
		}
		if !foundMirrored {
			t.Error("did not find doc3.md in results")
		}
	})

	t.Run("invalid extension returns error", func(t *testing.T) {
		t.Parallel()

		inputPath := filepath.Join(tempDir, "ignored.txt")
		_, err := discoverFiles(inputPath, "")
		if err == nil {
			t.Error("expected error for invalid extension")
		}
	})

	t.Run("nonexistent path returns error", func(t *testing.T) {
		t.Parallel()

		_, err := discoverFiles("/nonexistent/path", "")
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
	})
}

func TestResolveCSSContent(t *testing.T) {
	t.Parallel()

	t.Run("empty file and no config returns empty string", func(t *testing.T) {
		t.Parallel()
		got, err := resolveCSSContent("", nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("reads CSS file content", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "style.css")
		cssContent := "body { color: red; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		got, err := resolveCSSContent(cssPath, nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != cssContent {
			t.Errorf("got %q, want %q", got, cssContent)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		t.Parallel()

		_, err := resolveCSSContent("/nonexistent/style.css", nil, false)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("config style loads from embedded assets", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{CSS: CSSConfig{Style: "nord"}}
		got, err := resolveCSSContent("", cfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "" {
			t.Error("expected CSS content from embedded assets, got empty string")
		}
	})

	t.Run("css flag overrides config style", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "override.css")
		cssContent := "body { color: blue; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		cfg := &Config{CSS: CSSConfig{Style: "nord"}}
		got, err := resolveCSSContent(cssPath, cfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != cssContent {
			t.Errorf("got %q, want %q (flag should override config)", got, cssContent)
		}
	})

	t.Run("unknown config style returns error", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{CSS: CSSConfig{Style: "nonexistent"}}
		_, err := resolveCSSContent("", cfg, false)
		if err == nil {
			t.Error("expected error for unknown style")
		}
	})

	t.Run("noStyle flag returns empty even with config style", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{CSS: CSSConfig{Style: "nord"}}
		got, err := resolveCSSContent("", cfg, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string (noStyle should disable CSS)", got)
		}
	})

	t.Run("noStyle flag returns empty even with css file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "style.css")
		if err := os.WriteFile(cssPath, []byte("body { color: red; }"), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		got, err := resolveCSSContent(cssPath, nil, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string (noStyle should disable CSS)", got)
		}
	})
}

func TestPrintResults(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for all success", func(t *testing.T) {
		t.Parallel()
		results := []ConversionResult{
			{InputPath: "a.md", OutputPath: "a.pdf", Err: nil},
			{InputPath: "b.md", OutputPath: "b.pdf", Err: nil},
		}
		failed := printResults(results, true, false)
		if failed != 0 {
			t.Errorf("failed = %d, want 0", failed)
		}
	})

	t.Run("returns count for failures", func(t *testing.T) {
		t.Parallel()

		results := []ConversionResult{
			{InputPath: "a.md", OutputPath: "a.pdf", Err: nil},
			{InputPath: "b.md", OutputPath: "b.pdf", Err: ErrReadMarkdown},
			{InputPath: "c.md", OutputPath: "c.pdf", Err: ErrReadMarkdown},
		}
		failed := printResults(results, true, false)
		if failed != 2 {
			t.Errorf("failed = %d, want 2", failed)
		}
	})

	t.Run("returns zero for empty results", func(t *testing.T) {
		t.Parallel()

		failed := printResults(nil, true, false)
		if failed != 0 {
			t.Errorf("failed = %d, want 0", failed)
		}
	})
}

func TestBuildSignatureData(t *testing.T) {
	t.Parallel()

	t.Run("noSignature flag returns nil", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			Author:    AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{Enabled: true},
		}
		got, err := buildSignatureData(cfg, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil when noSignature=true")
		}
	})

	t.Run("signature disabled in config returns nil", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:    AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{Enabled: false},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil when signature.enabled=false")
		}
	})

	t.Run("valid signature config returns SignatureData", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{
				Name:  "John Doe",
				Title: "Developer",
				Email: "john@example.com",
			},
			Signature: SignatureConfig{
				Enabled: true,
				Links: []Link{
					{Label: "GitHub", URL: "https://github.com/johndoe"},
				},
			},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected SignatureData, got nil")
		}
		if got.Name != "John Doe" {
			t.Errorf("Name = %q, want %q", got.Name, "John Doe")
		}
		if got.Title != "Developer" {
			t.Errorf("Title = %q, want %q", got.Title, "Developer")
		}
		if got.Email != "john@example.com" {
			t.Errorf("Email = %q, want %q", got.Email, "john@example.com")
		}
		if len(got.Links) != 1 {
			t.Fatalf("Links count = %d, want 1", len(got.Links))
		}
		if got.Links[0].Label != "GitHub" || got.Links[0].URL != "https://github.com/johndoe" {
			t.Errorf("Links[0] = %+v, want {GitHub, https://github.com/johndoe}", got.Links[0])
		}
	})

	t.Run("URL image path is accepted without file validation", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{
				Enabled:   true,
				ImagePath: "https://example.com/logo.png",
			},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected SignatureData, got nil")
		}
		if got.ImagePath != "https://example.com/logo.png" {
			t.Errorf("ImagePath = %q, want URL", got.ImagePath)
		}
	})

	t.Run("nonexistent local image path returns error", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{
				Enabled:   true,
				ImagePath: "/nonexistent/path/to/image.png",
			},
		}
		_, err := buildSignatureData(cfg, false)
		if err == nil {
			t.Fatal("expected error for nonexistent image path")
		}
		if !errors.Is(err, ErrSignatureImagePath) {
			t.Errorf("error = %v, want ErrSignatureImagePath", err)
		}
	})

	t.Run("existing local image path is accepted", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		imagePath := filepath.Join(tempDir, "logo.png")
		if err := os.WriteFile(imagePath, []byte("fake png"), 0644); err != nil {
			t.Fatalf("failed to create test image: %v", err)
		}

		cfg := &Config{
			Author: AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{
				Enabled:   true,
				ImagePath: imagePath,
			},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected SignatureData, got nil")
		}
		if got.ImagePath != imagePath {
			t.Errorf("ImagePath = %q, want %q", got.ImagePath, imagePath)
		}
	})

	t.Run("empty image path is accepted", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:    AuthorConfig{Name: "Test"},
			Signature: SignatureConfig{Enabled: true},
		}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected SignatureData, got nil")
		}
	})
}

func TestIsURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://example.com/path/to/file.png", true},
		{"/local/path/to/file.png", false},
		{"relative/path.png", false},
		{"", false},
		{"ftp://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := md2pdf.IsURL(tt.input)
			if got != tt.want {
				t.Errorf("IsURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildFooterData(t *testing.T) {
	t.Parallel()

	t.Run("footer disabled returns nil", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Footer: FooterConfig{
			Enabled:        false,
			Position:       "right",
			ShowPageNumber: true,
			Text:           "Footer Text",
		}}
		got := buildFooterData(cfg, false)
		if got != nil {
			t.Error("expected nil when footer.enabled=false")
		}
	})

	t.Run("footer enabled returns FooterData", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{
				Date:    "2025-01-15",
				Version: "DRAFT",
			},
			Footer: FooterConfig{
				Enabled:        true,
				Position:       "center",
				ShowPageNumber: true,
				Text:           "Footer Text",
			},
		}
		got := buildFooterData(cfg, false)
		if got == nil {
			t.Fatal("expected FooterData, got nil")
		}
		if got.Position != "center" {
			t.Errorf("Position = %q, want %q", got.Position, "center")
		}
		if !got.ShowPageNumber {
			t.Error("ShowPageNumber = false, want true")
		}
		if got.Date != "2025-01-15" {
			t.Errorf("Date = %q, want %q", got.Date, "2025-01-15")
		}
		if got.Status != "DRAFT" {
			t.Errorf("Status = %q, want %q", got.Status, "DRAFT")
		}
		if got.Text != "Footer Text" {
			t.Errorf("Text = %q, want %q", got.Text, "Footer Text")
		}
	})

	t.Run("footer enabled with minimal config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Footer: FooterConfig{
			Enabled: true,
			// All other fields empty/false
		}}
		got := buildFooterData(cfg, false)
		if got == nil {
			t.Fatal("expected FooterData, got nil")
		}
		// All fields should be zero values
		if got.Position != "" {
			t.Errorf("Position = %q, want empty", got.Position)
		}
		if got.ShowPageNumber {
			t.Error("ShowPageNumber = true, want false")
		}
		if got.Date != "" {
			t.Errorf("Date = %q, want empty", got.Date)
		}
		if got.Status != "" {
			t.Errorf("Status = %q, want empty", got.Status)
		}
		if got.Text != "" {
			t.Errorf("Text = %q, want empty", got.Text)
		}
	})

	t.Run("noFooter flag returns nil even when enabled in config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Footer: FooterConfig{
			Enabled:        true,
			Position:       "center",
			ShowPageNumber: true,
			Text:           "Footer Text",
		}}
		got := buildFooterData(cfg, true)
		if got != nil {
			t.Error("expected nil when noFooter=true, got FooterData")
		}
	})
}

func TestConvertFile_ErrorPaths(t *testing.T) {
	t.Parallel()

	// Mock converter that returns success
	mockConv := &staticMockConverter{result: []byte("%PDF-1.4 mock")}

	t.Run("mkdir failure returns error", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create a file where directory should be (blocks mkdir)
		blockingFile := filepath.Join(tempDir, "blocked")
		if err := os.WriteFile(blockingFile, []byte("blocker"), 0644); err != nil {
			t.Fatalf("failed to create blocking file: %v", err)
		}

		// Create input file
		inputPath := filepath.Join(tempDir, "doc.md")
		if err := os.WriteFile(inputPath, []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		// Try to output to a path under the blocking file (will fail mkdir)
		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(blockingFile, "subdir", "out.pdf"),
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Error("expected error when mkdir fails")
		}
	})

	t.Run("write failure returns ErrWritePDF", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create input file
		inputPath := filepath.Join(tempDir, "doc.md")
		if err := os.WriteFile(inputPath, []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create input: %v", err)
		}

		// Create output directory as read-only
		outDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(outDir, 0750); err != nil {
			t.Fatalf("failed to create output dir: %v", err)
		}
		if err := os.Chmod(outDir, 0500); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		t.Cleanup(func() {
			os.Chmod(outDir, 0750) // Restore for cleanup
		})

		f := FileToConvert{
			InputPath:  inputPath,
			OutputPath: filepath.Join(outDir, "out.pdf"),
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Error("expected error when write fails")
		}
		if !errors.Is(result.Err, ErrWritePDF) {
			t.Errorf("expected ErrWritePDF, got: %v", result.Err)
		}
	})

	t.Run("read failure returns ErrReadMarkdown", func(t *testing.T) {
		t.Parallel()

		f := FileToConvert{
			InputPath:  "/nonexistent/doc.md",
			OutputPath: "/tmp/out.pdf",
		}

		result := convertFile(context.Background(), mockConv, f, &conversionParams{cfg: config.DefaultConfig()})

		if result.Err == nil {
			t.Error("expected error when read fails")
		}
		if !errors.Is(result.Err, ErrReadMarkdown) {
			t.Errorf("expected ErrReadMarkdown, got: %v", result.Err)
		}
	})
}

// staticMockConverter is a simple mock that returns a fixed result.
type staticMockConverter struct {
	result []byte
	err    error
}

func (m *staticMockConverter) Convert(_ context.Context, _ md2pdf.Input) ([]byte, error) {
	return m.result, m.err
}

func TestBuildPageSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		flags           *cliFlags
		cfg             *Config
		wantNil         bool
		wantSize        string
		wantOrientation string
		wantMargin      float64
		wantErr         bool
	}{
		{
			name:    "no flags no config returns nil",
			flags:   &cliFlags{},
			cfg:     &Config{},
			wantNil: true,
		},
		{
			name:            "flags only",
			flags:           &cliFlags{page: pageFlags{size: "a4", orientation: "landscape", margin: 1.0}},
			cfg:             &Config{},
			wantSize:        "a4",
			wantOrientation: "landscape",
			wantMargin:      1.0,
		},
		{
			name:  "config only",
			flags: &cliFlags{},
			cfg: &Config{Page: PageConfig{
				Size:        "legal",
				Orientation: "portrait",
				Margin:      0.75,
			}},
			wantSize:        "legal",
			wantOrientation: "portrait",
			wantMargin:      0.75,
		},
		{
			name:  "flags override config",
			flags: &cliFlags{page: pageFlags{size: "a4", orientation: "landscape", margin: 1.5}},
			cfg: &Config{Page: PageConfig{
				Size:        "legal",
				Orientation: "portrait",
				Margin:      0.5,
			}},
			wantSize:        "a4",
			wantOrientation: "landscape",
			wantMargin:      1.5,
		},
		{
			name:  "partial flags override - size only",
			flags: &cliFlags{page: pageFlags{size: "a4"}},
			cfg: &Config{Page: PageConfig{
				Size:        "letter",
				Orientation: "landscape",
				Margin:      1.0,
			}},
			wantSize:        "a4",
			wantOrientation: "landscape",
			wantMargin:      1.0,
		},
		{
			name:  "partial flags override - orientation only",
			flags: &cliFlags{page: pageFlags{orientation: "landscape"}},
			cfg: &Config{Page: PageConfig{
				Size:        "a4",
				Orientation: "portrait",
				Margin:      0.75,
			}},
			wantSize:        "a4",
			wantOrientation: "landscape",
			wantMargin:      0.75,
		},
		{
			name:  "partial flags override - margin only",
			flags: &cliFlags{page: pageFlags{margin: 2.0}},
			cfg: &Config{Page: PageConfig{
				Size:        "legal",
				Orientation: "landscape",
				Margin:      0.5,
			}},
			wantSize:        "legal",
			wantOrientation: "landscape",
			wantMargin:      2.0,
		},
		{
			name:            "defaults applied when config partial",
			flags:           &cliFlags{},
			cfg:             &Config{Page: PageConfig{Size: "a4"}},
			wantSize:        "a4",
			wantOrientation: md2pdf.OrientationPortrait,
			wantMargin:      md2pdf.DefaultMargin,
		},
		{
			name:            "flags trigger validation with defaults",
			flags:           &cliFlags{page: pageFlags{size: "letter"}},
			cfg:             &Config{},
			wantSize:        "letter",
			wantOrientation: md2pdf.OrientationPortrait,
			wantMargin:      md2pdf.DefaultMargin,
		},
		{
			name:    "invalid size returns error",
			flags:   &cliFlags{page: pageFlags{size: "tabloid"}},
			cfg:     &Config{},
			wantErr: true,
		},
		{
			name:    "invalid orientation returns error",
			flags:   &cliFlags{page: pageFlags{orientation: "diagonal"}},
			cfg:     &Config{},
			wantErr: true,
		},
		{
			name:    "invalid margin returns error",
			flags:   &cliFlags{page: pageFlags{margin: 10.0}},
			cfg:     &Config{},
			wantErr: true,
		},
		{
			name:    "margin below minimum returns error",
			flags:   &cliFlags{page: pageFlags{margin: 0.1}},
			cfg:     &Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildPageSettings(tt.flags, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected PageSettings, got nil")
			}
			if got.Size != tt.wantSize {
				t.Errorf("Size = %q, want %q", got.Size, tt.wantSize)
			}
			if got.Orientation != tt.wantOrientation {
				t.Errorf("Orientation = %q, want %q", got.Orientation, tt.wantOrientation)
			}
			if got.Margin != tt.wantMargin {
				t.Errorf("Margin = %v, want %v", got.Margin, tt.wantMargin)
			}
		})
	}
}

// PageConfig alias for test file
type PageConfig = config.PageConfig

func TestValidateWorkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		n       int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "negative returns error",
			n:       -1,
			wantErr: true,
			errMsg:  "must be >= 0",
		},
		{
			name:    "zero is valid (auto mode)",
			n:       0,
			wantErr: false,
		},
		{
			name:    "one is valid",
			n:       1,
			wantErr: false,
		},
		{
			name:    "max workers is valid",
			n:       md2pdf.MaxPoolSize,
			wantErr: false,
		},
		{
			name:    "above max returns error",
			n:       md2pdf.MaxPoolSize + 1,
			wantErr: true,
			errMsg:  "maximum is 8",
		},
		{
			name:    "large number returns error",
			n:       100,
			wantErr: true,
			errMsg:  "maximum is 8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateWorkers(tt.n)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidWorkerCount) {
					t.Errorf("error = %v, want ErrInvalidWorkerCount", err)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message %q should contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// WatermarkConfig alias for test file
type WatermarkConfig = config.WatermarkConfig

// CoverConfig alias for test file
type CoverConfig = config.CoverConfig

func TestBuildWatermarkData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		flags       *cliFlags
		cfg         *Config
		wantNil     bool
		wantText    string
		wantColor   string
		wantOpacity float64
		wantAngle   float64
		wantErr     bool
		errContains string
	}{
		{
			name:    "noWatermark flag returns nil",
			flags:   &cliFlags{watermark: watermarkFlags{disabled: true, angle: watermarkAngleSentinel}},
			cfg:     &Config{Watermark: WatermarkConfig{Enabled: true, Text: "DRAFT"}},
			wantNil: true,
		},
		{
			name:    "neither flags nor config returns nil",
			flags:   &cliFlags{watermark: watermarkFlags{angle: watermarkAngleSentinel}},
			cfg:     &Config{},
			wantNil: true,
		},
		{
			name:  "config only returns watermark",
			flags: &cliFlags{watermark: watermarkFlags{angle: watermarkAngleSentinel}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "CONFIDENTIAL",
				Color:   "#ff0000",
				Opacity: 0.2,
				Angle:   -30,
			}},
			wantText:    "CONFIDENTIAL",
			wantColor:   "#ff0000",
			wantOpacity: 0.2,
			wantAngle:   -30,
		},
		{
			name:        "flags only returns watermark with defaults",
			flags:       &cliFlags{watermark: watermarkFlags{text: "DRAFT", angle: watermarkAngleSentinel}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888", // default
			wantOpacity: 0.1,       // default
			wantAngle:   -45,       // default
		},
		{
			name: "flags override config",
			flags: &cliFlags{watermark: watermarkFlags{
				text:    "OVERRIDE",
				color:   "#00ff00",
				opacity: 0.5,
				angle:   15,
			}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "ORIGINAL",
				Color:   "#ff0000",
				Opacity: 0.2,
				Angle:   -30,
			}},
			wantText:    "OVERRIDE",
			wantColor:   "#00ff00",
			wantOpacity: 0.5,
			wantAngle:   15,
		},
		{
			name: "partial flags override - text only",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "NEW TEXT",
				angle: watermarkAngleSentinel,
			}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "ORIGINAL",
				Color:   "#ff0000",
				Opacity: 0.3,
				Angle:   -20,
			}},
			wantText:    "NEW TEXT",
			wantColor:   "#ff0000",
			wantOpacity: 0.3,
			wantAngle:   -20,
		},
		{
			name: "angle zero is valid (not sentinel)",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: 0,
			}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1,
			wantAngle:   0, // explicit zero, not default
		},
		{
			name:  "config angle zero preserved",
			flags: &cliFlags{watermark: watermarkFlags{angle: watermarkAngleSentinel}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "DRAFT",
				Color:   "#888888",
				Opacity: 0.1,
				Angle:   0, // explicit zero in config
			}},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1,
			wantAngle:   0,
		},
		{
			name:        "empty text when enabled returns error",
			flags:       &cliFlags{watermark: watermarkFlags{color: "#888888", angle: watermarkAngleSentinel}},
			cfg:         &Config{Watermark: WatermarkConfig{Enabled: true, Text: ""}},
			wantErr:     true,
			errContains: "watermark text is required",
		},
		{
			name: "invalid opacity above 1 returns error",
			flags: &cliFlags{watermark: watermarkFlags{
				text:    "DRAFT",
				opacity: 1.5,
				angle:   -999,
			}},
			cfg:         &Config{},
			wantErr:     true,
			errContains: "watermark opacity must be between",
		},
		{
			name: "invalid opacity below 0 returns error",
			flags: &cliFlags{watermark: watermarkFlags{
				text:    "DRAFT",
				opacity: -0.1,
				angle:   -999,
			}},
			cfg:         &Config{},
			wantErr:     true,
			errContains: "watermark opacity must be between",
		},
		{
			name: "invalid angle above 90 returns error",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: 100,
			}},
			cfg:         &Config{},
			wantErr:     true,
			errContains: "angle must be between -90 and 90",
		},
		{
			name: "invalid angle below -90 returns error",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: -100,
			}},
			cfg:         &Config{},
			wantErr:     true,
			errContains: "angle must be between -90 and 90",
		},
		{
			name: "invalid color format returns error",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				color: "red", // invalid - must be hex
				angle: watermarkAngleSentinel,
			}},
			cfg:         &Config{},
			wantErr:     true,
			errContains: "invalid watermark color",
		},
		{
			name: "invalid color from config returns error",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: watermarkAngleSentinel,
			}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "DRAFT",
				Color:   "invalid",
			}},
			wantErr:     true,
			errContains: "invalid watermark color",
		},
		{
			name: "boundary angle -90 is valid",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: -90,
			}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1,
			wantAngle:   -90,
		},
		{
			name: "boundary angle 90 is valid",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: 90,
			}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1,
			wantAngle:   90,
		},
		{
			name: "boundary opacity 0 from config gets default",
			flags: &cliFlags{watermark: watermarkFlags{
				text:  "DRAFT",
				angle: watermarkAngleSentinel,
			}},
			cfg: &Config{Watermark: WatermarkConfig{
				Enabled: true,
				Text:    "DRAFT",
				Opacity: 0, // zero opacity in config - will get default
			}},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 0.1, // default applied because 0 is treated as "not set"
			wantAngle:   0,   // config angle (0) is preserved when config is enabled
		},
		{
			name: "boundary opacity 1 is valid",
			flags: &cliFlags{watermark: watermarkFlags{
				text:    "DRAFT",
				opacity: 1.0,
				angle:   -999,
			}},
			cfg:         &Config{},
			wantText:    "DRAFT",
			wantColor:   "#888888",
			wantOpacity: 1.0,
			wantAngle:   -45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildWatermarkData(tt.flags, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected Watermark, got nil")
			}
			if got.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", got.Text, tt.wantText)
			}
			if got.Color != tt.wantColor {
				t.Errorf("Color = %q, want %q", got.Color, tt.wantColor)
			}
			if got.Opacity != tt.wantOpacity {
				t.Errorf("Opacity = %v, want %v", got.Opacity, tt.wantOpacity)
			}
			if got.Angle != tt.wantAngle {
				t.Errorf("Angle = %v, want %v", got.Angle, tt.wantAngle)
			}
		})
	}
}

func TestExtractFirstHeading(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "simple H1",
			markdown: "# Hello World\n\nSome content",
			want:     "Hello World",
		},
		{
			name:     "H1 with leading/trailing spaces trimmed",
			markdown: "#   Spaced Title   \n\nContent",
			want:     "Spaced Title",
		},
		{
			name:     "H2 ignored - only H1 extracted",
			markdown: "## This is H2\n\n# This is H1",
			want:     "This is H1",
		},
		{
			name:     "no heading returns empty",
			markdown: "Just some paragraph text.\n\nNo headings here.",
			want:     "",
		},
		{
			name:     "multiple H1 returns first",
			markdown: "# First Heading\n\n# Second Heading\n\n# Third",
			want:     "First Heading",
		},
		{
			name:     "H1 with inline formatting",
			markdown: "# Title with **bold** and *italic*\n\nContent",
			want:     "Title with **bold** and *italic*",
		},
		{
			name:     "empty markdown returns empty",
			markdown: "",
			want:     "",
		},
		{
			name:     "H1 at end of file",
			markdown: "Some intro\n\n# Final Heading",
			want:     "Final Heading",
		},
		{
			name:     "hash in middle of line not H1",
			markdown: "This has a # in the middle\n\n# Real H1",
			want:     "Real H1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractFirstHeading(tt.markdown)
			if got != tt.want {
				t.Errorf("extractFirstHeading() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveDateWithTime(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	mockNow := func() time.Time { return fixedTime }

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "auto returns fixed date",
			input: "auto",
			want:  "2025-06-15",
		},
		{
			name:  "AUTO case insensitive",
			input: "AUTO",
			want:  "2025-06-15",
		},
		{
			name:  "Auto mixed case",
			input: "Auto",
			want:  "2025-06-15",
		},
		{
			name:  "explicit date preserved",
			input: "2025-01-01",
			want:  "2025-01-01",
		},
		{
			name:  "empty string preserved",
			input: "",
			want:  "",
		},
		{
			name:  "custom format preserved",
			input: "January 2025",
			want:  "January 2025",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveDateWithTime(tt.input, mockNow)
			if got != tt.want {
				t.Errorf("resolveDateWithTime(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildCoverData(t *testing.T) {
	t.Parallel()

	// Create a temp file for logo path tests
	tempDir := t.TempDir()
	existingLogo := filepath.Join(tempDir, "logo.png")
	if err := os.WriteFile(existingLogo, []byte("fake png"), 0644); err != nil {
		t.Fatalf("failed to create test logo: %v", err)
	}

	t.Run("cover disabled in config returns nil", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Cover: CoverConfig{Enabled: false}}
		got, err := buildCoverData(cfg, "# Markdown", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil when cover.enabled=false")
		}
	})

	t.Run("title from document config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Config Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "# Markdown H1", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Title != "Config Title" {
			t.Errorf("Title = %q, want %q", got.Title, "Config Title")
		}
	})

	t.Run("title extracted from H1 when no document title", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Cover: CoverConfig{Enabled: true}}
		got, err := buildCoverData(cfg, "# My Document Title\n\nContent here", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Title != "My Document Title" {
			t.Errorf("Title = %q, want %q", got.Title, "My Document Title")
		}
	})

	t.Run("title fallback to filename when no H1", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Cover: CoverConfig{Enabled: true}}
		got, err := buildCoverData(cfg, "No headings here, just content.", "my-document.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Title != "my-document" {
			t.Errorf("Title = %q, want %q", got.Title, "my-document")
		}
	})

	t.Run("subtitle from document config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title", Subtitle: "A Comprehensive Guide"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Subtitle != "A Comprehensive Guide" {
			t.Errorf("Subtitle = %q, want %q", got.Subtitle, "A Comprehensive Guide")
		}
	})

	t.Run("logo from cover config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true, Logo: existingLogo},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Logo != existingLogo {
			t.Errorf("Logo = %q, want %q", got.Logo, existingLogo)
		}
	})

	t.Run("logo URL accepted without validation", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true, Logo: "https://example.com/logo.png"},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Logo != "https://example.com/logo.png" {
			t.Errorf("Logo = %q, want URL", got.Logo)
		}
	})

	t.Run("author from author config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:   AuthorConfig{Name: "John Doe"},
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Author != "John Doe" {
			t.Errorf("Author = %q, want %q", got.Author, "John Doe")
		}
	})

	t.Run("authorTitle from author config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:   AuthorConfig{Name: "John", Title: "Senior Developer"},
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.AuthorTitle != "Senior Developer" {
			t.Errorf("AuthorTitle = %q, want %q", got.AuthorTitle, "Senior Developer")
		}
	})

	t.Run("organization from author config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author:   AuthorConfig{Organization: "Acme Corp"},
			Document: DocumentConfig{Title: "Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Organization != "Acme Corp" {
			t.Errorf("Organization = %q, want %q", got.Organization, "Acme Corp")
		}
	})

	t.Run("date from document config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title", Date: "2025-01-15"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Date != "2025-01-15" {
			t.Errorf("Date = %q, want %q", got.Date, "2025-01-15")
		}
	})

	t.Run("version from document config", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Title", Version: "v2.0.0"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Version != "v2.0.0" {
			t.Errorf("Version = %q, want %q", got.Version, "v2.0.0")
		}
	})

	t.Run("all fields populated correctly", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Author: AuthorConfig{
				Name:         "Author Name",
				Title:        "Author Role",
				Organization: "Org Name",
			},
			Document: DocumentConfig{
				Title:    "Doc Title",
				Subtitle: "A Subtitle",
				Date:     "2025-03-15",
				Version:  "v1.0.0",
			},
			Cover: CoverConfig{
				Enabled: true,
				Logo:    existingLogo,
			},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Title != "Doc Title" {
			t.Errorf("Title = %q, want %q", got.Title, "Doc Title")
		}
		if got.Subtitle != "A Subtitle" {
			t.Errorf("Subtitle = %q, want %q", got.Subtitle, "A Subtitle")
		}
		if got.Logo != existingLogo {
			t.Errorf("Logo = %q, want %q", got.Logo, existingLogo)
		}
		if got.Author != "Author Name" {
			t.Errorf("Author = %q, want %q", got.Author, "Author Name")
		}
		if got.AuthorTitle != "Author Role" {
			t.Errorf("AuthorTitle = %q, want %q", got.AuthorTitle, "Author Role")
		}
		if got.Organization != "Org Name" {
			t.Errorf("Organization = %q, want %q", got.Organization, "Org Name")
		}
		if got.Date != "2025-03-15" {
			t.Errorf("Date = %q, want %q", got.Date, "2025-03-15")
		}
		if got.Version != "v1.0.0" {
			t.Errorf("Version = %q, want %q", got.Version, "v1.0.0")
		}
	})

	t.Run("empty optional fields preserved", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Document: DocumentConfig{Title: "Just Title"},
			Cover:    CoverConfig{Enabled: true},
		}
		got, err := buildCoverData(cfg, "", "doc.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected Cover, got nil")
		}
		if got.Subtitle != "" {
			t.Errorf("Subtitle = %q, want empty", got.Subtitle)
		}
		if got.Logo != "" {
			t.Errorf("Logo = %q, want empty", got.Logo)
		}
		if got.Author != "" {
			t.Errorf("Author = %q, want empty", got.Author)
		}
		if got.Organization != "" {
			t.Errorf("Organization = %q, want empty", got.Organization)
		}
	})
}

// TOCConfig alias for test file
type TOCConfig = config.TOCConfig

func TestBuildTOCData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cfg          *Config
		flags        tocFlags
		wantNil      bool
		wantTitle    string
		wantMaxDepth int
	}{
		{
			name:    "noTOC flag returns nil",
			cfg:     &Config{TOC: TOCConfig{Enabled: true, Title: "Contents", MaxDepth: 3}},
			flags:   tocFlags{disabled: true},
			wantNil: true,
		},
		{
			name:    "config disabled returns nil",
			cfg:     &Config{TOC: TOCConfig{Enabled: false, Title: "Contents", MaxDepth: 3}},
			flags:   tocFlags{},
			wantNil: true,
		},
		{
			name:    "neither flag nor config enabled returns nil",
			cfg:     &Config{},
			flags:   tocFlags{},
			wantNil: true,
		},
		{
			name:         "config enabled with title and depth",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, Title: "Table of Contents", MaxDepth: 4}},
			flags:        tocFlags{},
			wantTitle:    "Table of Contents",
			wantMaxDepth: 4,
		},
		{
			name:         "config enabled empty title preserved",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, Title: "", MaxDepth: 3}},
			flags:        tocFlags{},
			wantTitle:    "",
			wantMaxDepth: 3,
		},
		{
			name:         "config depth 0 gets default",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, Title: "TOC", MaxDepth: 0}},
			flags:        tocFlags{},
			wantTitle:    "TOC",
			wantMaxDepth: md2pdf.DefaultTOCMaxDepth,
		},
		{
			name:         "config depth 1 boundary",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MaxDepth: 1}},
			flags:        tocFlags{},
			wantMaxDepth: 1,
		},
		{
			name:         "config depth 6 boundary",
			cfg:          &Config{TOC: TOCConfig{Enabled: true, MaxDepth: 6}},
			flags:        tocFlags{},
			wantMaxDepth: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildTOCData(tt.cfg, tt.flags)

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected TOC, got nil")
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if got.MaxDepth != tt.wantMaxDepth {
				t.Errorf("MaxDepth = %d, want %d", got.MaxDepth, tt.wantMaxDepth)
			}
		})
	}
}

func TestParseFlags_NoTOC(t *testing.T) {
	t.Parallel()

	t.Run("--no-toc flag sets noTOC true", func(t *testing.T) {
		t.Parallel()
		flags, _, err := parseFlags([]string{"md2pdf", "--no-toc", "test.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !flags.toc.disabled {
			t.Error("expected noTOC=true when --no-toc flag provided")
		}
	})

	t.Run("no --no-toc flag leaves noTOC false", func(t *testing.T) {
		t.Parallel()

		flags, _, err := parseFlags([]string{"md2pdf", "test.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if flags.toc.disabled {
			t.Error("expected noTOC=false when --no-toc flag not provided")
		}
	})

	t.Run("--no-toc combined with other flags", func(t *testing.T) {
		t.Parallel()

		flags, _, err := parseFlags([]string{"md2pdf", "--no-toc", "--no-cover", "--quiet", "test.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !flags.toc.disabled {
			t.Error("expected noTOC=true")
		}
		if !flags.cover.disabled {
			t.Error("expected noCover=true")
		}
		if !flags.common.quiet {
			t.Error("expected quiet=true")
		}
	})
}

// PageBreaksConfig alias for test file
type PageBreaksConfig = config.PageBreaksConfig

func TestParseBreakBefore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		wantH1 bool
		wantH2 bool
		wantH3 bool
	}{
		{
			name:   "empty string returns all false",
			input:  "",
			wantH1: false,
			wantH2: false,
			wantH3: false,
		},
		{
			name:   "h1 only",
			input:  "h1",
			wantH1: true,
			wantH2: false,
			wantH3: false,
		},
		{
			name:   "h2 only",
			input:  "h2",
			wantH1: false,
			wantH2: true,
			wantH3: false,
		},
		{
			name:   "h3 only",
			input:  "h3",
			wantH1: false,
			wantH2: false,
			wantH3: true,
		},
		{
			name:   "h1,h2 comma separated",
			input:  "h1,h2",
			wantH1: true,
			wantH2: true,
			wantH3: false,
		},
		{
			name:   "h2,h3 comma separated",
			input:  "h2,h3",
			wantH1: false,
			wantH2: true,
			wantH3: true,
		},
		{
			name:   "all headings h1,h2,h3",
			input:  "h1,h2,h3",
			wantH1: true,
			wantH2: true,
			wantH3: true,
		},
		{
			name:   "case insensitive H1,H2,H3",
			input:  "H1,H2,H3",
			wantH1: true,
			wantH2: true,
			wantH3: true,
		},
		{
			name:   "mixed case with spaces",
			input:  " H1 , h2 , H3 ",
			wantH1: true,
			wantH2: true,
			wantH3: true,
		},
		{
			name:   "duplicate entries",
			input:  "h1,h1,h1",
			wantH1: true,
			wantH2: false,
			wantH3: false,
		},
		{
			name:   "unrecognized entries ignored",
			input:  "h1,h4,h5,h6,invalid",
			wantH1: true,
			wantH2: false,
			wantH3: false,
		},
		{
			name:   "only unrecognized entries",
			input:  "h4,h5,h6,invalid",
			wantH1: false,
			wantH2: false,
			wantH3: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotH1, gotH2, gotH3 := parseBreakBefore(tt.input)

			if gotH1 != tt.wantH1 {
				t.Errorf("h1 = %v, want %v", gotH1, tt.wantH1)
			}
			if gotH2 != tt.wantH2 {
				t.Errorf("h2 = %v, want %v", gotH2, tt.wantH2)
			}
			if gotH3 != tt.wantH3 {
				t.Errorf("h3 = %v, want %v", gotH3, tt.wantH3)
			}
		})
	}
}

func TestBuildPageBreaksData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flags        *cliFlags
		cfg          *Config
		wantNil      bool
		wantBeforeH1 bool
		wantBeforeH2 bool
		wantBeforeH3 bool
		wantOrphans  int
		wantWidows   int
	}{
		{
			name:    "noPageBreaks flag returns nil",
			flags:   &cliFlags{pageBreaks: pageBreakFlags{disabled: true}},
			cfg:     &Config{PageBreaks: PageBreaksConfig{Enabled: true, BeforeH1: true}},
			wantNil: true,
		},
		{
			name:        "neither flags nor config returns defaults",
			flags:       &cliFlags{},
			cfg:         &Config{},
			wantOrphans: md2pdf.DefaultOrphans,
			wantWidows:  md2pdf.DefaultWidows,
		},
		{
			name:         "config only returns config values",
			flags:        &cliFlags{},
			cfg:          &Config{PageBreaks: PageBreaksConfig{Enabled: true, BeforeH1: true, BeforeH2: true, Orphans: 3, Widows: 4}},
			wantBeforeH1: true,
			wantBeforeH2: true,
			wantBeforeH3: false,
			wantOrphans:  3,
			wantWidows:   4,
		},
		{
			name:         "breakBefore flag overrides config",
			flags:        &cliFlags{pageBreaks: pageBreakFlags{breakBefore: "h2,h3"}},
			cfg:          &Config{PageBreaks: PageBreaksConfig{Enabled: true, BeforeH1: true, BeforeH2: false}},
			wantBeforeH1: false,
			wantBeforeH2: true,
			wantBeforeH3: true,
			wantOrphans:  md2pdf.DefaultOrphans,
			wantWidows:   md2pdf.DefaultWidows,
		},
		{
			name:        "orphans flag overrides config",
			flags:       &cliFlags{pageBreaks: pageBreakFlags{orphans: 5}},
			cfg:         &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 3}},
			wantOrphans: 5,
			wantWidows:  md2pdf.DefaultWidows,
		},
		{
			name:        "widows flag overrides config",
			flags:       &cliFlags{pageBreaks: pageBreakFlags{widows: 5}},
			cfg:         &Config{PageBreaks: PageBreaksConfig{Enabled: true, Widows: 3}},
			wantOrphans: md2pdf.DefaultOrphans,
			wantWidows:  5,
		},
		{
			name:         "all flags override config",
			flags:        &cliFlags{pageBreaks: pageBreakFlags{breakBefore: "h1", orphans: 4, widows: 5}},
			cfg:          &Config{PageBreaks: PageBreaksConfig{Enabled: true, BeforeH2: true, BeforeH3: true, Orphans: 2, Widows: 2}},
			wantBeforeH1: true,
			wantBeforeH2: false,
			wantBeforeH3: false,
			wantOrphans:  4,
			wantWidows:   5,
		},
		{
			name:         "config disabled but has values - uses defaults",
			flags:        &cliFlags{},
			cfg:          &Config{PageBreaks: PageBreaksConfig{Enabled: false, BeforeH1: true, Orphans: 5}},
			wantBeforeH1: false,
			wantOrphans:  md2pdf.DefaultOrphans,
			wantWidows:   md2pdf.DefaultWidows,
		},
		{
			name:        "config orphans 0 uses default",
			flags:       &cliFlags{},
			cfg:         &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 0, Widows: 3}},
			wantOrphans: md2pdf.DefaultOrphans,
			wantWidows:  3,
		},
		{
			name:        "config widows 0 uses default",
			flags:       &cliFlags{},
			cfg:         &Config{PageBreaks: PageBreaksConfig{Enabled: true, Orphans: 3, Widows: 0}},
			wantOrphans: 3,
			wantWidows:  md2pdf.DefaultWidows,
		},
		{
			name:         "breakBefore flag with empty config",
			flags:        &cliFlags{pageBreaks: pageBreakFlags{breakBefore: "h1,h2,h3"}},
			cfg:          &Config{},
			wantBeforeH1: true,
			wantBeforeH2: true,
			wantBeforeH3: true,
			wantOrphans:  md2pdf.DefaultOrphans,
			wantWidows:   md2pdf.DefaultWidows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildPageBreaksData(tt.flags, tt.cfg)

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected PageBreaks, got nil")
			}
			if got.BeforeH1 != tt.wantBeforeH1 {
				t.Errorf("BeforeH1 = %v, want %v", got.BeforeH1, tt.wantBeforeH1)
			}
			if got.BeforeH2 != tt.wantBeforeH2 {
				t.Errorf("BeforeH2 = %v, want %v", got.BeforeH2, tt.wantBeforeH2)
			}
			if got.BeforeH3 != tt.wantBeforeH3 {
				t.Errorf("BeforeH3 = %v, want %v", got.BeforeH3, tt.wantBeforeH3)
			}
			if got.Orphans != tt.wantOrphans {
				t.Errorf("Orphans = %d, want %d", got.Orphans, tt.wantOrphans)
			}
			if got.Widows != tt.wantWidows {
				t.Errorf("Widows = %d, want %d", got.Widows, tt.wantWidows)
			}
		})
	}
}

func TestParseFlags_PageBreaks(t *testing.T) {
	t.Parallel()

	t.Run("--no-page-breaks flag sets noPageBreaks true", func(t *testing.T) {
		t.Parallel()
		flags, _, err := parseFlags([]string{"md2pdf", "--no-page-breaks", "test.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !flags.pageBreaks.disabled {
			t.Error("expected noPageBreaks=true when --no-page-breaks flag provided")
		}
	})

	t.Run("--break-before flag parses value", func(t *testing.T) {
		t.Parallel()

		flags, _, err := parseFlags([]string{"md2pdf", "--break-before", "h1,h2", "test.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if flags.pageBreaks.breakBefore != "h1,h2" {
			t.Errorf("breakBefore = %q, want %q", flags.pageBreaks.breakBefore, "h1,h2")
		}
	})

	t.Run("--orphans flag parses value", func(t *testing.T) {
		t.Parallel()

		flags, _, err := parseFlags([]string{"md2pdf", "--orphans", "3", "test.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if flags.pageBreaks.orphans != 3 {
			t.Errorf("orphans = %d, want 3", flags.pageBreaks.orphans)
		}
	})

	t.Run("--widows flag parses value", func(t *testing.T) {
		t.Parallel()

		flags, _, err := parseFlags([]string{"md2pdf", "--widows", "4", "test.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if flags.pageBreaks.widows != 4 {
			t.Errorf("widows = %d, want 4", flags.pageBreaks.widows)
		}
	})

	t.Run("all page break flags combined", func(t *testing.T) {
		t.Parallel()

		flags, _, err := parseFlags([]string{
			"md2pdf",
			"--break-before", "h1,h2,h3",
			"--orphans", "5",
			"--widows", "5",
			"test.md",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if flags.pageBreaks.breakBefore != "h1,h2,h3" {
			t.Errorf("breakBefore = %q, want %q", flags.pageBreaks.breakBefore, "h1,h2,h3")
		}
		if flags.pageBreaks.orphans != 5 {
			t.Errorf("orphans = %d, want 5", flags.pageBreaks.orphans)
		}
		if flags.pageBreaks.widows != 5 {
			t.Errorf("widows = %d, want 5", flags.pageBreaks.widows)
		}
	})

	t.Run("--no-page-breaks with other page break flags", func(t *testing.T) {
		t.Parallel()

		flags, _, err := parseFlags([]string{
			"md2pdf",
			"--no-page-breaks",
			"--break-before", "h1",
			"--orphans", "3",
			"test.md",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !flags.pageBreaks.disabled {
			t.Error("expected noPageBreaks=true")
		}
		// Other flags are still parsed, but noPageBreaks takes precedence
		if flags.pageBreaks.breakBefore != "h1" {
			t.Errorf("breakBefore = %q, want %q", flags.pageBreaks.breakBefore, "h1")
		}
		if flags.pageBreaks.orphans != 3 {
			t.Errorf("orphans = %d, want 3", flags.pageBreaks.orphans)
		}
	})
}

func TestMergeFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		flags *convertFlags
		cfg   *Config
		check func(t *testing.T, cfg *Config)
	}{
		{
			name:  "empty flags preserve config author",
			flags: &convertFlags{},
			cfg:   &Config{Author: AuthorConfig{Name: "Config Author"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "Config Author" {
					t.Errorf("Author.Name = %q, want %q", cfg.Author.Name, "Config Author")
				}
			},
		},
		{
			name:  "author.name overrides config",
			flags: &convertFlags{author: authorFlags{name: "CLI Author"}},
			cfg:   &Config{Author: AuthorConfig{Name: "Config Author"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Author" {
					t.Errorf("Author.Name = %q, want %q", cfg.Author.Name, "CLI Author")
				}
			},
		},
		{
			name:  "author.title overrides config",
			flags: &convertFlags{author: authorFlags{title: "CLI Title"}},
			cfg:   &Config{Author: AuthorConfig{Title: "Config Title"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Title != "CLI Title" {
					t.Errorf("Author.Title = %q, want %q", cfg.Author.Title, "CLI Title")
				}
			},
		},
		{
			name:  "author.email overrides config",
			flags: &convertFlags{author: authorFlags{email: "cli@test.com"}},
			cfg:   &Config{Author: AuthorConfig{Email: "config@test.com"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Email != "cli@test.com" {
					t.Errorf("Author.Email = %q, want %q", cfg.Author.Email, "cli@test.com")
				}
			},
		},
		{
			name:  "author.org overrides config",
			flags: &convertFlags{author: authorFlags{org: "CLI Org"}},
			cfg:   &Config{Author: AuthorConfig{Organization: "Config Org"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Organization != "CLI Org" {
					t.Errorf("Author.Organization = %q, want %q", cfg.Author.Organization, "CLI Org")
				}
			},
		},
		{
			name:  "document.title overrides config",
			flags: &convertFlags{document: documentFlags{title: "CLI Title"}},
			cfg:   &Config{Document: DocumentConfig{Title: "Config Title"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Title != "CLI Title" {
					t.Errorf("Document.Title = %q, want %q", cfg.Document.Title, "CLI Title")
				}
			},
		},
		{
			name:  "document.subtitle overrides config",
			flags: &convertFlags{document: documentFlags{subtitle: "CLI Subtitle"}},
			cfg:   &Config{Document: DocumentConfig{Subtitle: "Config Subtitle"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Subtitle != "CLI Subtitle" {
					t.Errorf("Document.Subtitle = %q, want %q", cfg.Document.Subtitle, "CLI Subtitle")
				}
			},
		},
		{
			name:  "document.version overrides config",
			flags: &convertFlags{document: documentFlags{version: "v2.0"}},
			cfg:   &Config{Document: DocumentConfig{Version: "v1.0"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Version != "v2.0" {
					t.Errorf("Document.Version = %q, want %q", cfg.Document.Version, "v2.0")
				}
			},
		},
		{
			name:  "document.date overrides config",
			flags: &convertFlags{document: documentFlags{date: "2025-06-01"}},
			cfg:   &Config{Document: DocumentConfig{Date: "auto"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Date != "2025-06-01" {
					t.Errorf("Document.Date = %q, want %q", cfg.Document.Date, "2025-06-01")
				}
			},
		},
		{
			name:  "footer.position overrides config",
			flags: &convertFlags{footer: footerFlags{position: "left"}},
			cfg:   &Config{Footer: FooterConfig{Position: "right"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Position != "left" {
					t.Errorf("Footer.Position = %q, want %q", cfg.Footer.Position, "left")
				}
			},
		},
		{
			name:  "footer.text overrides config",
			flags: &convertFlags{footer: footerFlags{text: "CLI Footer"}},
			cfg:   &Config{Footer: FooterConfig{Text: "Config Footer"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Text != "CLI Footer" {
					t.Errorf("Footer.Text = %q, want %q", cfg.Footer.Text, "CLI Footer")
				}
			},
		},
		{
			name:  "footer.pageNumber enables footer",
			flags: &convertFlags{footer: footerFlags{pageNumber: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false, ShowPageNumber: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.ShowPageNumber {
					t.Error("Footer.ShowPageNumber should be true")
				}
				if !cfg.Footer.Enabled {
					t.Error("Footer.Enabled should be true when pageNumber is set")
				}
			},
		},
		{
			name:  "footer.disabled disables footer",
			flags: &convertFlags{footer: footerFlags{disabled: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Enabled {
					t.Error("Footer.Enabled should be false when disabled flag is set")
				}
			},
		},
		{
			name:  "cover.logo overrides config",
			flags: &convertFlags{cover: coverFlags{logo: "/cli/logo.png"}},
			cfg:   &Config{Cover: CoverConfig{Logo: "/config/logo.png"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Cover.Logo != "/cli/logo.png" {
					t.Errorf("Cover.Logo = %q, want %q", cfg.Cover.Logo, "/cli/logo.png")
				}
			},
		},
		{
			name:  "cover.disabled disables cover",
			flags: &convertFlags{cover: coverFlags{disabled: true}},
			cfg:   &Config{Cover: CoverConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Cover.Enabled {
					t.Error("Cover.Enabled should be false when disabled flag is set")
				}
			},
		},
		{
			name:  "signature.image overrides config",
			flags: &convertFlags{signature: signatureFlags{image: "/cli/sig.png"}},
			cfg:   &Config{Signature: SignatureConfig{ImagePath: "/config/sig.png"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Signature.ImagePath != "/cli/sig.png" {
					t.Errorf("Signature.ImagePath = %q, want %q", cfg.Signature.ImagePath, "/cli/sig.png")
				}
			},
		},
		{
			name:  "signature.disabled disables signature",
			flags: &convertFlags{signature: signatureFlags{disabled: true}},
			cfg:   &Config{Signature: SignatureConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Signature.Enabled {
					t.Error("Signature.Enabled should be false when disabled flag is set")
				}
			},
		},
		{
			name:  "toc.title overrides config",
			flags: &convertFlags{toc: tocFlags{title: "CLI Contents"}},
			cfg:   &Config{TOC: TOCConfig{Title: "Config Contents"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Title != "CLI Contents" {
					t.Errorf("TOC.Title = %q, want %q", cfg.TOC.Title, "CLI Contents")
				}
			},
		},
		{
			name:  "toc.depth overrides config",
			flags: &convertFlags{toc: tocFlags{depth: 4}},
			cfg:   &Config{TOC: TOCConfig{MaxDepth: 2}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.MaxDepth != 4 {
					t.Errorf("TOC.MaxDepth = %d, want %d", cfg.TOC.MaxDepth, 4)
				}
			},
		},
		{
			name:  "toc.disabled disables toc",
			flags: &convertFlags{toc: tocFlags{disabled: true}},
			cfg:   &Config{TOC: TOCConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Enabled {
					t.Error("TOC.Enabled should be false when disabled flag is set")
				}
			},
		},
		{
			name: "multiple author flags combined",
			flags: &convertFlags{author: authorFlags{
				name:  "CLI Name",
				title: "CLI Title",
				email: "cli@test.com",
				org:   "CLI Org",
			}},
			cfg: &Config{Author: AuthorConfig{
				Name:         "Config Name",
				Title:        "Config Title",
				Email:        "config@test.com",
				Organization: "Config Org",
			}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Name" {
					t.Errorf("Author.Name = %q, want %q", cfg.Author.Name, "CLI Name")
				}
				if cfg.Author.Title != "CLI Title" {
					t.Errorf("Author.Title = %q, want %q", cfg.Author.Title, "CLI Title")
				}
				if cfg.Author.Email != "cli@test.com" {
					t.Errorf("Author.Email = %q, want %q", cfg.Author.Email, "cli@test.com")
				}
				if cfg.Author.Organization != "CLI Org" {
					t.Errorf("Author.Organization = %q, want %q", cfg.Author.Organization, "CLI Org")
				}
			},
		},
		{
			name:  "partial override preserves other fields",
			flags: &convertFlags{author: authorFlags{name: "CLI Name"}},
			cfg: &Config{Author: AuthorConfig{
				Name:         "Config Name",
				Title:        "Config Title",
				Email:        "config@test.com",
				Organization: "Config Org",
			}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Name" {
					t.Errorf("Author.Name = %q, want %q", cfg.Author.Name, "CLI Name")
				}
				if cfg.Author.Title != "Config Title" {
					t.Errorf("Author.Title = %q, want %q (should be preserved)", cfg.Author.Title, "Config Title")
				}
				if cfg.Author.Email != "config@test.com" {
					t.Errorf("Author.Email = %q, want %q (should be preserved)", cfg.Author.Email, "config@test.com")
				}
				if cfg.Author.Organization != "Config Org" {
					t.Errorf("Author.Organization = %q, want %q (should be preserved)", cfg.Author.Organization, "Config Org")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mergeFlags(tt.flags, tt.cfg)
			tt.check(t, tt.cfg)
		})
	}
}

func TestParseConvertFlags_NewFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, flags *convertFlags)
	}{
		{
			name: "author-name flag",
			args: []string{"--author-name", "John Doe"},
			check: func(t *testing.T, f *convertFlags) {
				if f.author.name != "John Doe" {
					t.Errorf("author.name = %q, want %q", f.author.name, "John Doe")
				}
			},
		},
		{
			name: "author-title flag",
			args: []string{"--author-title", "Senior Developer"},
			check: func(t *testing.T, f *convertFlags) {
				if f.author.title != "Senior Developer" {
					t.Errorf("author.title = %q, want %q", f.author.title, "Senior Developer")
				}
			},
		},
		{
			name: "author-email flag",
			args: []string{"--author-email", "john@example.com"},
			check: func(t *testing.T, f *convertFlags) {
				if f.author.email != "john@example.com" {
					t.Errorf("author.email = %q, want %q", f.author.email, "john@example.com")
				}
			},
		},
		{
			name: "author-org flag",
			args: []string{"--author-org", "Acme Corp"},
			check: func(t *testing.T, f *convertFlags) {
				if f.author.org != "Acme Corp" {
					t.Errorf("author.org = %q, want %q", f.author.org, "Acme Corp")
				}
			},
		},
		{
			name: "doc-title flag",
			args: []string{"--doc-title", "My Document"},
			check: func(t *testing.T, f *convertFlags) {
				if f.document.title != "My Document" {
					t.Errorf("document.title = %q, want %q", f.document.title, "My Document")
				}
			},
		},
		{
			name: "doc-subtitle flag",
			args: []string{"--doc-subtitle", "A Comprehensive Guide"},
			check: func(t *testing.T, f *convertFlags) {
				if f.document.subtitle != "A Comprehensive Guide" {
					t.Errorf("document.subtitle = %q, want %q", f.document.subtitle, "A Comprehensive Guide")
				}
			},
		},
		{
			name: "doc-version flag",
			args: []string{"--doc-version", "v1.0.0"},
			check: func(t *testing.T, f *convertFlags) {
				if f.document.version != "v1.0.0" {
					t.Errorf("document.version = %q, want %q", f.document.version, "v1.0.0")
				}
			},
		},
		{
			name: "doc-date flag",
			args: []string{"--doc-date", "auto"},
			check: func(t *testing.T, f *convertFlags) {
				if f.document.date != "auto" {
					t.Errorf("document.date = %q, want %q", f.document.date, "auto")
				}
			},
		},
		{
			name: "footer-position flag",
			args: []string{"--footer-position", "left"},
			check: func(t *testing.T, f *convertFlags) {
				if f.footer.position != "left" {
					t.Errorf("footer.position = %q, want %q", f.footer.position, "left")
				}
			},
		},
		{
			name: "footer-text flag",
			args: []string{"--footer-text", "Confidential"},
			check: func(t *testing.T, f *convertFlags) {
				if f.footer.text != "Confidential" {
					t.Errorf("footer.text = %q, want %q", f.footer.text, "Confidential")
				}
			},
		},
		{
			name: "footer-page-number flag",
			args: []string{"--footer-page-number"},
			check: func(t *testing.T, f *convertFlags) {
				if !f.footer.pageNumber {
					t.Error("footer.pageNumber should be true")
				}
			},
		},
		{
			name: "cover-logo flag",
			args: []string{"--cover-logo", "/path/to/logo.png"},
			check: func(t *testing.T, f *convertFlags) {
				if f.cover.logo != "/path/to/logo.png" {
					t.Errorf("cover.logo = %q, want %q", f.cover.logo, "/path/to/logo.png")
				}
			},
		},
		{
			name: "sig-image flag",
			args: []string{"--sig-image", "/path/to/sig.png"},
			check: func(t *testing.T, f *convertFlags) {
				if f.signature.image != "/path/to/sig.png" {
					t.Errorf("signature.image = %q, want %q", f.signature.image, "/path/to/sig.png")
				}
			},
		},
		{
			name: "toc-title flag",
			args: []string{"--toc-title", "Table of Contents"},
			check: func(t *testing.T, f *convertFlags) {
				if f.toc.title != "Table of Contents" {
					t.Errorf("toc.title = %q, want %q", f.toc.title, "Table of Contents")
				}
			},
		},
		{
			name: "toc-depth flag",
			args: []string{"--toc-depth", "4"},
			check: func(t *testing.T, f *convertFlags) {
				if f.toc.depth != 4 {
					t.Errorf("toc.depth = %d, want %d", f.toc.depth, 4)
				}
			},
		},
		{
			name: "wm-text flag",
			args: []string{"--wm-text", "DRAFT"},
			check: func(t *testing.T, f *convertFlags) {
				if f.watermark.text != "DRAFT" {
					t.Errorf("watermark.text = %q, want %q", f.watermark.text, "DRAFT")
				}
			},
		},
		{
			name: "wm-color flag",
			args: []string{"--wm-color", "#ff0000"},
			check: func(t *testing.T, f *convertFlags) {
				if f.watermark.color != "#ff0000" {
					t.Errorf("watermark.color = %q, want %q", f.watermark.color, "#ff0000")
				}
			},
		},
		{
			name: "wm-opacity flag",
			args: []string{"--wm-opacity", "0.5"},
			check: func(t *testing.T, f *convertFlags) {
				if f.watermark.opacity != 0.5 {
					t.Errorf("watermark.opacity = %v, want %v", f.watermark.opacity, 0.5)
				}
			},
		},
		{
			name: "wm-angle flag",
			args: []string{"--wm-angle", "-30"},
			check: func(t *testing.T, f *convertFlags) {
				if f.watermark.angle != -30 {
					t.Errorf("watermark.angle = %v, want %v", f.watermark.angle, -30)
				}
			},
		},
		{
			name: "all author flags combined",
			args: []string{
				"--author-name", "John",
				"--author-title", "Dev",
				"--author-email", "j@x.com",
				"--author-org", "Acme",
			},
			check: func(t *testing.T, f *convertFlags) {
				if f.author.name != "John" {
					t.Errorf("author.name = %q, want %q", f.author.name, "John")
				}
				if f.author.title != "Dev" {
					t.Errorf("author.title = %q, want %q", f.author.title, "Dev")
				}
				if f.author.email != "j@x.com" {
					t.Errorf("author.email = %q, want %q", f.author.email, "j@x.com")
				}
				if f.author.org != "Acme" {
					t.Errorf("author.org = %q, want %q", f.author.org, "Acme")
				}
			},
		},
		{
			name: "all document flags combined",
			args: []string{
				"--doc-title", "Title",
				"--doc-subtitle", "Subtitle",
				"--doc-version", "v1.0",
				"--doc-date", "2025-01-15",
			},
			check: func(t *testing.T, f *convertFlags) {
				if f.document.title != "Title" {
					t.Errorf("document.title = %q, want %q", f.document.title, "Title")
				}
				if f.document.subtitle != "Subtitle" {
					t.Errorf("document.subtitle = %q, want %q", f.document.subtitle, "Subtitle")
				}
				if f.document.version != "v1.0" {
					t.Errorf("document.version = %q, want %q", f.document.version, "v1.0")
				}
				if f.document.date != "2025-01-15" {
					t.Errorf("document.date = %q, want %q", f.document.date, "2025-01-15")
				}
			},
		},
		{
			name: "positional args after flags",
			args: []string{"--author-name", "John", "doc.md", "doc2.md"},
			check: func(t *testing.T, f *convertFlags) {
				if f.author.name != "John" {
					t.Errorf("author.name = %q, want %q", f.author.name, "John")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flags, _, err := parseConvertFlags(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, flags)
		})
	}
}

func TestParseConvertFlags_PositionalArgs(t *testing.T) {
	t.Parallel()

	flags, positional, err := parseConvertFlags([]string{"--author-name", "John", "doc.md", "doc2.md"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flags.author.name != "John" {
		t.Errorf("author.name = %q, want %q", flags.author.name, "John")
	}
	if len(positional) != 2 {
		t.Fatalf("positional count = %d, want 2", len(positional))
	}
	if positional[0] != "doc.md" {
		t.Errorf("positional[0] = %q, want %q", positional[0], "doc.md")
	}
	if positional[1] != "doc2.md" {
		t.Errorf("positional[1] = %q, want %q", positional[1], "doc2.md")
	}
}
