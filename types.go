package md2pdf

import "time"

// Input contains conversion parameters.
type Input struct {
	Markdown  string     // Markdown content (required)
	CSS       string     // Custom CSS (optional)
	Footer    *Footer    // Footer config (optional)
	Signature *Signature // Signature config (optional)
}

// Footer configures the PDF footer.
type Footer struct {
	Position       string // "left", "center", "right" (default: "right")
	ShowPageNumber bool
	Date           string
	Status         string
	Text           string
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
type Option func(*serviceConfig)

// serviceConfig holds internal configuration for Service.
type serviceConfig struct {
	timeout time.Duration
}

// defaultTimeout is used when no timeout is specified.
const defaultTimeout = 30 * time.Second

// WithTimeout sets the conversion timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *serviceConfig) {
		c.timeout = d
	}
}
