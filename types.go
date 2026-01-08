package md2pdf

import (
	"fmt"
	"strings"
	"time"
)

// Page size constants.
const (
	PageSizeLetter = "letter"
	PageSizeA4     = "a4"
	PageSizeLegal  = "legal"
)

// Orientation constants.
const (
	OrientationPortrait  = "portrait"
	OrientationLandscape = "landscape"
)

// Margin bounds in inches.
const (
	MinMargin     = 0.25
	MaxMargin     = 3.0
	DefaultMargin = 0.5
)

// PageSettings configures PDF page dimensions.
type PageSettings struct {
	Size        string  // "letter", "a4", "legal"
	Orientation string  // "portrait", "landscape"
	Margin      float64 // inches, applied to all sides
}

// DefaultPageSettings returns page settings with default values.
func DefaultPageSettings() *PageSettings {
	return &PageSettings{
		Size:        PageSizeLetter,
		Orientation: OrientationPortrait,
		Margin:      DefaultMargin,
	}
}

// Validate checks that page settings are valid.
// Returns nil if p is nil (nil means use defaults).
// Does not mutate - uses case-insensitive comparison.
func (p *PageSettings) Validate() error {
	if p == nil {
		return nil
	}

	if !isValidPageSize(p.Size) {
		return fmt.Errorf("%w: %q", ErrInvalidPageSize, p.Size)
	}

	if !isValidOrientation(p.Orientation) {
		return fmt.Errorf("%w: %q", ErrInvalidOrientation, p.Orientation)
	}

	if p.Margin < MinMargin || p.Margin > MaxMargin {
		return fmt.Errorf("%w: %.2f (must be between %.2f and %.2f)", ErrInvalidMargin, p.Margin, MinMargin, MaxMargin)
	}

	return nil
}

// isValidPageSize checks if size is a known page size (case-insensitive).
func isValidPageSize(size string) bool {
	switch strings.ToLower(size) {
	case PageSizeLetter, PageSizeA4, PageSizeLegal:
		return true
	}
	return false
}

// isValidOrientation checks if orientation is valid (case-insensitive).
func isValidOrientation(orientation string) bool {
	switch strings.ToLower(orientation) {
	case OrientationPortrait, OrientationLandscape:
		return true
	}
	return false
}

// Input contains conversion parameters.
type Input struct {
	Markdown  string        // Markdown content (required)
	CSS       string        // Custom CSS (optional)
	Footer    *Footer       // Footer config (optional)
	Signature *Signature    // Signature config (optional)
	Page      *PageSettings // Page settings (optional, nil = defaults)
}

// Footer configures the PDF footer.
type Footer struct {
	Position       string // "left", "center", "right" (default: "right")
	ShowPageNumber bool
	Date           string
	Status         string
	Text           string
}

// Validate checks that footer settings are valid.
// Returns nil if f is nil (nil means no footer).
func (f *Footer) Validate() error {
	if f == nil {
		return nil
	}
	switch strings.ToLower(f.Position) {
	case "", "left", "center", "right":
		return nil
	default:
		return fmt.Errorf("%w: %q (must be left, center, or right)", ErrInvalidFooterPosition, f.Position)
	}
}

// Signature configures the signature block.
type Signature struct {
	Name      string
	Title     string
	Email     string
	ImagePath string
	Links     []Link
}

// Link represents a clickable link.
type Link struct {
	Label string
	URL   string
}

// Option configures a Service.
type Option func(*Service)

// serviceConfig holds internal configuration for Service.
type serviceConfig struct {
	timeout time.Duration
}

// defaultTimeout is used when no timeout is specified.
const defaultTimeout = 30 * time.Second

// WithTimeout sets the conversion timeout.
// Panics if d <= 0 (programmer error, similar to time.NewTicker).
func WithTimeout(d time.Duration) Option {
	if d <= 0 {
		panic("md2pdf: WithTimeout duration must be positive")
	}
	return func(s *Service) {
		s.cfg.timeout = d
	}
}
