package main

import (
	"os"

	flag "github.com/spf13/pflag"
)

// watermarkAngleSentinel detects if --wm-angle was explicitly set.
// Since 0 is a valid angle (horizontal), we use an out-of-range sentinel.
// Valid range is -90 to 90; -999 is safely outside this range.
const watermarkAngleSentinel = -999.0

// commonFlags holds flags shared across commands.
type commonFlags struct {
	config  string
	quiet   bool
	verbose bool
}

// authorFlags holds author-related flags.
type authorFlags struct {
	name       string
	title      string
	email      string
	org        string
	phone      string
	address    string
	department string
}

// documentFlags holds document metadata flags.
type documentFlags struct {
	title        string
	subtitle     string
	version      string
	date         string
	clientName   string
	projectName  string
	documentType string
	documentID   string
	description  string
}

// pageFlags holds page layout flags.
type pageFlags struct {
	size        string
	orientation string
	margin      float64
}

// footerFlags holds footer-related flags.
type footerFlags struct {
	position       string
	text           string
	pageNumber     bool
	showDocumentID bool
	disabled       bool
}

// coverFlags holds cover page flags.
type coverFlags struct {
	logo           string
	showDepartment bool
	disabled       bool
}

// signatureFlags holds signature block flags.
type signatureFlags struct {
	image    string
	disabled bool
}

// tocFlags holds table of contents flags.
type tocFlags struct {
	title    string
	minDepth int
	maxDepth int
	disabled bool
}

// watermarkFlags holds watermark-related flags.
type watermarkFlags struct {
	text     string
	color    string
	opacity  float64
	angle    float64
	disabled bool
}

// pageBreakFlags holds page break flags.
type pageBreakFlags struct {
	breakBefore string
	orphans     int
	widows      int
	disabled    bool
}

// assetFlags holds asset-related flags (CSS, templates, custom asset path).
type assetFlags struct {
	style     string // Name or path for CSS (replaces --css)
	template  string // Name or path for template (future use)
	assetPath string // Override asset directory
	noStyle   bool   // Disable CSS styling
}

// outputFlags holds output mode flags for debugging.
type outputFlags struct {
	html     bool // Output HTML alongside PDF
	htmlOnly bool // Output HTML only, skip PDF
}

// convertFlags holds all flags for the convert command.
type convertFlags struct {
	common     commonFlags
	output     string
	workers    int
	timeout    string
	author     authorFlags
	document   documentFlags
	page       pageFlags
	footer     footerFlags
	cover      coverFlags
	signature  signatureFlags
	toc        tocFlags
	watermark  watermarkFlags
	pageBreaks pageBreakFlags
	assets     assetFlags
	outputMode outputFlags
}

// addCommonFlags adds common flags to a FlagSet.
func addCommonFlags(fs *flag.FlagSet, f *commonFlags) {
	fs.StringVarP(&f.config, "config", "c", "", "config file name or path")
	fs.BoolVarP(&f.quiet, "quiet", "q", false, "only show errors")
	fs.BoolVarP(&f.verbose, "verbose", "v", false, "show detailed timing")
}

// addAuthorFlags adds author flags to a FlagSet.
func addAuthorFlags(fs *flag.FlagSet, f *authorFlags) {
	fs.StringVar(&f.name, "author-name", "", "author name")
	fs.StringVar(&f.title, "author-title", "", "author professional title")
	fs.StringVar(&f.email, "author-email", "", "author email")
	fs.StringVar(&f.org, "author-org", "", "organization name")
	fs.StringVar(&f.phone, "author-phone", "", "author phone number")
	fs.StringVar(&f.address, "author-address", "", "author postal address")
	fs.StringVar(&f.department, "author-dept", "", "author department")
}

// addDocumentFlags adds document metadata flags to a FlagSet.
func addDocumentFlags(fs *flag.FlagSet, f *documentFlags) {
	fs.StringVar(&f.title, "doc-title", "", "document title (\"\" = auto from H1)")
	fs.StringVar(&f.subtitle, "doc-subtitle", "", "document subtitle")
	fs.StringVar(&f.version, "doc-version", "", "document version")
	fs.StringVar(&f.date, "doc-date", "", "document date (\"auto\" = today)")
	fs.StringVar(&f.clientName, "doc-client", "", "client name")
	fs.StringVar(&f.projectName, "doc-project", "", "project name")
	fs.StringVar(&f.documentType, "doc-type", "", "document type")
	fs.StringVar(&f.documentID, "doc-id", "", "document ID/reference")
	fs.StringVar(&f.description, "doc-desc", "", "document description")
}

// addPageFlags adds page layout flags to a FlagSet.
func addPageFlags(fs *flag.FlagSet, f *pageFlags) {
	fs.StringVarP(&f.size, "page-size", "p", "", "page size: letter, a4, legal")
	fs.StringVar(&f.orientation, "orientation", "", "page orientation: portrait, landscape")
	fs.Float64Var(&f.margin, "margin", 0, "page margin in inches (0.25-3.0)")
}

// addFooterFlags adds footer flags to a FlagSet.
func addFooterFlags(fs *flag.FlagSet, f *footerFlags) {
	fs.StringVar(&f.position, "footer-position", "", "footer position: left, center, right")
	fs.StringVar(&f.text, "footer-text", "", "custom footer text")
	fs.BoolVar(&f.pageNumber, "footer-page-number", false, "show page numbers in footer")
	fs.BoolVar(&f.showDocumentID, "footer-doc-id", false, "show document ID in footer")
	fs.BoolVar(&f.disabled, "no-footer", false, "disable footer")
}

// addCoverFlags adds cover page flags to a FlagSet.
func addCoverFlags(fs *flag.FlagSet, f *coverFlags) {
	fs.StringVar(&f.logo, "cover-logo", "", "cover page logo path or URL")
	fs.BoolVar(&f.showDepartment, "cover-dept", false, "show author department on cover")
	fs.BoolVar(&f.disabled, "no-cover", false, "disable cover page")
}

// addSignatureFlags adds signature block flags to a FlagSet.
func addSignatureFlags(fs *flag.FlagSet, f *signatureFlags) {
	fs.StringVar(&f.image, "sig-image", "", "signature image path")
	fs.BoolVar(&f.disabled, "no-signature", false, "disable signature block")
}

// addTOCFlags adds TOC flags to a FlagSet.
func addTOCFlags(fs *flag.FlagSet, f *tocFlags) {
	fs.StringVar(&f.title, "toc-title", "", "table of contents heading")
	fs.IntVar(&f.minDepth, "toc-min-depth", 0, "min heading depth for TOC (1-6, default: 2)")
	fs.IntVar(&f.maxDepth, "toc-max-depth", 0, "max heading depth for TOC (1-6, default: 3)")
	fs.BoolVar(&f.disabled, "no-toc", false, "disable table of contents")
}

// addWatermarkFlags adds watermark flags to a FlagSet.
func addWatermarkFlags(fs *flag.FlagSet, f *watermarkFlags) {
	fs.StringVar(&f.text, "wm-text", "", "watermark text")
	fs.StringVar(&f.color, "wm-color", "", "watermark color (hex)")
	fs.Float64Var(&f.opacity, "wm-opacity", 0, "watermark opacity (0.0-1.0)")
	fs.Float64Var(&f.angle, "wm-angle", watermarkAngleSentinel, "watermark angle in degrees")
	fs.BoolVar(&f.disabled, "no-watermark", false, "disable watermark")
}

// addPageBreakFlags adds page break flags to a FlagSet.
func addPageBreakFlags(fs *flag.FlagSet, f *pageBreakFlags) {
	fs.StringVar(&f.breakBefore, "break-before", "", "page breaks before headings: h1,h2,h3")
	fs.IntVar(&f.orphans, "orphans", 0, "min lines at page bottom (1-5)")
	fs.IntVar(&f.widows, "widows", 0, "min lines at page top (1-5)")
	fs.BoolVar(&f.disabled, "no-page-breaks", false, "disable page break features")
}

// addAssetFlags adds asset-related flags to a FlagSet.
func addAssetFlags(fs *flag.FlagSet, f *assetFlags) {
	fs.StringVar(&f.style, "style", "", "CSS style name or file path")
	fs.StringVar(&f.template, "template", "", "template name or directory path")
	fs.StringVar(&f.assetPath, "asset-path", "", "custom asset directory")
	fs.BoolVar(&f.noStyle, "no-style", false, "disable CSS styling")
}

// addOutputFlags adds output mode flags to a FlagSet.
func addOutputFlags(fs *flag.FlagSet, f *outputFlags) {
	fs.BoolVar(&f.html, "html", false, "output HTML alongside PDF")
	fs.BoolVar(&f.htmlOnly, "html-only", false, "output HTML only, skip PDF")
}

// parseConvertFlags parses convert command flags and returns positional args.
func parseConvertFlags(args []string) (*convertFlags, []string, error) {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	f := &convertFlags{}

	// I/O flags
	fs.StringVarP(&f.output, "output", "o", "", "output file or directory")
	fs.IntVarP(&f.workers, "workers", "w", 0, "parallel workers (0 = auto)")
	fs.StringVarP(&f.timeout, "timeout", "t", "", "PDF generation timeout (e.g., 30s, 2m)")

	// Flag groups
	addCommonFlags(fs, &f.common)
	addAuthorFlags(fs, &f.author)
	addDocumentFlags(fs, &f.document)
	addPageFlags(fs, &f.page)
	addFooterFlags(fs, &f.footer)
	addCoverFlags(fs, &f.cover)
	addSignatureFlags(fs, &f.signature)
	addTOCFlags(fs, &f.toc)
	addWatermarkFlags(fs, &f.watermark)
	addPageBreakFlags(fs, &f.pageBreaks)
	addAssetFlags(fs, &f.assets)
	addOutputFlags(fs, &f.outputMode)

	fs.Usage = func() { printConvertUsage(os.Stderr) }

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	return f, fs.Args(), nil
}
