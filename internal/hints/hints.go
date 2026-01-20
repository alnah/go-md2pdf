// Package hints provides actionable error hints for common failure scenarios.
// Hints are formatted consistently as "\n  hint: <text>" for appending to error messages.
package hints

import (
	"os"
	"strings"

	"github.com/alnah/go-md2pdf/internal/fileutil"
)

// IsInContainer detects if running inside a Docker container or similar.
// Checks for /.dockerenv file which Docker creates automatically.
var IsInContainer = func() bool {
	return fileutil.FileExists("/.dockerenv")
}

// ForBrowserConnect returns hints for browser connection errors.
// Detects CI/Docker environment and suggests relevant environment variables.
func ForBrowserConnect() string {
	var hints []string

	// Detect CI environment
	inCI := os.Getenv("CI") != "" ||
		os.Getenv("GITHUB_ACTIONS") != "" ||
		os.Getenv("GITLAB_CI") != "" ||
		os.Getenv("JENKINS_URL") != ""

	// Suggest ROD_NO_SANDBOX for container/CI environments
	if (inCI || IsInContainer()) && os.Getenv("ROD_NO_SANDBOX") != "1" {
		hints = append(hints, "set ROD_NO_SANDBOX=1 for Docker/CI")
	}

	// Suggest ROD_BROWSER_BIN if not set
	if os.Getenv("ROD_BROWSER_BIN") == "" {
		hints = append(hints, "set ROD_BROWSER_BIN to use custom Chrome")
	}

	return formatHints(hints)
}

// ForTimeout returns a hint about increasing timeout for slow operations.
func ForTimeout() string {
	return format("for large documents, use --timeout flag")
}

// ForConfigNotFound returns hints for config file not found errors.
// Suggests --config flag and creating a config in ~/.config/go-md2pdf/.
func ForConfigNotFound(searchedPaths []string) string {
	hint := "use --config /path/to/file.yaml"

	// Find a user config path (contains .config/go-md2pdf) to suggest
	for _, p := range searchedPaths {
		if strings.Contains(p, ".config/go-md2pdf") {
			hint += " or create " + p
			break
		}
	}

	return format(hint)
}

// ForOutputDirectory returns hints for output directory creation errors.
func ForOutputDirectory() string {
	return format("check parent directory exists and is writable")
}

// ForStyleNotFound returns hints for style not found errors.
func ForStyleNotFound(available []string) string {
	if len(available) == 0 {
		return ""
	}
	return format("available: " + strings.Join(available, ", "))
}

// ForSignatureImage returns hints for signature image not found errors.
func ForSignatureImage() string {
	return format("supported formats: PNG, JPG, SVG; use absolute path or URL")
}

// format creates a single hint string with consistent formatting.
func format(hint string) string {
	if hint == "" {
		return ""
	}
	return "\n  hint: " + hint
}

// formatHints joins multiple hints with consistent formatting.
func formatHints(hints []string) string {
	if len(hints) == 0 {
		return ""
	}
	return format(strings.Join(hints, "; "))
}
