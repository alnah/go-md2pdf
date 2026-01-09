package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/yamlutil"
)

// Sentinel errors for config operations.
var (
	ErrConfigNotFound  = errors.New("config file not found")
	ErrEmptyConfigName = errors.New("config name cannot be empty")
	ErrConfigParse     = errors.New("failed to parse config")
	ErrFieldTooLong    = errors.New("field exceeds maximum length")
)

// Field length limits for multi-tenant safety.
const (
	MaxNameLength           = 100  // Full name (generous)
	MaxTitleLength          = 100  // Professional title
	MaxEmailLength          = 254  // RFC 5321
	MaxURLLength            = 2048 // Browser limit
	MaxTextLength           = 500  // Footer/free-form text
	MaxLabelLength          = 100  // Link label
	MaxPageSizeLength       = 10   // "letter", "a4", "legal"
	MaxOrientationLength    = 10   // "portrait", "landscape"
	MaxWatermarkTextLength  = 50   // "DRAFT", "CONFIDENTIAL"
	MaxWatermarkColorLength = 20   // "#888888" or color name
	MaxDocTitleLength       = 200  // Document title
	MaxSubtitleLength       = 200  // Document subtitle
	MaxOrganizationLength   = 100  // Organization name
	MaxVersionLength        = 50   // Version string
	MaxDateLength           = 30   // "2025-12-31" or "December 31, 2025"
	MaxTOCTitleLength       = 100  // TOC title
)

// Config holds all configuration for document generation.
type Config struct {
	Author     AuthorConfig     `yaml:"author"`
	Document   DocumentConfig   `yaml:"document"`
	Input      InputConfig      `yaml:"input"`
	Output     OutputConfig     `yaml:"output"`
	CSS        CSSConfig        `yaml:"css"`
	Footer     FooterConfig     `yaml:"footer"`
	Signature  SignatureConfig  `yaml:"signature"`
	Assets     AssetsConfig     `yaml:"assets"`
	Page       PageConfig       `yaml:"page"`
	Watermark  WatermarkConfig  `yaml:"watermark"`
	Cover      CoverConfig      `yaml:"cover"`
	TOC        TOCConfig        `yaml:"toc"`
	PageBreaks PageBreaksConfig `yaml:"pageBreaks"`
}

// AuthorConfig holds shared author metadata used by cover and signature.
type AuthorConfig struct {
	Name         string `yaml:"name"`
	Title        string `yaml:"title"`
	Email        string `yaml:"email"`
	Organization string `yaml:"organization"`
}

// Validate checks author field lengths.
func (a *AuthorConfig) Validate() error {
	if err := validateFieldLength("author.name", a.Name, MaxNameLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.title", a.Title, MaxTitleLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.email", a.Email, MaxEmailLength); err != nil {
		return err
	}
	if err := validateFieldLength("author.organization", a.Organization, MaxOrganizationLength); err != nil {
		return err
	}
	return nil
}

// DocumentConfig holds shared document metadata used by cover and footer.
type DocumentConfig struct {
	Title    string `yaml:"title"`    // "" = auto per-file (H1 → filename)
	Subtitle string `yaml:"subtitle"` // Optional subtitle
	Version  string `yaml:"version"`  // Version string (used in cover and footer)
	Date     string `yaml:"date"`     // "auto" = YYYY-MM-DD at startup
}

// Validate checks document field lengths.
func (d *DocumentConfig) Validate() error {
	if err := validateFieldLength("document.title", d.Title, MaxDocTitleLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.subtitle", d.Subtitle, MaxSubtitleLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.version", d.Version, MaxVersionLength); err != nil {
		return err
	}
	if err := validateFieldLength("document.date", d.Date, MaxDateLength); err != nil {
		return err
	}
	return nil
}

// InputConfig defines input source options.
type InputConfig struct {
	DefaultDir string `yaml:"defaultDir"` // Default input directory (empty = must specify)
}

// OutputConfig defines output destination options.
type OutputConfig struct {
	DefaultDir string `yaml:"defaultDir"` // Default output directory (empty = same as source)
}

// CSSConfig defines CSS styling options.
type CSSConfig struct {
	Style string `yaml:"style"` // Name of style in internal/assets/styles/ (empty = no CSS)
}

// FooterConfig defines page footer options.
// Uses document.date and document.version for date/status display.
type FooterConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Position       string `yaml:"position"`       // "left", "center", "right" (default: "right")
	ShowPageNumber bool   `yaml:"showPageNumber"` // Show page numbers
	Text           string `yaml:"text"`           // Optional free-form text
}

// Validate checks footer field values.
func (f *FooterConfig) Validate() error {
	if err := validateFieldLength("footer.text", f.Text, MaxTextLength); err != nil {
		return err
	}
	if f.Position != "" {
		switch strings.ToLower(f.Position) {
		case "left", "center", "right":
			// valid
		default:
			return fmt.Errorf("footer.position: invalid value %q (must be left, center, or right)", f.Position)
		}
	}
	return nil
}

// SignatureConfig defines signature block options.
// Uses author.name, author.title, author.email, author.organization for display.
type SignatureConfig struct {
	Enabled   bool   `yaml:"enabled"`
	ImagePath string `yaml:"imagePath"` // Signature image path or URL
	Links     []Link `yaml:"links"`     // Additional links
}

// Validate checks signature field values.
func (s *SignatureConfig) Validate() error {
	if err := validateFieldLength("signature.imagePath", s.ImagePath, MaxURLLength); err != nil {
		return err
	}
	for i, link := range s.Links {
		if err := validateFieldLength(fmt.Sprintf("signature.links[%d].label", i), link.Label, MaxLabelLength); err != nil {
			return err
		}
		if err := validateFieldLength(fmt.Sprintf("signature.links[%d].url", i), link.URL, MaxURLLength); err != nil {
			return err
		}
	}
	return nil
}

// Link represents a clickable link in the signature.
type Link struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// AssetsConfig defines asset loading options.
type AssetsConfig struct {
	BasePath string `yaml:"basePath"` // Empty = use embedded assets
}

// PageConfig defines PDF page settings.
type PageConfig struct {
	Size        string  `yaml:"size"`        // "letter", "a4", "legal" (default: "letter")
	Orientation string  `yaml:"orientation"` // "portrait", "landscape" (default: "portrait")
	Margin      float64 `yaml:"margin"`      // inches (default: 0.5)
}

// Validate checks page field values.
func (p *PageConfig) Validate() error {
	if err := validateFieldLength("page.size", p.Size, MaxPageSizeLength); err != nil {
		return err
	}
	if err := validateFieldLength("page.orientation", p.Orientation, MaxOrientationLength); err != nil {
		return err
	}
	return nil
}

// WatermarkConfig defines background watermark options.
type WatermarkConfig struct {
	Enabled bool    `yaml:"enabled"`
	Text    string  `yaml:"text"`    // Text to display (e.g., "DRAFT", "CONFIDENTIAL")
	Color   string  `yaml:"color"`   // Hex color (default: "#888888")
	Opacity float64 `yaml:"opacity"` // 0.0 to 1.0 (default: 0.1)
	Angle   float64 `yaml:"angle"`   // Rotation in degrees (default: -45)
}

// Validate checks watermark field values.
func (w *WatermarkConfig) Validate() error {
	if !w.Enabled {
		return nil
	}
	if w.Text == "" {
		return fmt.Errorf("watermark.text: required when watermark is enabled")
	}
	if err := validateFieldLength("watermark.text", w.Text, MaxWatermarkTextLength); err != nil {
		return err
	}
	if err := validateFieldLength("watermark.color", w.Color, MaxWatermarkColorLength); err != nil {
		return err
	}
	if w.Opacity < md2pdf.MinWatermarkOpacity || w.Opacity > md2pdf.MaxWatermarkOpacity {
		return fmt.Errorf("watermark.opacity: must be between %.1f and %.1f, got %.2f", md2pdf.MinWatermarkOpacity, md2pdf.MaxWatermarkOpacity, w.Opacity)
	}
	if w.Angle < md2pdf.MinWatermarkAngle || w.Angle > md2pdf.MaxWatermarkAngle {
		return fmt.Errorf("watermark.angle: must be between %.0f and %.0f, got %.2f", md2pdf.MinWatermarkAngle, md2pdf.MaxWatermarkAngle, w.Angle)
	}
	return nil
}

// CoverConfig defines cover page options.
// Uses author.* and document.* for author info and metadata.
type CoverConfig struct {
	Enabled bool   `yaml:"enabled"`
	Logo    string `yaml:"logo"` // Logo path or URL (cover-specific)
}

// Validate checks cover field values.
func (c *CoverConfig) Validate() error {
	if err := validateFieldLength("cover.logo", c.Logo, MaxURLLength); err != nil {
		return err
	}
	return nil
}

// TOCConfig defines table of contents options.
type TOCConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Title    string `yaml:"title"`    // Empty = no title above TOC
	MaxDepth int    `yaml:"maxDepth"` // 1-6, default 3
}

// Validate checks TOC field values.
func (t *TOCConfig) Validate() error {
	if err := validateFieldLength("toc.title", t.Title, MaxTOCTitleLength); err != nil {
		return err
	}
	if t.Enabled && t.MaxDepth != 0 {
		if t.MaxDepth < 1 || t.MaxDepth > 6 {
			return fmt.Errorf("toc.maxDepth: must be between 1 and 6, got %d", t.MaxDepth)
		}
	}
	return nil
}

// PageBreaksConfig defines page break options.
type PageBreaksConfig struct {
	Enabled  bool `yaml:"enabled"`  // Enable page break features (default: true for orphan/widow)
	BeforeH1 bool `yaml:"beforeH1"` // Page break before H1 headings
	BeforeH2 bool `yaml:"beforeH2"` // Page break before H2 headings
	BeforeH3 bool `yaml:"beforeH3"` // Page break before H3 headings
	Orphans  int  `yaml:"orphans"`  // Min lines at page bottom (1-5, default 2)
	Widows   int  `yaml:"widows"`   // Min lines at page top (1-5, default 2)
}

// Validate checks page breaks field values.
func (pb *PageBreaksConfig) Validate() error {
	if pb.Orphans != 0 {
		if pb.Orphans < 1 || pb.Orphans > 5 {
			return fmt.Errorf("pageBreaks.orphans: must be between 1 and 5, got %d", pb.Orphans)
		}
	}
	if pb.Widows != 0 {
		if pb.Widows < 1 || pb.Widows > 5 {
			return fmt.Errorf("pageBreaks.widows: must be between 1 and 5, got %d", pb.Widows)
		}
	}
	return nil
}

// Validate checks field lengths to prevent abuse in multi-tenant scenarios.
// Called automatically by LoadConfig, but available for consumers
// who construct Config manually (e.g., API adapters, library users).
func (c *Config) Validate() error {
	if err := c.Author.Validate(); err != nil {
		return err
	}
	if err := c.Document.Validate(); err != nil {
		return err
	}
	if err := c.Footer.Validate(); err != nil {
		return err
	}
	if err := c.Signature.Validate(); err != nil {
		return err
	}
	if err := c.Page.Validate(); err != nil {
		return err
	}
	if err := c.Watermark.Validate(); err != nil {
		return err
	}
	if err := c.Cover.Validate(); err != nil {
		return err
	}
	if err := c.TOC.Validate(); err != nil {
		return err
	}
	if err := c.PageBreaks.Validate(); err != nil {
		return err
	}
	return nil
}

// validateFieldLength checks if a field exceeds its maximum allowed length.
func validateFieldLength(fieldName, value string, maxLength int) error {
	if len(value) > maxLength {
		return fmt.Errorf("%w: %s (%d chars, max %d)", ErrFieldTooLong, fieldName, len(value), maxLength)
	}
	return nil
}

// DefaultConfig returns a neutral configuration with all features disabled.
func DefaultConfig() *Config {
	return &Config{
		Author:     AuthorConfig{},
		Document:   DocumentConfig{},
		Input:      InputConfig{DefaultDir: ""},
		Output:     OutputConfig{DefaultDir: ""},
		CSS:        CSSConfig{Style: ""},
		Footer:     FooterConfig{Enabled: false},
		Signature:  SignatureConfig{Enabled: false},
		Assets:     AssetsConfig{BasePath: ""},
		Page:       PageConfig{},
		Watermark:  WatermarkConfig{Enabled: false},
		Cover:      CoverConfig{Enabled: false},
		TOC:        TOCConfig{Enabled: false},
		PageBreaks: PageBreaksConfig{Enabled: false},
	}
}

// LoadConfig loads configuration from a file path or config name.
// If nameOrPath contains a path separator, it's treated as a file path.
// Otherwise, it's treated as a config name and searched in standard locations.
// Returns error if the file is not found (no silent fallback).
func LoadConfig(nameOrPath string) (*Config, error) {
	if nameOrPath == "" {
		return nil, ErrEmptyConfigName
	}

	var configPath string
	var err error

	if isFilePath(nameOrPath) {
		configPath = nameOrPath
	} else {
		configPath, err = resolveConfigPath(nameOrPath)
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(configPath) // #nosec G304 -- config path is user-provided
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, configPath)
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yamlutil.UnmarshalStrict(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigParse, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// isFilePath returns true if the string looks like a file path.
func isFilePath(s string) bool {
	return strings.ContainsAny(s, "/\\")
}

// resolveConfigPath searches for a config file by name in standard locations.
// Tries extensions in order: .yaml, .yml
// Tries locations in order: current directory, ~/.config/go-md2pdf/
func resolveConfigPath(name string) (string, error) {
	extensions := []string{".yaml", ".yml"}
	triedPaths := make([]string, 0, len(extensions)*2) // 2 locations

	// Try current directory first (both extensions)
	for _, ext := range extensions {
		localPath := name + ext
		if fileExists(localPath) {
			return localPath, nil
		}
		triedPaths = append(triedPaths, localPath)
	}

	// Try user config directory (both extensions)
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		for _, ext := range extensions {
			userPath := filepath.Join(userConfigDir, "go-md2pdf", name+ext)
			if fileExists(userPath) {
				return userPath, nil
			}
			triedPaths = append(triedPaths, userPath)
		}
	}

	return "", fmt.Errorf("%w: tried %s", ErrConfigNotFound, strings.Join(triedPaths, ", "))
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
