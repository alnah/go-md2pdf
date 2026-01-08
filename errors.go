package md2pdf

import "errors"

// Sentinel errors for library operations.
var (
	ErrEmptyMarkdown   = errors.New("markdown content cannot be empty")
	ErrHTMLConversion  = errors.New("HTML conversion failed")
	ErrPDFGeneration   = errors.New("PDF generation failed")
	ErrBrowserConnect  = errors.New("failed to connect to browser")
	ErrPageCreate      = errors.New("failed to create browser page")
	ErrPageLoad        = errors.New("failed to load page")
	ErrSignatureRender = errors.New("signature template rendering failed")

	// Page settings validation errors.
	ErrInvalidPageSize    = errors.New("invalid page size")
	ErrInvalidOrientation = errors.New("invalid orientation")
	ErrInvalidMargin      = errors.New("invalid margin")

	// Footer validation errors.
	ErrInvalidFooterPosition = errors.New("invalid footer position")
)
