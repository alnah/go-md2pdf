package main

import (
	"io"
	"os"
	"time"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

// Environment holds injectable dependencies for testability.
// Includes I/O, time, configuration, and asset loading.
type Environment struct {
	Now         func() time.Time
	Stdout      io.Writer
	Stderr      io.Writer
	AssetLoader md2pdf.AssetLoader
	Config      *config.Config // Loaded once, shared across pipeline
}

// DefaultEnv returns production environment with embedded assets.
func DefaultEnv() *Environment {
	loader, err := md2pdf.NewAssetLoader("")
	if err != nil {
		// NewAssetLoader("") only fails if basePath validation fails.
		// Empty string bypasses validation, so this is unreachable.
		panic("md2pdf: embedded asset loader initialization failed: " + err.Error())
	}
	return &Environment{
		Now:         time.Now,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		AssetLoader: loader,
		Config:      config.DefaultConfig(),
	}
}
