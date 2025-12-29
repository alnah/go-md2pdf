package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Input.DefaultDir != "" {
		t.Errorf("Input.DefaultDir = %q, want empty", cfg.Input.DefaultDir)
	}
	if cfg.Output.DefaultDir != "" {
		t.Errorf("Output.DefaultDir = %q, want empty", cfg.Output.DefaultDir)
	}
	if cfg.CSS.Style != "" {
		t.Errorf("CSS.Style = %q, want empty", cfg.CSS.Style)
	}
	if cfg.Footer.Enabled {
		t.Error("Footer.Enabled = true, want false")
	}
	if cfg.Signature.Enabled {
		t.Error("Signature.Enabled = true, want false")
	}
	if cfg.Assets.BasePath != "" {
		t.Errorf("Assets.BasePath = %q, want empty", cfg.Assets.BasePath)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("empty name returns ErrEmptyConfigName", func(t *testing.T) {
		_, err := LoadConfig("")
		if !errors.Is(err, ErrEmptyConfigName) {
			t.Errorf("error = %v, want ErrEmptyConfigName", err)
		}
	})

	t.Run("valid file path loads config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `css:
  style: "default"
footer:
  enabled: true
  position: "center"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if cfg.CSS.Style != "default" {
			t.Errorf("CSS.Style = %q, want %q", cfg.CSS.Style, "default")
		}
		if !cfg.Footer.Enabled {
			t.Error("Footer.Enabled = false, want true")
		}
		if cfg.Footer.Position != "center" {
			t.Errorf("Footer.Position = %q, want %q", cfg.Footer.Position, "center")
		}
	})

	t.Run("loads input and output directories", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `input:
  defaultDir: "/path/to/input"
output:
  defaultDir: "/path/to/output"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if cfg.Input.DefaultDir != "/path/to/input" {
			t.Errorf("Input.DefaultDir = %q, want %q", cfg.Input.DefaultDir, "/path/to/input")
		}
		if cfg.Output.DefaultDir != "/path/to/output" {
			t.Errorf("Output.DefaultDir = %q, want %q", cfg.Output.DefaultDir, "/path/to/output")
		}
	})

	t.Run("nonexistent file path returns ErrConfigNotFound", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/config.yaml")
		if !errors.Is(err, ErrConfigNotFound) {
			t.Errorf("error = %v, want ErrConfigNotFound", err)
		}
	})

	t.Run("invalid YAML returns ErrConfigParse", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "invalid.yaml")
		if err := os.WriteFile(configPath, []byte("css:\n  style: [unclosed"), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrConfigParse) {
			t.Errorf("error = %v, want ErrConfigParse", err)
		}
	})

	t.Run("unknown field returns ErrConfigParse in strict mode", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "unknown.yaml")
		content := `css:
  style: "default"
unknownField: "should fail"
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrConfigParse) {
			t.Errorf("error = %v, want ErrConfigParse", err)
		}
	})

	t.Run("unreadable file returns read error not ErrConfigNotFound", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "unreadable.yaml")
		if err := os.WriteFile(configPath, []byte("css:\n  style: test\n"), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := os.Chmod(configPath, 0000); err != nil {
			t.Fatalf("setup chmod: %v", err)
		}
		defer os.Chmod(configPath, 0600)

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Fatal("expected error for unreadable file")
		}
		if errors.Is(err, ErrConfigNotFound) {
			t.Error("error should not be ErrConfigNotFound for permission error")
		}
	})

	t.Run("config name resolves yaml in current directory", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "myconfig.yaml")
		if err := os.WriteFile(configPath, []byte("css:\n  style: fromname\n"), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		cfg, err := LoadConfig("myconfig")
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if cfg.CSS.Style != "fromname" {
			t.Errorf("CSS.Style = %q, want %q", cfg.CSS.Style, "fromname")
		}
	})

	t.Run("config name resolves yml when yaml not found", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "myconfig.yml")
		if err := os.WriteFile(configPath, []byte("css:\n  style: fromyml\n"), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		cfg, err := LoadConfig("myconfig")
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if cfg.CSS.Style != "fromyml" {
			t.Errorf("CSS.Style = %q, want %q", cfg.CSS.Style, "fromyml")
		}
	})

	t.Run("config name prefers yaml over yml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "myconfig.yaml"), []byte("css:\n  style: yaml\n"), 0600); err != nil {
			t.Fatalf("setup yaml: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "myconfig.yml"), []byte("css:\n  style: yml\n"), 0600); err != nil {
			t.Fatalf("setup yml: %v", err)
		}

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		cfg, err := LoadConfig("myconfig")
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if cfg.CSS.Style != "yaml" {
			t.Errorf("CSS.Style = %q, want %q (should prefer .yaml)", cfg.CSS.Style, "yaml")
		}
	})

	t.Run("config name resolves from user config directory", func(t *testing.T) {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			t.Skip("cannot get user config dir")
		}

		appConfigDir := filepath.Join(userConfigDir, "go-md2pdf")
		configPath := filepath.Join(appConfigDir, "testconfig.yaml")

		if err := os.MkdirAll(appConfigDir, 0755); err != nil {
			t.Fatalf("setup mkdir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("css:\n  style: userdir\n"), 0600); err != nil {
			t.Fatalf("setup write: %v", err)
		}
		defer os.Remove(configPath)

		// Change to empty dir so local file isn't found
		dir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		cfg, err := LoadConfig("testconfig")
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if cfg.CSS.Style != "userdir" {
			t.Errorf("CSS.Style = %q, want %q", cfg.CSS.Style, "userdir")
		}
	})

	t.Run("config name not found returns ErrConfigNotFound", func(t *testing.T) {
		dir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		_, err := LoadConfig("nonexistent")
		if !errors.Is(err, ErrConfigNotFound) {
			t.Errorf("error = %v, want ErrConfigNotFound", err)
		}
	})
}
