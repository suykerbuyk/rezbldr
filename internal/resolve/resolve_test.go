// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

// setupVault creates a minimal vault directory tree for testing.
func setupVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		"jobs/target",
		"resumes/generated",
		"cover-letters",
		"experience",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		"jobs/target/2026-04-01_acme-corp.md":                                    "# Acme",
		"jobs/target/2026-04-05_globex-inc.md":                                   "# Globex",
		"resumes/generated/2026-04-01_acme-corp_john-suykerbuyk_resume.md":       "# Resume",
		"resumes/generated/2026-04-05_globex-inc_john-suykerbuyk_resume.md":      "# Resume",
		"cover-letters/2026-04-01_acme-corp_john-suykerbuyk_cover.md":            "# Cover",
		"experience/senior-storage-engineer-western-digital.md":                   "# WD",
		"experience/principal-engineer-seagate.md":                                "# Seagate",
	}
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestLatest(t *testing.T) {
	root := setupVault(t)

	tests := []struct {
		ft   FileType
		want string // expected basename
	}{
		{TypeJob, "2026-04-05_globex-inc.md"},
		{TypeResume, "2026-04-05_globex-inc_john-suykerbuyk_resume.md"},
		{TypeCover, "2026-04-01_acme-corp_john-suykerbuyk_cover.md"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ft), func(t *testing.T) {
			got, err := Latest(root, tt.ft)
			if err != nil {
				t.Fatalf("Latest(%s) error: %v", tt.ft, err)
			}
			if filepath.Base(got) != tt.want {
				t.Errorf("Latest(%s) = %q, want basename %q", tt.ft, got, tt.want)
			}
		})
	}
}

func TestLatest_EmptyDir(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "jobs", "target"), 0o755)

	_, err := Latest(root, TypeJob)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestLatest_UnknownType(t *testing.T) {
	_, err := Latest(t.TempDir(), FileType("bogus"))
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestGenerate(t *testing.T) {
	root := "/vault"

	tests := []struct {
		ft        FileType
		slug      string
		date      string
		candidate string
		want      string
	}{
		{TypeResume, "acme-corp", "2026-04-05", "john-suykerbuyk",
			"/vault/resumes/generated/2026-04-05_acme-corp_john-suykerbuyk_resume.md"},
		{TypeCover, "acme-corp", "2026-04-05", "john-suykerbuyk",
			"/vault/cover-letters/2026-04-05_acme-corp_john-suykerbuyk_cover.md"},
		{TypeJob, "acme-corp", "", "",
			"/vault/jobs/target/acme-corp.md"},
		{TypeExperience, "senior-engineer-acme", "", "",
			"/vault/experience/senior-engineer-acme.md"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ft)+"_"+tt.slug, func(t *testing.T) {
			got := Generate(root, tt.ft, tt.slug, tt.date, tt.candidate)
			if got != tt.want {
				t.Errorf("Generate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExists(t *testing.T) {
	root := setupVault(t)

	t.Run("exact match by slug", func(t *testing.T) {
		path, exists, alts := Exists(root, TypeJob, "acme-corp", "")
		if !exists {
			t.Fatalf("expected to find acme-corp job, got alternatives: %v", alts)
		}
		if filepath.Base(path) != "2026-04-01_acme-corp.md" {
			t.Errorf("got %q", path)
		}
	})

	t.Run("exact match by slug and date", func(t *testing.T) {
		path, exists, _ := Exists(root, TypeResume, "globex-inc", "2026-04-05")
		if !exists {
			t.Fatal("expected to find globex resume")
		}
		if filepath.Base(path) != "2026-04-05_globex-inc_john-suykerbuyk_resume.md" {
			t.Errorf("got %q", path)
		}
	})

	t.Run("no match returns alternatives", func(t *testing.T) {
		_, exists, alts := Exists(root, TypeJob, "initech", "")
		if exists {
			t.Fatal("expected no match")
		}
		if len(alts) != 0 {
			t.Errorf("expected no alternatives, got %v", alts)
		}
	})

	t.Run("partial match by date only", func(t *testing.T) {
		_, exists, alts := Exists(root, TypeResume, "initech", "2026-04-05")
		if exists {
			t.Fatal("expected no exact match")
		}
		if len(alts) == 0 {
			t.Fatal("expected alternatives from date match")
		}
	})
}

func TestParseFileType(t *testing.T) {
	tests := []struct {
		input string
		want  FileType
		err   bool
	}{
		{"job", TypeJob, false},
		{"Resume", TypeResume, false},
		{"COVER", TypeCover, false},
		{"experience", TypeExperience, false},
		{"bogus", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFileType(tt.input)
			if (err != nil) != tt.err {
				t.Fatalf("ParseFileType(%q) error = %v, wantErr %v", tt.input, err, tt.err)
			}
			if got != tt.want {
				t.Errorf("ParseFileType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
