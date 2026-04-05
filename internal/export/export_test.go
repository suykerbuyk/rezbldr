// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package export

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOutputPath(t *testing.T) {
	tests := []struct {
		name   string
		source string
		format string
		outDir string
		want   string
	}{
		{
			name:   "docx",
			source: "/vault/resumes/generated/2026-04-05_acme_john-suykerbuyk_resume.md",
			format: "docx",
			outDir: "/vault/resumes/generated",
			want:   "/vault/resumes/generated/2026-04-05_acme_john-suykerbuyk_resume.docx",
		},
		{
			name:   "pdf",
			source: "/vault/resumes/generated/2026-04-05_acme_john-suykerbuyk_resume.md",
			format: "pdf",
			outDir: "/output",
			want:   "/output/2026-04-05_acme_john-suykerbuyk_resume.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OutputPath(tt.source, tt.format, tt.outDir)
			if got != tt.want {
				t.Errorf("OutputPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindMatchingCoverLetter(t *testing.T) {
	// Create temp vault structure.
	root := t.TempDir()
	coverDir := filepath.Join(root, "cover-letters")
	if err := os.MkdirAll(coverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a cover letter file.
	coverFile := "2026-04-05_acme_john-suykerbuyk_cover.md"
	if err := os.WriteFile(filepath.Join(coverDir, coverFile), []byte("# Cover"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("match found", func(t *testing.T) {
		resume := filepath.Join(root, "resumes", "generated", "2026-04-05_acme_john-suykerbuyk_resume.md")
		got := FindMatchingCoverLetter(root, resume)
		want := filepath.Join(coverDir, coverFile)
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("no match", func(t *testing.T) {
		resume := filepath.Join(root, "resumes", "generated", "2026-04-05_globex_john-suykerbuyk_resume.md")
		got := FindMatchingCoverLetter(root, resume)
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("short name", func(t *testing.T) {
		resume := "/vault/resumes/nodash.md"
		got := FindMatchingCoverLetter(root, resume)
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestExport_UnsupportedFormat(t *testing.T) {
	_, err := Export(Request{Source: "/dev/null", Format: "html"})
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestExport_MissingSource(t *testing.T) {
	_, err := Export(Request{Source: "/nonexistent/file.md", Format: "docx"})
	if err == nil {
		t.Fatal("expected error for missing source file")
	}
}

func TestExport_DefaultFormat(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not installed")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "test.md")
	os.WriteFile(source, []byte("# Hello\n\nWorld.\n"), 0o644)

	// Empty format should default to docx.
	result, err := Export(Request{Source: source, OutDir: dir})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if result.Format != "docx" {
		t.Errorf("default format = %q, want docx", result.Format)
	}
}

func TestExport_DefaultOutDir(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not installed")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "test.md")
	os.WriteFile(source, []byte("# Hello\n"), 0o644)

	// OutDir empty should default to source's directory.
	result, err := Export(Request{Source: source, Format: "docx"})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if filepath.Dir(result.Path) != dir {
		t.Errorf("output dir = %q, want %q", filepath.Dir(result.Path), dir)
	}
}

func TestExport_Integration(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not installed, skipping integration test")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "test_resume.md")
	content := `---
title: Test Resume
generated: 2026-04-05
---

# John Doe

## Experience

### Senior Engineer at Acme Corp

Built things.
`
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Export(Request{
		Source: source,
		Format: "docx",
		OutDir: dir,
	})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if result.Format != "docx" {
		t.Errorf("format = %q, want docx", result.Format)
	}
	if result.Size == 0 {
		t.Error("output file is empty")
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Errorf("output file not found: %v", err)
	}
}
