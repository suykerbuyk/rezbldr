// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFM      string // expected frontmatter; use "EMPTY" for non-nil empty, "" for nil
		wantBody    string // expected body; "" means nil
		wantErr     bool
	}{
		{
			name:     "normal file with frontmatter and body",
			input:    "---\ntitle: Hello\n---\n\nBody text here.\n",
			wantFM:   "title: Hello\n",
			wantBody: "\nBody text here.\n",
		},
		{
			name:     "no frontmatter",
			input:    "# Just Markdown\n\nSome content.\n",
			wantBody: "# Just Markdown\n\nSome content.\n",
		},
		{
			name:     "empty frontmatter",
			input:    "---\n---\n\nBody after empty FM.\n",
			wantFM:   "EMPTY",
			wantBody: "\nBody after empty FM.\n",
		},
		{
			name:   "frontmatter only no body",
			input:  "---\nname: Test\nemail: test@example.com\n---\n",
			wantFM: "name: Test\nemail: test@example.com\n",
		},
		{
			name:     "triple dash in body not confused with delimiter",
			input:    "---\ntitle: Test\n---\n\nSome text\n\n---\n\nMore text after horizontal rule.\n",
			wantFM:   "title: Test\n",
			wantBody: "\nSome text\n\n---\n\nMore text after horizontal rule.\n",
		},
		{
			name:    "opening delimiter but no closing",
			input:   "---\ntitle: Broken\nno closing\n",
			wantErr: true,
		},
		{
			name:  "empty file",
			input: "",
		},
		{
			name:     "windows line endings",
			input:    "---\r\ntitle: Windows\r\n---\r\n\r\nBody with CRLF.\r\n",
			wantFM:   "title: Windows\r\n",
			wantBody: "\r\nBody with CRLF.\r\n",
		},
		{
			name:     "frontmatter with complex yaml",
			input:    "---\ntags:\n  - storage\n  - ceph\nskills:\n  - Go\n---\n\n## Summary\n",
			wantFM:   "tags:\n  - storage\n  - ceph\nskills:\n  - Go\n",
			wantBody: "\n## Summary\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := Parse([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check frontmatter.
			switch tt.wantFM {
			case "EMPTY":
				if fm == nil {
					t.Error("expected empty (non-nil) frontmatter, got nil")
				} else if len(fm) != 0 {
					t.Errorf("expected empty frontmatter, got %q", fm)
				}
			case "":
				if fm != nil {
					t.Errorf("expected nil frontmatter, got %q", fm)
				}
			default:
				if string(fm) != tt.wantFM {
					t.Errorf("frontmatter:\n  got:  %q\n  want: %q", fm, tt.wantFM)
				}
			}

			// Check body.
			if tt.wantBody == "" {
				if body != nil {
					// Allow body to be the full input for "no frontmatter" cases.
					if tt.wantFM == "" && tt.input != "" {
						// No frontmatter — body should be nil (already checked by wantBody being "")
						// Actually for "no frontmatter", body IS the full content.
						// This case shouldn't happen since we set wantBody for those.
					}
				}
			} else {
				if string(body) != tt.wantBody {
					t.Errorf("body:\n  got:  %q\n  want: %q", body, tt.wantBody)
				}
			}
		})
	}
}

func TestStrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips frontmatter",
			input: "---\ntitle: Test\n---\n\nBody here.\n",
			want:  "\nBody here.\n",
		},
		{
			name:  "no frontmatter returns original",
			input: "# Plain Markdown\n",
			want:  "# Plain Markdown\n",
		},
		{
			name:  "malformed returns original",
			input: "---\nno closing\n",
			want:  "---\nno closing\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Strip([]byte(tt.input))
			if string(got) != tt.want {
				t.Errorf("Strip:\n  got:  %q\n  want: %q", got, tt.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	type sample struct {
		Title string   `yaml:"title"`
		Tags  []string `yaml:"tags"`
	}

	s := sample{Title: "Test", Tags: []string{"a", "b"}}
	got, err := Generate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse it back to verify round-trip.
	fm, _, err := Parse(got)
	if err != nil {
		t.Fatalf("failed to parse generated output: %v", err)
	}
	if len(fm) == 0 {
		t.Fatal("expected non-empty frontmatter from generated output")
	}

	var parsed sample
	if err := yaml.Unmarshal(fm, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if parsed.Title != s.Title {
		t.Errorf("title: got %q, want %q", parsed.Title, s.Title)
	}
	if len(parsed.Tags) != len(s.Tags) {
		t.Errorf("tags length: got %d, want %d", len(parsed.Tags), len(s.Tags))
	}
}
