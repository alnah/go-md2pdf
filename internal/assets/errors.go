package assets

import "errors"

// Sentinel errors for asset operations.
var (
	// ErrStyleNotFound indicates the requested style does not exist.
	ErrStyleNotFound = errors.New("style not found")

	// ErrTemplateNotFound indicates the requested template does not exist.
	ErrTemplateNotFound = errors.New("template not found")

	// ErrTemplateSetNotFound indicates the requested template set does not exist.
	ErrTemplateSetNotFound = errors.New("template set not found")

	// ErrIncompleteTemplateSet indicates the template set is missing required templates.
	ErrIncompleteTemplateSet = errors.New("template set missing required template")

	// ErrInvalidAssetName indicates the asset name contains invalid characters
	// such as path separators or traversal sequences.
	ErrInvalidAssetName = errors.New("invalid asset name")

	// ErrInvalidBasePath indicates the configured base path is not a valid directory.
	ErrInvalidBasePath = errors.New("invalid base path")

	// ErrAssetRead indicates an I/O error occurred while reading an asset file.
	ErrAssetRead = errors.New("failed to read asset")

	// ErrPathTraversal indicates an attempt to access files outside the base path.
	ErrPathTraversal = errors.New("path traversal detected")
)
