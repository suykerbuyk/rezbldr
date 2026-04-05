package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractYear(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"2024", 2024, false},
		{"2024-08", 2024, false},
		{"2026-04-02", 2026, false},
		{"2026-04-03T14:30:00Z", 2026, false},
		{"1984", 1984, false},
		{"Present", 0, true},
		{"present", 0, true},
		{"", 0, true},
		{"abc", 0, true},
		{"12", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ExtractYear(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractYear(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadAndParse(t *testing.T) {
	// Use the contact fixture to test loadAndParse.
	path := filepath.Join("testdata", "vault", "profile", "contact.md")

	// Verify fixture exists relative to package.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("fixture not found at %s — run from project root", path)
	}

	var c Contact
	body, err := loadAndParse(path, &c)
	if err != nil {
		t.Fatalf("loadAndParse: %v", err)
	}

	if c.Name != "Test User" {
		t.Errorf("Name = %q, want %q", c.Name, "Test User")
	}
	if c.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", c.Email, "test@example.com")
	}

	// contact.md has no body (frontmatter only).
	if len(body) != 0 {
		t.Errorf("expected empty body for contact.md, got %d bytes", len(body))
	}
}

func TestLoadAndParse_NotFound(t *testing.T) {
	var c Contact
	_, err := loadAndParse("/nonexistent/file.md", &c)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// testdataPath returns the path to testdata relative to the vault package.
func testdataPath(parts ...string) string {
	elems := append([]string{"testdata"}, parts...)
	return filepath.Join(elems...)
}
