package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alnah/go-md2pdf/internal/yamlutil"
)

// Sentinel errors for config operations.
var (
	ErrConfigNotFound  = errors.New("config file not found")
	ErrEmptyConfigName = errors.New("config name cannot be empty")
	ErrConfigParse     = errors.New("failed to parse config")
)

// Config holds all configuration for document generation.
type Config struct {
	Input     InputConfig     `yaml:"input"`
	Output    OutputConfig    `yaml:"output"`
	CSS       CSSConfig       `yaml:"css"`
	Footer    FooterConfig    `yaml:"footer"`
	Signature SignatureConfig `yaml:"signature"`
	Assets    AssetsConfig    `yaml:"assets"`
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
type FooterConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Position       string `yaml:"position"` // "left", "center", "right" (default: "right")
	ShowPageNumber bool   `yaml:"showPageNumber"`
	Date           string `yaml:"date"`   // Optional, format YYYY-MM-DD
	Status         string `yaml:"status"` // Optional: "DRAFT", "FINAL", "v1.2"
	Text           string `yaml:"text"`   // Optional free-form text
}

// SignatureConfig defines signature block options.
type SignatureConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Name      string `yaml:"name"`
	Title     string `yaml:"title"`
	Email     string `yaml:"email"`
	ImagePath string `yaml:"imagePath"`
	Links     []Link `yaml:"links"`
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

// DefaultConfig returns a neutral configuration with all features disabled.
func DefaultConfig() *Config {
	return &Config{
		Input:     InputConfig{DefaultDir: ""},
		Output:    OutputConfig{DefaultDir: ""},
		CSS:       CSSConfig{Style: ""},
		Footer:    FooterConfig{Enabled: false},
		Signature: SignatureConfig{Enabled: false},
		Assets:    AssetsConfig{BasePath: ""},
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
