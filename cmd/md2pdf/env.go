package main

import (
	"io"
	"os"
	"time"

	"github.com/alnah/go-md2pdf/internal/assets"
	"github.com/alnah/go-md2pdf/internal/config"
)

// Environment holds injectable dependencies for testability.
// Includes I/O, time, configuration, and asset loading.
type Environment struct {
	Now         func() time.Time
	Stdout      io.Writer
	Stderr      io.Writer
	AssetLoader assets.AssetLoader
	Config      *config.Config // Loaded once, shared across pipeline
}

// DefaultEnv returns production environment with embedded assets.
func DefaultEnv() *Environment {
	return &Environment{
		Now:         time.Now,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		AssetLoader: assets.NewEmbeddedLoader(),
		Config:      config.DefaultConfig(),
	}
}
