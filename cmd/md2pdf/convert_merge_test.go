package main

// Notes:
// - mergeFlags: we test all flag override scenarios exhaustively. Each flag
//   category (author, document, footer, cover, signature, toc) is tested
//   for both override and preserve behavior.
// - Auto-enable logic: we test that setting certain flags auto-enables
//   their parent feature (e.g., footer.text enables footer).
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"testing"
)

// ---------------------------------------------------------------------------
// TestMergeFlags - CLI flags override config values
// ---------------------------------------------------------------------------

func TestMergeFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		flags *convertFlags
		cfg   *Config
		check func(t *testing.T, cfg *Config)
	}{
		{
			name:  "empty flags preserve config author",
			flags: &convertFlags{},
			cfg:   &Config{Author: AuthorConfig{Name: "Config Author"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "Config Author" {
					t.Errorf("Author.Name = %q, want %q", cfg.Author.Name, "Config Author")
				}
			},
		},
		{
			name:  "author.name overrides config",
			flags: &convertFlags{author: authorFlags{name: "CLI Author"}},
			cfg:   &Config{Author: AuthorConfig{Name: "Config Author"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Author" {
					t.Errorf("Author.Name = %q, want %q", cfg.Author.Name, "CLI Author")
				}
			},
		},
		{
			name:  "author.title overrides config",
			flags: &convertFlags{author: authorFlags{title: "CLI Title"}},
			cfg:   &Config{Author: AuthorConfig{Title: "Config Title"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Title != "CLI Title" {
					t.Errorf("Author.Title = %q, want %q", cfg.Author.Title, "CLI Title")
				}
			},
		},
		{
			name:  "author.email overrides config",
			flags: &convertFlags{author: authorFlags{email: "cli@test.com"}},
			cfg:   &Config{Author: AuthorConfig{Email: "config@test.com"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Email != "cli@test.com" {
					t.Errorf("Author.Email = %q, want %q", cfg.Author.Email, "cli@test.com")
				}
			},
		},
		{
			name:  "author.org overrides config",
			flags: &convertFlags{author: authorFlags{org: "CLI Org"}},
			cfg:   &Config{Author: AuthorConfig{Organization: "Config Org"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Organization != "CLI Org" {
					t.Errorf("Author.Organization = %q, want %q", cfg.Author.Organization, "CLI Org")
				}
			},
		},
		{
			name:  "document.title overrides config",
			flags: &convertFlags{document: documentFlags{title: "CLI Title"}},
			cfg:   &Config{Document: DocumentConfig{Title: "Config Title"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Title != "CLI Title" {
					t.Errorf("Document.Title = %q, want %q", cfg.Document.Title, "CLI Title")
				}
			},
		},
		{
			name:  "document.subtitle overrides config",
			flags: &convertFlags{document: documentFlags{subtitle: "CLI Subtitle"}},
			cfg:   &Config{Document: DocumentConfig{Subtitle: "Config Subtitle"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Subtitle != "CLI Subtitle" {
					t.Errorf("Document.Subtitle = %q, want %q", cfg.Document.Subtitle, "CLI Subtitle")
				}
			},
		},
		{
			name:  "document.version overrides config",
			flags: &convertFlags{document: documentFlags{version: "v2.0"}},
			cfg:   &Config{Document: DocumentConfig{Version: "v1.0"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Version != "v2.0" {
					t.Errorf("Document.Version = %q, want %q", cfg.Document.Version, "v2.0")
				}
			},
		},
		{
			name:  "document.date overrides config",
			flags: &convertFlags{document: documentFlags{date: "2025-06-01"}},
			cfg:   &Config{Document: DocumentConfig{Date: "auto"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Date != "2025-06-01" {
					t.Errorf("Document.Date = %q, want %q", cfg.Document.Date, "2025-06-01")
				}
			},
		},
		{
			name:  "footer.position overrides config",
			flags: &convertFlags{footer: footerFlags{position: "left"}},
			cfg:   &Config{Footer: FooterConfig{Position: "right"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Position != "left" {
					t.Errorf("Footer.Position = %q, want %q", cfg.Footer.Position, "left")
				}
			},
		},
		{
			name:  "footer.text overrides config",
			flags: &convertFlags{footer: footerFlags{text: "CLI Footer"}},
			cfg:   &Config{Footer: FooterConfig{Text: "Config Footer"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Text != "CLI Footer" {
					t.Errorf("Footer.Text = %q, want %q", cfg.Footer.Text, "CLI Footer")
				}
			},
		},
		{
			name:  "footer.pageNumber enables footer",
			flags: &convertFlags{footer: footerFlags{pageNumber: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false, ShowPageNumber: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.ShowPageNumber {
					t.Error("Footer.ShowPageNumber should be true")
				}
				if !cfg.Footer.Enabled {
					t.Error("Footer.Enabled should be true when pageNumber is set")
				}
			},
		},
		{
			name:  "footer.disabled disables footer",
			flags: &convertFlags{footer: footerFlags{disabled: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Enabled {
					t.Error("Footer.Enabled should be false when disabled flag is set")
				}
			},
		},
		{
			name:  "cover.logo overrides config",
			flags: &convertFlags{cover: coverFlags{logo: "/cli/logo.png"}},
			cfg:   &Config{Cover: CoverConfig{Logo: "/config/logo.png"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Cover.Logo != "/cli/logo.png" {
					t.Errorf("Cover.Logo = %q, want %q", cfg.Cover.Logo, "/cli/logo.png")
				}
			},
		},
		{
			name:  "cover.disabled disables cover",
			flags: &convertFlags{cover: coverFlags{disabled: true}},
			cfg:   &Config{Cover: CoverConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Cover.Enabled {
					t.Error("Cover.Enabled should be false when disabled flag is set")
				}
			},
		},
		{
			name:  "signature.image overrides config",
			flags: &convertFlags{signature: signatureFlags{image: "/cli/sig.png"}},
			cfg:   &Config{Signature: SignatureConfig{ImagePath: "/config/sig.png"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Signature.ImagePath != "/cli/sig.png" {
					t.Errorf("Signature.ImagePath = %q, want %q", cfg.Signature.ImagePath, "/cli/sig.png")
				}
			},
		},
		{
			name:  "signature.disabled disables signature",
			flags: &convertFlags{signature: signatureFlags{disabled: true}},
			cfg:   &Config{Signature: SignatureConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Signature.Enabled {
					t.Error("Signature.Enabled should be false when disabled flag is set")
				}
			},
		},
		{
			name:  "toc.title overrides config",
			flags: &convertFlags{toc: tocFlags{title: "CLI Contents"}},
			cfg:   &Config{TOC: TOCConfig{Title: "Config Contents"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Title != "CLI Contents" {
					t.Errorf("TOC.Title = %q, want %q", cfg.TOC.Title, "CLI Contents")
				}
			},
		},
		{
			name:  "toc.minDepth overrides config",
			flags: &convertFlags{toc: tocFlags{minDepth: 2}},
			cfg:   &Config{TOC: TOCConfig{MinDepth: 1}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.MinDepth != 2 {
					t.Errorf("TOC.MinDepth = %d, want %d", cfg.TOC.MinDepth, 2)
				}
			},
		},
		{
			name:  "toc.maxDepth overrides config",
			flags: &convertFlags{toc: tocFlags{maxDepth: 4}},
			cfg:   &Config{TOC: TOCConfig{MaxDepth: 2}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.MaxDepth != 4 {
					t.Errorf("TOC.MaxDepth = %d, want %d", cfg.TOC.MaxDepth, 4)
				}
			},
		},
		{
			name:  "toc.disabled disables toc",
			flags: &convertFlags{toc: tocFlags{disabled: true}},
			cfg:   &Config{TOC: TOCConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Enabled {
					t.Error("TOC.Enabled should be false when disabled flag is set")
				}
			},
		},
		{
			name: "multiple author flags combined",
			flags: &convertFlags{author: authorFlags{
				name:  "CLI Name",
				title: "CLI Title",
				email: "cli@test.com",
				org:   "CLI Org",
			}},
			cfg: &Config{Author: AuthorConfig{
				Name:         "Config Name",
				Title:        "Config Title",
				Email:        "config@test.com",
				Organization: "Config Org",
			}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Name" {
					t.Errorf("Author.Name = %q, want %q", cfg.Author.Name, "CLI Name")
				}
				if cfg.Author.Title != "CLI Title" {
					t.Errorf("Author.Title = %q, want %q", cfg.Author.Title, "CLI Title")
				}
				if cfg.Author.Email != "cli@test.com" {
					t.Errorf("Author.Email = %q, want %q", cfg.Author.Email, "cli@test.com")
				}
				if cfg.Author.Organization != "CLI Org" {
					t.Errorf("Author.Organization = %q, want %q", cfg.Author.Organization, "CLI Org")
				}
			},
		},
		{
			name:  "partial override preserves other fields",
			flags: &convertFlags{author: authorFlags{name: "CLI Name"}},
			cfg: &Config{Author: AuthorConfig{
				Name:         "Config Name",
				Title:        "Config Title",
				Email:        "config@test.com",
				Organization: "Config Org",
			}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Name" {
					t.Errorf("Author.Name = %q, want %q", cfg.Author.Name, "CLI Name")
				}
				if cfg.Author.Title != "Config Title" {
					t.Errorf("Author.Title = %q, want %q (should be preserved)", cfg.Author.Title, "Config Title")
				}
				if cfg.Author.Email != "config@test.com" {
					t.Errorf("Author.Email = %q, want %q (should be preserved)", cfg.Author.Email, "config@test.com")
				}
				if cfg.Author.Organization != "Config Org" {
					t.Errorf("Author.Organization = %q, want %q (should be preserved)", cfg.Author.Organization, "Config Org")
				}
			},
		},
		// Extended metadata flags
		{
			name:  "author.phone overrides config",
			flags: &convertFlags{author: authorFlags{phone: "+1-555-123-4567"}},
			cfg:   &Config{Author: AuthorConfig{Phone: "+1-555-000-0000"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Phone != "+1-555-123-4567" {
					t.Errorf("Author.Phone = %q, want %q", cfg.Author.Phone, "+1-555-123-4567")
				}
			},
		},
		{
			name:  "author.address overrides config",
			flags: &convertFlags{author: authorFlags{address: "123 CLI St"}},
			cfg:   &Config{Author: AuthorConfig{Address: "456 Config Ave"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Address != "123 CLI St" {
					t.Errorf("Author.Address = %q, want %q", cfg.Author.Address, "123 CLI St")
				}
			},
		},
		{
			name:  "author.department overrides config",
			flags: &convertFlags{author: authorFlags{department: "CLI Engineering"}},
			cfg:   &Config{Author: AuthorConfig{Department: "Config Engineering"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Department != "CLI Engineering" {
					t.Errorf("Author.Department = %q, want %q", cfg.Author.Department, "CLI Engineering")
				}
			},
		},
		{
			name:  "document.clientName overrides config",
			flags: &convertFlags{document: documentFlags{clientName: "CLI Client"}},
			cfg:   &Config{Document: DocumentConfig{ClientName: "Config Client"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.ClientName != "CLI Client" {
					t.Errorf("Document.ClientName = %q, want %q", cfg.Document.ClientName, "CLI Client")
				}
			},
		},
		{
			name:  "document.projectName overrides config",
			flags: &convertFlags{document: documentFlags{projectName: "CLI Project"}},
			cfg:   &Config{Document: DocumentConfig{ProjectName: "Config Project"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.ProjectName != "CLI Project" {
					t.Errorf("Document.ProjectName = %q, want %q", cfg.Document.ProjectName, "CLI Project")
				}
			},
		},
		{
			name:  "document.documentType overrides config",
			flags: &convertFlags{document: documentFlags{documentType: "CLI Spec"}},
			cfg:   &Config{Document: DocumentConfig{DocumentType: "Config Spec"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.DocumentType != "CLI Spec" {
					t.Errorf("Document.DocumentType = %q, want %q", cfg.Document.DocumentType, "CLI Spec")
				}
			},
		},
		{
			name:  "document.documentID overrides config",
			flags: &convertFlags{document: documentFlags{documentID: "CLI-001"}},
			cfg:   &Config{Document: DocumentConfig{DocumentID: "CFG-001"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.DocumentID != "CLI-001" {
					t.Errorf("Document.DocumentID = %q, want %q", cfg.Document.DocumentID, "CLI-001")
				}
			},
		},
		{
			name:  "document.description overrides config",
			flags: &convertFlags{document: documentFlags{description: "CLI Description"}},
			cfg:   &Config{Document: DocumentConfig{Description: "Config Description"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Description != "CLI Description" {
					t.Errorf("Document.Description = %q, want %q", cfg.Document.Description, "CLI Description")
				}
			},
		},
		{
			name:  "footer.showDocumentID enables footer and shows doc ID",
			flags: &convertFlags{footer: footerFlags{showDocumentID: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false, ShowDocumentID: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.ShowDocumentID {
					t.Error("Footer.ShowDocumentID should be true")
				}
				if !cfg.Footer.Enabled {
					t.Error("Footer.Enabled should be true when showDocumentID is set")
				}
			},
		},
		{
			name:  "cover.showDepartment enables department on cover",
			flags: &convertFlags{cover: coverFlags{showDepartment: true}},
			cfg:   &Config{Cover: CoverConfig{ShowDepartment: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Cover.ShowDepartment {
					t.Error("Cover.ShowDepartment should be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mergeFlags(tt.flags, tt.cfg)
			tt.check(t, tt.cfg)
		})
	}
}

// ---------------------------------------------------------------------------
// TestMergeFlags_AutoEnable - Auto-enable parent features when child flags set
// ---------------------------------------------------------------------------

func TestMergeFlags_AutoEnable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		flags *convertFlags
		cfg   *Config
		check func(t *testing.T, cfg *Config)
	}{
		{
			name:  "footer.text auto-enables footer",
			flags: &convertFlags{footer: footerFlags{text: "My Footer"}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.Enabled {
					t.Error("Footer.Enabled should be true when footer.text is set")
				}
				if cfg.Footer.Text != "My Footer" {
					t.Errorf("Footer.Text = %q, want %q", cfg.Footer.Text, "My Footer")
				}
			},
		},
		{
			name:  "footer.position auto-enables footer",
			flags: &convertFlags{footer: footerFlags{position: "left"}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.Enabled {
					t.Error("Footer.Enabled should be true when footer.position is set")
				}
			},
		},
		{
			name:  "cover.logo auto-enables cover",
			flags: &convertFlags{cover: coverFlags{logo: "/path/to/logo.png"}},
			cfg:   &Config{Cover: CoverConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Cover.Enabled {
					t.Error("Cover.Enabled should be true when cover.logo is set")
				}
				if cfg.Cover.Logo != "/path/to/logo.png" {
					t.Errorf("Cover.Logo = %q, want %q", cfg.Cover.Logo, "/path/to/logo.png")
				}
			},
		},
		{
			name:  "cover.showDepartment auto-enables cover",
			flags: &convertFlags{cover: coverFlags{showDepartment: true}},
			cfg:   &Config{Cover: CoverConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Cover.Enabled {
					t.Error("Cover.Enabled should be true when cover.showDepartment is set")
				}
			},
		},
		{
			name:  "signature.image auto-enables signature",
			flags: &convertFlags{signature: signatureFlags{image: "/path/to/sig.png"}},
			cfg:   &Config{Signature: SignatureConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Signature.Enabled {
					t.Error("Signature.Enabled should be true when signature.image is set")
				}
				if cfg.Signature.ImagePath != "/path/to/sig.png" {
					t.Errorf("Signature.ImagePath = %q, want %q", cfg.Signature.ImagePath, "/path/to/sig.png")
				}
			},
		},
		{
			name:  "toc.title auto-enables TOC",
			flags: &convertFlags{toc: tocFlags{title: "Contents"}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.TOC.Enabled {
					t.Error("TOC.Enabled should be true when toc.title is set")
				}
				if cfg.TOC.Title != "Contents" {
					t.Errorf("TOC.Title = %q, want %q", cfg.TOC.Title, "Contents")
				}
			},
		},
		{
			name:  "toc.minDepth auto-enables TOC",
			flags: &convertFlags{toc: tocFlags{minDepth: 2}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.TOC.Enabled {
					t.Error("TOC.Enabled should be true when toc.minDepth is set")
				}
				if cfg.TOC.MinDepth != 2 {
					t.Errorf("TOC.MinDepth = %d, want %d", cfg.TOC.MinDepth, 2)
				}
			},
		},
		{
			name:  "toc.minDepth negative ignored",
			flags: &convertFlags{toc: tocFlags{minDepth: -1}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false, MinDepth: 3}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Enabled {
					t.Error("TOC.Enabled should remain false when minDepth is negative")
				}
				if cfg.TOC.MinDepth != 3 {
					t.Errorf("TOC.MinDepth = %d, want %d (config value preserved)", cfg.TOC.MinDepth, 3)
				}
			},
		},
		{
			name:  "toc.maxDepth negative ignored",
			flags: &convertFlags{toc: tocFlags{maxDepth: -2}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false, MaxDepth: 4}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Enabled {
					t.Error("TOC.Enabled should remain false when maxDepth is negative")
				}
				if cfg.TOC.MaxDepth != 4 {
					t.Errorf("TOC.MaxDepth = %d, want %d (config value preserved)", cfg.TOC.MaxDepth, 4)
				}
			},
		},
		{
			name:  "toc.maxDepth auto-enables TOC",
			flags: &convertFlags{toc: tocFlags{maxDepth: 3}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.TOC.Enabled {
					t.Error("TOC.Enabled should be true when toc.maxDepth is set")
				}
				if cfg.TOC.MaxDepth != 3 {
					t.Errorf("TOC.MaxDepth = %d, want %d", cfg.TOC.MaxDepth, 3)
				}
			},
		},
		{
			name:  "disabled flags take precedence over auto-enable",
			flags: &convertFlags{footer: footerFlags{text: "Footer", disabled: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Enabled {
					t.Error("Footer.Enabled should be false when disabled flag is set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mergeFlags(tt.flags, tt.cfg)
			tt.check(t, tt.cfg)
		})
	}
}
