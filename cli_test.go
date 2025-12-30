package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantConfig     string
		wantOutput     string
		wantCSS        string
		wantQuiet      bool
		wantVerbose    bool
		wantPositional []string
		wantErr        bool
	}{
		{
			name:           "no args",
			args:           []string{"go-md2pdf"},
			wantPositional: []string{},
		},
		{
			name:           "single file",
			args:           []string{"go-md2pdf", "doc.md"},
			wantPositional: []string{"doc.md"},
		},
		{
			name:           "config flag",
			args:           []string{"go-md2pdf", "--config", "work"},
			wantConfig:     "work",
			wantPositional: []string{},
		},
		{
			name:           "output flag short",
			args:           []string{"go-md2pdf", "-o", "./out/"},
			wantOutput:     "./out/",
			wantPositional: []string{},
		},
		{
			name:           "css flag",
			args:           []string{"go-md2pdf", "--css", "style.css"},
			wantCSS:        "style.css",
			wantPositional: []string{},
		},
		{
			name:           "quiet flag",
			args:           []string{"go-md2pdf", "--quiet"},
			wantQuiet:      true,
			wantPositional: []string{},
		},
		{
			name:           "verbose flag",
			args:           []string{"go-md2pdf", "--verbose"},
			wantVerbose:    true,
			wantPositional: []string{},
		},
		{
			name:           "all flags with file",
			args:           []string{"go-md2pdf", "--config", "work", "-o", "out.pdf", "--css", "style.css", "--verbose", "doc.md"},
			wantConfig:     "work",
			wantOutput:     "out.pdf",
			wantCSS:        "style.css",
			wantVerbose:    true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:    "unknown flag returns error",
			args:    []string{"go-md2pdf", "--unknown"},
			wantErr: true,
		},
		{
			name:           "flags after positional argument",
			args:           []string{"go-md2pdf", "doc.md", "-o", "./out/", "--verbose"},
			wantOutput:     "./out/",
			wantVerbose:    true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:           "short flags",
			args:           []string{"go-md2pdf", "-c", "work", "-q", "-v", "doc.md"},
			wantConfig:     "work",
			wantQuiet:      true,
			wantVerbose:    true,
			wantPositional: []string{"doc.md"},
		},
		{
			name:           "mixed long and short flags",
			args:           []string{"go-md2pdf", "--config", "work", "-o", "./out/", "doc.md", "-v"},
			wantConfig:     "work",
			wantOutput:     "./out/",
			wantVerbose:    true,
			wantPositional: []string{"doc.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if flags.configName != tt.wantConfig {
				t.Errorf("configName = %q, want %q", flags.configName, tt.wantConfig)
			}
			if flags.outputPath != tt.wantOutput {
				t.Errorf("outputPath = %q, want %q", flags.outputPath, tt.wantOutput)
			}
			if flags.cssFile != tt.wantCSS {
				t.Errorf("cssFile = %q, want %q", flags.cssFile, tt.wantCSS)
			}
			if flags.quiet != tt.wantQuiet {
				t.Errorf("quiet = %v, want %v", flags.quiet, tt.wantQuiet)
			}
			if flags.verbose != tt.wantVerbose {
				t.Errorf("verbose = %v, want %v", flags.verbose, tt.wantVerbose)
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
			got := resolveOutputDir(tt.flagOutput, tt.cfg)
			if got != tt.want {
				t.Errorf("resolveOutputDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveOutputPath(t *testing.T) {
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
			got := resolveOutputPath(tt.inputPath, tt.outputDir, tt.baseInputDir)
			if got != tt.want {
				t.Errorf("resolveOutputPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateMarkdownExtension(t *testing.T) {
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
			err := validateMarkdownExtension(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMarkdownExtension() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiscoverFiles(t *testing.T) {
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
		got, err := discoverFiles(tempDir, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 4 {
			t.Errorf("got %d files, want 4 (doc1.md, doc2.markdown, subdir/doc3.md, subdir/deep/doc4.md)", len(got))
		}
	})

	t.Run("directory with output dir mirrors structure", func(t *testing.T) {
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
		inputPath := filepath.Join(tempDir, "ignored.txt")
		_, err := discoverFiles(inputPath, "")
		if err == nil {
			t.Error("expected error for invalid extension")
		}
	})

	t.Run("nonexistent path returns error", func(t *testing.T) {
		_, err := discoverFiles("/nonexistent/path", "")
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
	})
}

func TestResolveCSSContent(t *testing.T) {
	t.Run("empty file and no config returns empty string", func(t *testing.T) {
		got, err := resolveCSSContent("", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("reads CSS file content", func(t *testing.T) {
		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "style.css")
		cssContent := "body { color: red; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		got, err := resolveCSSContent(cssPath, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != cssContent {
			t.Errorf("got %q, want %q", got, cssContent)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := resolveCSSContent("/nonexistent/style.css", nil)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("config style loads from embedded assets", func(t *testing.T) {
		cfg := &Config{CSS: CSSConfig{Style: "fle"}}
		got, err := resolveCSSContent("", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "" {
			t.Error("expected CSS content from embedded assets, got empty string")
		}
	})

	t.Run("css flag overrides config style", func(t *testing.T) {
		tempDir := t.TempDir()
		cssPath := filepath.Join(tempDir, "override.css")
		cssContent := "body { color: blue; }"
		if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
			t.Fatalf("failed to write CSS file: %v", err)
		}

		cfg := &Config{CSS: CSSConfig{Style: "fle"}}
		got, err := resolveCSSContent(cssPath, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != cssContent {
			t.Errorf("got %q, want %q (flag should override config)", got, cssContent)
		}
	})

	t.Run("unknown config style returns error", func(t *testing.T) {
		cfg := &Config{CSS: CSSConfig{Style: "nonexistent"}}
		_, err := resolveCSSContent("", cfg)
		if err == nil {
			t.Error("expected error for unknown style")
		}
	})
}

func TestPrintResults(t *testing.T) {
	t.Run("returns zero for all success", func(t *testing.T) {
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
		failed := printResults(nil, true, false)
		if failed != 0 {
			t.Errorf("failed = %d, want 0", failed)
		}
	})
}

func TestBuildSignatureData(t *testing.T) {
	t.Run("noSignature flag returns nil", func(t *testing.T) {
		cfg := &Config{Signature: SignatureConfig{Enabled: true, Name: "Test"}}
		got, err := buildSignatureData(cfg, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil when noSignature=true")
		}
	})

	t.Run("signature disabled in config returns nil", func(t *testing.T) {
		cfg := &Config{Signature: SignatureConfig{Enabled: false, Name: "Test"}}
		got, err := buildSignatureData(cfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil when signature.enabled=false")
		}
	})

	t.Run("valid signature config returns SignatureData", func(t *testing.T) {
		cfg := &Config{Signature: SignatureConfig{
			Enabled: true,
			Name:    "John Doe",
			Title:   "Developer",
			Email:   "john@example.com",
			Links: []Link{
				{Label: "GitHub", URL: "https://github.com/johndoe"},
			},
		}}
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
		cfg := &Config{Signature: SignatureConfig{
			Enabled:   true,
			Name:      "Test",
			ImagePath: "https://example.com/logo.png",
		}}
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
		cfg := &Config{Signature: SignatureConfig{
			Enabled:   true,
			Name:      "Test",
			ImagePath: "/nonexistent/path/to/image.png",
		}}
		_, err := buildSignatureData(cfg, false)
		if err == nil {
			t.Fatal("expected error for nonexistent image path")
		}
		if !errors.Is(err, ErrSignatureImagePath) {
			t.Errorf("error = %v, want ErrSignatureImagePath", err)
		}
	})

	t.Run("existing local image path is accepted", func(t *testing.T) {
		tempDir := t.TempDir()
		imagePath := filepath.Join(tempDir, "logo.png")
		if err := os.WriteFile(imagePath, []byte("fake png"), 0644); err != nil {
			t.Fatalf("failed to create test image: %v", err)
		}

		cfg := &Config{Signature: SignatureConfig{
			Enabled:   true,
			Name:      "Test",
			ImagePath: imagePath,
		}}
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
		cfg := &Config{Signature: SignatureConfig{
			Enabled:   true,
			Name:      "Test",
			ImagePath: "",
		}}
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
			got := isURL(tt.input)
			if got != tt.want {
				t.Errorf("isURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildFooterData(t *testing.T) {
	t.Run("footer disabled returns nil", func(t *testing.T) {
		cfg := &Config{Footer: FooterConfig{
			Enabled:        false,
			Position:       "right",
			ShowPageNumber: true,
			Text:           "Footer Text",
		}}
		got := buildFooterData(cfg)
		if got != nil {
			t.Error("expected nil when footer.enabled=false")
		}
	})

	t.Run("footer enabled returns FooterData", func(t *testing.T) {
		cfg := &Config{Footer: FooterConfig{
			Enabled:        true,
			Position:       "center",
			ShowPageNumber: true,
			Date:           "2025-01-15",
			Status:         "DRAFT",
			Text:           "Footer Text",
		}}
		got := buildFooterData(cfg)
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
		cfg := &Config{Footer: FooterConfig{
			Enabled: true,
			// All other fields empty/false
		}}
		got := buildFooterData(cfg)
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
}
