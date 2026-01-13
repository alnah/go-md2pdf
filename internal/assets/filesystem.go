package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemLoader loads assets from a directory on the filesystem.
// Implements AssetLoader interface.
type FilesystemLoader struct {
	basePath string
}

// NewFilesystemLoader creates a FilesystemLoader for the given base path.
// Returns ErrInvalidBasePath if the path is not a valid, readable directory.
func NewFilesystemLoader(basePath string) (*FilesystemLoader, error) {
	if basePath == "" {
		return nil, fmt.Errorf("%w: empty path", ErrInvalidBasePath)
	}

	// Clean and resolve to absolute path
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBasePath, err)
	}

	// Resolve symlinks in base path for consistent comparisons
	// This ensures path containment checks work when basePath contains symlinks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = realPath
	}

	// Verify it's a readable directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: directory does not exist: %s", ErrInvalidBasePath, absPath)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidBasePath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: not a directory: %s", ErrInvalidBasePath, absPath)
	}

	// Verify read access by attempting to read directory
	if _, err := os.ReadDir(absPath); err != nil {
		return nil, fmt.Errorf("%w: cannot read directory: %v", ErrInvalidBasePath, err)
	}

	return &FilesystemLoader{basePath: absPath}, nil
}

// LoadStyle loads a CSS style from the filesystem.
// Looks for {basePath}/styles/{name}.css
func (f *FilesystemLoader) LoadStyle(name string) (string, error) {
	if err := ValidateAssetName(name); err != nil {
		return "", err
	}

	filePath := filepath.Join(f.basePath, "styles", name+".css")

	// Path containment check: ensure resolved path is within basePath
	if err := f.verifyPathContainment(filePath); err != nil {
		return "", err
	}

	content, err := os.ReadFile(filePath) // #nosec G304 -- path validated above
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %q", ErrStyleNotFound, name)
		}
		return "", fmt.Errorf("%w: %v", ErrAssetRead, err)
	}

	return string(content), nil
}

// LoadTemplateSet loads a set of HTML templates from the filesystem.
// Looks for {basePath}/templates/{name}/cover.html and signature.html
func (f *FilesystemLoader) LoadTemplateSet(name string) (*TemplateSet, error) {
	if err := ValidateAssetName(name); err != nil {
		return nil, err
	}

	dirPath := filepath.Join(f.basePath, "templates", name)

	// Path containment check for the directory
	if err := f.verifyPathContainment(dirPath + string(filepath.Separator)); err != nil {
		return nil, err
	}

	coverPath := filepath.Join(dirPath, "cover.html")
	sigPath := filepath.Join(dirPath, "signature.html")

	cover, coverErr := os.ReadFile(coverPath) // #nosec G304 -- path validated above
	signature, sigErr := os.ReadFile(sigPath) // #nosec G304 -- path validated above

	// If both files are missing, the template set doesn't exist
	if os.IsNotExist(coverErr) && os.IsNotExist(sigErr) {
		return nil, fmt.Errorf("%w: %q", ErrTemplateSetNotFound, name)
	}

	// Handle read errors (not just not-exist)
	if coverErr != nil && !os.IsNotExist(coverErr) {
		return nil, fmt.Errorf("%w: reading cover.html: %v", ErrAssetRead, coverErr)
	}
	if sigErr != nil && !os.IsNotExist(sigErr) {
		return nil, fmt.Errorf("%w: reading signature.html: %v", ErrAssetRead, sigErr)
	}

	// If only one file is missing, the template set is incomplete
	if os.IsNotExist(coverErr) {
		return nil, fmt.Errorf("%w: %q missing cover.html", ErrIncompleteTemplateSet, name)
	}
	if os.IsNotExist(sigErr) {
		return nil, fmt.Errorf("%w: %q missing signature.html", ErrIncompleteTemplateSet, name)
	}

	return &TemplateSet{
		Name:      name,
		Cover:     string(cover),
		Signature: string(signature),
	}, nil
}

// verifyPathContainment ensures the resolved file path is within basePath.
// Prevents path traversal attacks even if name validation is bypassed.
// Resolves symlinks to prevent escape via symlink pointing outside basePath.
func (f *FilesystemLoader) verifyPathContainment(filePath string) error {
	// Resolve to absolute path and clean
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve path", ErrPathTraversal)
	}

	// Resolve symlinks to get the real path
	// This prevents symlink-based escape attacks
	realPath, err := filepath.EvalSymlinks(absFilePath)
	if err == nil {
		// Use real path if symlink resolution succeeded
		absFilePath = realPath
	}
	// If EvalSymlinks fails (e.g., file doesn't exist yet), continue with absFilePath
	// The file will fail to open anyway, and we still do the prefix check

	// Ensure the file path starts with the base path
	// Add separator to prevent prefix attacks (e.g., /base/path vs /base/pathevil)
	if !strings.HasPrefix(absFilePath, f.basePath+string(filepath.Separator)) {
		return fmt.Errorf("%w: path escapes base directory", ErrPathTraversal)
	}

	return nil
}

// Compile-time interface check.
var _ AssetLoader = (*FilesystemLoader)(nil)
