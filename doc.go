// Package md2pdf converts Markdown documents to PDF using headless Chrome.
//
// # Quick Start
//
// Create a converter, convert markdown, and close when done:
//
//	conv, err := md2pdf.NewConverter()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer conv.Close()
//
//	result, err := conv.Convert(ctx, md2pdf.Input{
//	    Markdown: "# Hello\n\nWorld",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	os.WriteFile("output.pdf", result.PDF, 0644)
//
// The result contains both the PDF bytes (result.PDF) and the intermediate
// HTML (result.HTML) for debugging. Use Input.HTMLOnly to skip PDF generation.
//
// # Conversion Pipeline
//
// The conversion process follows these stages:
//
//  1. Markdown preprocessing (line normalization, ==highlight== syntax)
//  2. Markdown to HTML conversion via Goldmark (GFM, syntax highlighting)
//  3. HTML injection (CSS, cover page, TOC, signature block)
//  4. PDF rendering via headless Chrome (go-rod)
//
// # Configuration
//
// Use functional options to customize the converter:
//
//	conv, err := md2pdf.NewConverter(
//	    md2pdf.WithTimeout(2 * time.Minute),
//	    md2pdf.WithStyle("technical"),
//	    md2pdf.WithAssetPath("/path/to/custom/assets"),
//	)
//
// Per-conversion options are passed via Input:
//
//	result, err := conv.Convert(ctx, md2pdf.Input{
//	    Markdown:  content,
//	    SourceDir: "/path/to/markdown",  // for relative image paths
//	    CSS:       "body { font-size: 14px; }",
//	    Page:      &md2pdf.PageSettings{Size: "a4"},
//	    Footer:    &md2pdf.Footer{ShowPageNumber: true},
//	    Cover:     &md2pdf.Cover{Title: "Report"},
//	    TOC:       &md2pdf.TOC{Title: "Contents"},
//	    Watermark: &md2pdf.Watermark{Text: "DRAFT"},
//	    Signature: &md2pdf.Signature{Name: "John Doe"},
//	})
//
// # Parallel Processing
//
// For batch conversion, use ConverterPool to manage multiple browser instances:
//
//	pool := md2pdf.NewConverterPool(4)
//	defer pool.Close()
//
//	conv := pool.Acquire()
//	defer pool.Release(conv)
//	result, err := conv.Convert(ctx, input)
//
// # Custom Assets
//
// Override built-in styles and templates using AssetLoader:
//
//	loader, err := md2pdf.NewAssetLoader("/path/to/assets")
//	conv, err := md2pdf.NewConverter(md2pdf.WithAssetLoader(loader))
//
// Asset directory structure:
//
//	assets/
//	├── styles/
//	│   └── custom.css
//	└── templates/
//	    └── custom/
//	        ├── cover.html
//	        └── signature.html
//
// # Browser Requirements
//
// PDF generation requires Chrome/Chromium. The go-rod library automatically
// downloads a managed Chromium instance on first run (~/.cache/rod/browser/).
//
// For containers and CI environments, set ROD_NO_SANDBOX=1 to disable the
// Chrome sandbox. Use ROD_BROWSER_BIN to specify a custom Chrome binary.
package md2pdf
