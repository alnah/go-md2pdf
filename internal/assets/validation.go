package assets

import (
	"fmt"
	"strings"
)

// ValidateAssetName checks that an asset name is safe for use as a filename.
// Returns ErrInvalidAssetName if the name is empty or contains path separators,
// dots (which could allow extension manipulation), or traversal characters.
func ValidateAssetName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty name", ErrInvalidAssetName)
	}
	if strings.ContainsAny(name, "/\\.") {
		return fmt.Errorf("%w: %q", ErrInvalidAssetName, name)
	}
	return nil
}
