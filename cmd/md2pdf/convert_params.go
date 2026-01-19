package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
	"github.com/alnah/go-md2pdf/internal/fileutil"
)

// Sentinel errors for CLI param building.
var (
	ErrSignatureImagePath = errors.New("signature image not found")
)

// conversionParams groups parameters shared across batch/file conversion.
type conversionParams struct {
	css        string
	footer     *md2pdf.Footer
	signature  *md2pdf.Signature
	page       *md2pdf.PageSettings
	watermark  *md2pdf.Watermark
	toc        *md2pdf.TOC
	pageBreaks *md2pdf.PageBreaks
	cfg        *config.Config
	htmlOnly   bool // Output HTML only, skip PDF
	htmlOutput bool // Output HTML alongside PDF
}

// buildSignatureData creates md2pdf.Signature from config.
// Uses cfg.Author.* for author information.
// Department is always shown if defined (signature always displays it).
func buildSignatureData(cfg *config.Config, noSignature bool) (*md2pdf.Signature, error) {
	if noSignature || !cfg.Signature.Enabled {
		return nil, nil
	}

	// Validate image path if set (and not a URL)
	if cfg.Signature.ImagePath != "" && !fileutil.IsURL(cfg.Signature.ImagePath) {
		if !fileutil.FileExists(cfg.Signature.ImagePath) {
			return nil, fmt.Errorf("%w: %s", ErrSignatureImagePath, cfg.Signature.ImagePath)
		}
	}

	// Convert config links to md2pdf.Link
	links := make([]md2pdf.Link, len(cfg.Signature.Links))
	for i, l := range cfg.Signature.Links {
		links[i] = md2pdf.Link{Label: l.Label, URL: l.URL}
	}

	return &md2pdf.Signature{
		Name:         cfg.Author.Name,
		Title:        cfg.Author.Title,
		Email:        cfg.Author.Email,
		Organization: cfg.Author.Organization,
		ImagePath:    cfg.Signature.ImagePath,
		Links:        links,
		Phone:        cfg.Author.Phone,
		Address:      cfg.Author.Address,
		Department:   cfg.Author.Department,
	}, nil
}

// buildFooterData creates md2pdf.Footer from config.
// Uses cfg.Document.Date and cfg.Document.Version for date/status.
// DocumentID is only shown if cfg.Footer.ShowDocumentID is true.
func buildFooterData(cfg *config.Config, noFooter bool) *md2pdf.Footer {
	if noFooter || !cfg.Footer.Enabled {
		return nil
	}

	var docID string
	if cfg.Footer.ShowDocumentID {
		docID = cfg.Document.DocumentID
	}

	return &md2pdf.Footer{
		Position:       cfg.Footer.Position,
		ShowPageNumber: cfg.Footer.ShowPageNumber,
		Date:           cfg.Document.Date,
		Status:         cfg.Document.Version,
		Text:           cfg.Footer.Text,
		DocumentID:     docID,
	}
}

// buildWatermarkData creates md2pdf.Watermark from config.
// Flags are merged into config by mergeFlags before this is called.
func buildWatermarkData(cfg *config.Config) (*md2pdf.Watermark, error) {
	if !cfg.Watermark.Enabled {
		return nil, nil
	}

	w := &md2pdf.Watermark{
		Text:    cfg.Watermark.Text,
		Color:   cfg.Watermark.Color,
		Opacity: cfg.Watermark.Opacity,
		Angle:   cfg.Watermark.Angle,
	}

	// Apply defaults for color and opacity.
	// Angle default is handled in mergeFlags to distinguish "not set" from "0".
	if w.Color == "" {
		w.Color = md2pdf.DefaultWatermarkColor
	}
	if w.Opacity == 0 {
		w.Opacity = md2pdf.DefaultWatermarkOpacity
	}

	// Validate
	if w.Text == "" {
		return nil, fmt.Errorf("watermark text is required when watermark is enabled")
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}

	return w, nil
}

// buildPageSettings creates md2pdf.PageSettings from config.
// Flags are merged into config by mergeFlags before this is called.
func buildPageSettings(cfg *config.Config) (*md2pdf.PageSettings, error) {
	hasConfig := cfg.Page.Size != "" || cfg.Page.Orientation != "" || cfg.Page.Margin > 0

	if !hasConfig {
		return nil, nil
	}

	ps := &md2pdf.PageSettings{
		Size:        cfg.Page.Size,
		Orientation: cfg.Page.Orientation,
		Margin:      cfg.Page.Margin,
	}

	// Apply defaults
	if ps.Size == "" {
		ps.Size = md2pdf.PageSizeLetter
	}
	if ps.Orientation == "" {
		ps.Orientation = md2pdf.OrientationPortrait
	}
	if ps.Margin == 0 {
		ps.Margin = md2pdf.DefaultMargin
	}

	if err := ps.Validate(); err != nil {
		return nil, err
	}

	return ps, nil
}

// firstHeadingPattern matches the first # heading in markdown content.
var firstHeadingPattern = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// extractFirstHeading extracts the first # heading from markdown content.
func extractFirstHeading(markdown string) string {
	matches := firstHeadingPattern.FindStringSubmatch(markdown)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// buildCoverData creates md2pdf.Cover from config and markdown content.
// Uses cfg.Author.* and cfg.Document.* for metadata.
// Department is only shown if cfg.Cover.ShowDepartment is true.
func buildCoverData(cfg *config.Config, markdownContent, filename string) (*md2pdf.Cover, error) {
	if !cfg.Cover.Enabled {
		return nil, nil
	}

	c := &md2pdf.Cover{
		Logo: cfg.Cover.Logo,
	}

	// Title: config -> H1 -> filename
	if cfg.Document.Title != "" {
		c.Title = cfg.Document.Title
	} else {
		c.Title = extractFirstHeading(markdownContent)
		if c.Title == "" {
			c.Title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
		}
	}

	c.Subtitle = cfg.Document.Subtitle
	c.Author = cfg.Author.Name
	c.AuthorTitle = cfg.Author.Title
	c.Organization = cfg.Author.Organization
	c.Date = cfg.Document.Date // Already resolved
	c.Version = cfg.Document.Version

	// Extended metadata fields
	c.ClientName = cfg.Document.ClientName
	c.ProjectName = cfg.Document.ProjectName
	c.DocumentType = cfg.Document.DocumentType
	c.DocumentID = cfg.Document.DocumentID
	c.Description = cfg.Document.Description

	// Department only if explicitly enabled on cover
	if cfg.Cover.ShowDepartment {
		c.Department = cfg.Author.Department
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// buildTOCData creates md2pdf.TOC from config.
func buildTOCData(cfg *config.Config, tocFlags tocFlags) *md2pdf.TOC {
	if tocFlags.disabled || !cfg.TOC.Enabled {
		return nil
	}

	maxDepth := cfg.TOC.MaxDepth
	if maxDepth == 0 {
		maxDepth = md2pdf.DefaultTOCMaxDepth
	}

	return &md2pdf.TOC{
		Title:    cfg.TOC.Title,
		MinDepth: cfg.TOC.MinDepth, // 0 = library defaults to 2
		MaxDepth: maxDepth,
	}
}

// buildPageBreaksData creates md2pdf.PageBreaks from config.
// Flags are merged into config by mergeFlags before this is called.
func buildPageBreaksData(cfg *config.Config) *md2pdf.PageBreaks {
	if !cfg.PageBreaks.Enabled {
		return nil
	}

	pb := &md2pdf.PageBreaks{
		BeforeH1: cfg.PageBreaks.BeforeH1,
		BeforeH2: cfg.PageBreaks.BeforeH2,
		BeforeH3: cfg.PageBreaks.BeforeH3,
		Orphans:  md2pdf.DefaultOrphans,
		Widows:   md2pdf.DefaultWidows,
	}

	if cfg.PageBreaks.Orphans > 0 {
		pb.Orphans = cfg.PageBreaks.Orphans
	}
	if cfg.PageBreaks.Widows > 0 {
		pb.Widows = cfg.PageBreaks.Widows
	}

	return pb
}
