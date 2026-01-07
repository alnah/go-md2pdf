package config

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

func TestValidateFieldLength(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		maxLength int
		wantErr   bool
	}{
		{
			name:      "empty value is valid",
			fieldName: "test",
			value:     "",
			maxLength: 10,
			wantErr:   false,
		},
		{
			name:      "value at limit is valid",
			fieldName: "test",
			value:     "1234567890",
			maxLength: 10,
			wantErr:   false,
		},
		{
			name:      "value under limit is valid",
			fieldName: "test",
			value:     "12345",
			maxLength: 10,
			wantErr:   false,
		},
		{
			name:      "value over limit returns error",
			fieldName: "test.field",
			value:     "12345678901",
			maxLength: 10,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFieldLength(tt.fieldName, tt.value, tt.maxLength)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, ErrFieldTooLong) {
					t.Errorf("error = %v, want ErrFieldTooLong", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config passes validation", func(t *testing.T) {
		cfg := &Config{
			Signature: SignatureConfig{
				Name:  "John Doe",
				Title: "Developer",
				Email: "john@example.com",
				Links: []Link{
					{Label: "GitHub", URL: "https://github.com/johndoe"},
				},
			},
			Footer: FooterConfig{
				Date:   "2025-01-15",
				Status: "FINAL",
				Text:   "Confidential",
			},
		}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("signature.name too long returns error", func(t *testing.T) {
		cfg := &Config{
			Signature: SignatureConfig{
				Name: string(make([]byte, MaxNameLength+1)),
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("signature.email too long returns error", func(t *testing.T) {
		cfg := &Config{
			Signature: SignatureConfig{
				Email: string(make([]byte, MaxEmailLength+1)),
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("signature.links[].url too long returns error", func(t *testing.T) {
		cfg := &Config{
			Signature: SignatureConfig{
				Links: []Link{
					{Label: "Valid", URL: string(make([]byte, MaxURLLength+1))},
				},
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("footer.text too long returns error", func(t *testing.T) {
		cfg := &Config{
			Footer: FooterConfig{
				Text: string(make([]byte, MaxTextLength+1)),
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("footer.status too long returns error", func(t *testing.T) {
		cfg := &Config{
			Footer: FooterConfig{
				Status: string(make([]byte, MaxStatusLength+1)),
			},
		}
		err := cfg.Validate()
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("error = %v, want ErrFieldTooLong", err)
		}
	})
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

	t.Run("field too long returns ErrFieldTooLong", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "toolong.yaml")
		longName := string(make([]byte, MaxNameLength+1))
		content := "signature:\n  name: \"" + longName + "\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("error = %v, want ErrFieldTooLong", err)
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

	t.Run("loads page settings", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		content := `page:
  size: "a4"
  orientation: "landscape"
  margin: 1.0
`
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if cfg.Page.Size != "a4" {
			t.Errorf("Page.Size = %q, want %q", cfg.Page.Size, "a4")
		}
		if cfg.Page.Orientation != "landscape" {
			t.Errorf("Page.Orientation = %q, want %q", cfg.Page.Orientation, "landscape")
		}
		if cfg.Page.Margin != 1.0 {
			t.Errorf("Page.Margin = %v, want %v", cfg.Page.Margin, 1.0)
		}
	})

	t.Run("page.size too long returns ErrFieldTooLong", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		longSize := string(make([]byte, MaxPageSizeLength+1))
		content := "page:\n  size: \"" + longSize + "\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("error = %v, want ErrFieldTooLong", err)
		}
	})

	t.Run("page.orientation too long returns ErrFieldTooLong", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "test.yaml")
		longOrientation := string(make([]byte, MaxOrientationLength+1))
		content := "page:\n  orientation: \"" + longOrientation + "\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := LoadConfig(configPath)
		if !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("error = %v, want ErrFieldTooLong", err)
		}
	})
}
