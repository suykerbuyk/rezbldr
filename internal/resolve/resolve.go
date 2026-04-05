// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package resolve provides file path resolution and naming convention
// helpers for the rezbldr vault.
package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileType identifies a vault file category.
type FileType string

const (
	TypeJob        FileType = "job"
	TypeResume     FileType = "resume"
	TypeCover      FileType = "cover"
	TypeExperience FileType = "experience"
)

// dir returns the vault-relative directory for a file type.
func dir(vaultRoot string, ft FileType) string {
	switch ft {
	case TypeJob:
		return filepath.Join(vaultRoot, "jobs", "target")
	case TypeResume:
		return filepath.Join(vaultRoot, "resumes", "generated")
	case TypeCover:
		return filepath.Join(vaultRoot, "cover-letters")
	case TypeExperience:
		return filepath.Join(vaultRoot, "experience")
	default:
		return ""
	}
}

// Latest finds the most recent .md file of the given type by filename
// (date-prefixed filenames sort lexicographically in chronological order).
func Latest(vaultRoot string, ft FileType) (string, error) {
	d := dir(vaultRoot, ft)
	if d == "" {
		return "", fmt.Errorf("unknown file type %q", ft)
	}

	entries, err := os.ReadDir(d)
	if err != nil {
		return "", fmt.Errorf("read directory %s: %w", d, err)
	}

	var mds []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mds = append(mds, e.Name())
		}
	}
	if len(mds) == 0 {
		return "", fmt.Errorf("no %s files found in %s", ft, d)
	}

	sort.Strings(mds)
	return filepath.Join(d, mds[len(mds)-1]), nil
}

// Generate constructs a filename from components following vault naming conventions.
//
// Patterns:
//   - resume:  YYYY-MM-DD_company-slug_candidate-name_resume.md
//   - cover:   YYYY-MM-DD_company-slug_candidate-name_cover.md
//   - job:     returns slug.md in jobs/target/
//   - experience: returns slug.md in experience/
func Generate(vaultRoot string, ft FileType, slug, date, candidate string) string {
	d := dir(vaultRoot, ft)

	switch ft {
	case TypeResume:
		return filepath.Join(d, fmt.Sprintf("%s_%s_%s_resume.md", date, slug, candidate))
	case TypeCover:
		return filepath.Join(d, fmt.Sprintf("%s_%s_%s_cover.md", date, slug, candidate))
	case TypeJob:
		return filepath.Join(d, slug+".md")
	case TypeExperience:
		return filepath.Join(d, slug+".md")
	default:
		return ""
	}
}

// Exists checks if a file matching the given criteria exists. If no exact
// match is found, it returns alternative files that partially match the slug.
func Exists(vaultRoot string, ft FileType, slug, date string) (path string, exists bool, alternatives []string) {
	d := dir(vaultRoot, ft)
	if d == "" {
		return "", false, nil
	}

	entries, err := os.ReadDir(d)
	if err != nil {
		return "", false, nil
	}

	// Build search terms.
	slugLower := strings.ToLower(slug)
	dateLower := strings.ToLower(date)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.ToLower(e.Name())
		matchSlug := slugLower != "" && strings.Contains(name, slugLower)
		matchDate := dateLower != "" && strings.Contains(name, dateLower)

		if matchSlug && (dateLower == "" || matchDate) {
			// Exact match (both slug and date match, or date not specified).
			return filepath.Join(d, e.Name()), true, nil
		}
		if matchSlug || matchDate {
			alternatives = append(alternatives, filepath.Join(d, e.Name()))
		}
	}

	return "", false, alternatives
}

// ParseFileType converts a string to a FileType, returning an error for
// unrecognized types.
func ParseFileType(s string) (FileType, error) {
	switch strings.ToLower(s) {
	case "job":
		return TypeJob, nil
	case "resume":
		return TypeResume, nil
	case "cover":
		return TypeCover, nil
	case "experience":
		return TypeExperience, nil
	default:
		return "", fmt.Errorf("unknown file type %q (want job, resume, cover, or experience)", s)
	}
}
