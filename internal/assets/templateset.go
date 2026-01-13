package assets

// TemplateSet holds the HTML templates for document generation.
// A template set contains cover and signature templates that work together.
type TemplateSet struct {
	Name      string // Identifier (name or directory path)
	Cover     string // Cover page template HTML content
	Signature string // Signature block template HTML content
}

// DefaultTemplateSetName is the name of the built-in template set.
const DefaultTemplateSetName = "default"

// DefaultStyleName is the name of the built-in CSS style.
const DefaultStyleName = "default"
