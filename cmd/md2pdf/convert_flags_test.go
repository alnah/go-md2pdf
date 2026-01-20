package main

// Notes:
// - parseFlags/parseConvertFlags: we test all flag combinations including
//   short/long forms, boolean flags, value flags, and positional arguments.
// - We don't test flag.Parse() internals (Go standard library responsibility).
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"testing"
)

// ---------------------------------------------------------------------------
// TestParseFlags - CLI flag parsing
// ---------------------------------------------------------------------------

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
			args:           []string{"md2pdf", "--style", "style.css"},
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
			args:           []string{"md2pdf", "--config", "work", "-o", "out.pdf", "--style", "style.css", "--verbose", "doc.md"},
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
		// Note: --version flag was removed from convert command (use 'md2pdf version' instead)
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
			if flags.assets.style != tt.wantCSS {
				t.Errorf("style = %q, want %q", flags.assets.style, tt.wantCSS)
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
			if flags.assets.noStyle != tt.wantNoStyle {
				t.Errorf("noStyle = %v, want %v", flags.assets.noStyle, tt.wantNoStyle)
			}
			if flags.footer.disabled != tt.wantNoFooter {
				t.Errorf("noFooter = %v, want %v", flags.footer.disabled, tt.wantNoFooter)
			}
			// Note: --version flag removed from convert command
			_ = tt.wantVersion // Unused, kept for test struct compatibility
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

// ---------------------------------------------------------------------------
// TestParseFlags_NoTOC - TOC disable flag
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// TestParseFlags_PageBreaks - Page break flags
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// TestParseConvertFlags_NewFlags - Extended flag set
// ---------------------------------------------------------------------------

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
		{
			name: "timeout flag long form",
			args: []string{"--timeout", "2m"},
			check: func(t *testing.T, f *convertFlags) {
				if f.timeout != "2m" {
					t.Errorf("timeout = %q, want %q", f.timeout, "2m")
				}
			},
		},
		{
			name: "timeout flag short form",
			args: []string{"-t", "30s"},
			check: func(t *testing.T, f *convertFlags) {
				if f.timeout != "30s" {
					t.Errorf("timeout = %q, want %q", f.timeout, "30s")
				}
			},
		},
		{
			name: "timeout flag combined duration",
			args: []string{"--timeout", "1m30s"},
			check: func(t *testing.T, f *convertFlags) {
				if f.timeout != "1m30s" {
					t.Errorf("timeout = %q, want %q", f.timeout, "1m30s")
				}
			},
		},
		{
			name: "timeout with other flags",
			args: []string{"--timeout", "5m", "--workers", "4", "-o", "output.pdf"},
			check: func(t *testing.T, f *convertFlags) {
				if f.timeout != "5m" {
					t.Errorf("timeout = %q, want %q", f.timeout, "5m")
				}
				if f.workers != 4 {
					t.Errorf("workers = %d, want %d", f.workers, 4)
				}
				if f.output != "output.pdf" {
					t.Errorf("output = %q, want %q", f.output, "output.pdf")
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

// ---------------------------------------------------------------------------
// TestParseConvertFlags_PositionalArgs - Positional argument handling
// ---------------------------------------------------------------------------

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
