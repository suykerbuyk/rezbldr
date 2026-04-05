// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package export provides a pandoc-based export pipeline for converting
// markdown resumes and cover letters to DOCX or PDF format.
package export

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// Request describes a single export operation.
type Request struct {
	Source   string // Resolved path to .md file.
	Format   string // "docx" or "pdf".
	Template string // Optional reference doc path.
	OutDir   string // Output directory (defaults to source's directory).
}

// Result describes the output of an export operation.
type Result struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Format string `json:"format"`
}

// Export converts a markdown file to the requested format via pandoc.
// It strips YAML frontmatter before passing content to pandoc.
func Export(req Request) (*Result, error) {
	if req.Format == "" {
		req.Format = "docx"
	}
	if req.Format != "docx" && req.Format != "pdf" {
		return nil, fmt.Errorf("unsupported format %q (want docx or pdf)", req.Format)
	}

	if _, err := exec.LookPath("pandoc"); err != nil {
		return nil, fmt.Errorf("pandoc not found in PATH: %w", err)
	}

	content, err := os.ReadFile(req.Source)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}

	stripped := vault.Strip(content)

	tmpFile, err := os.CreateTemp("", "rezbldr-export-*.md")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(stripped); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	outDir := req.OutDir
	if outDir == "" {
		outDir = filepath.Dir(req.Source)
	}

	outPath := OutputPath(req.Source, req.Format, outDir)

	args := []string{
		"--from=markdown",
		"--to=" + req.Format,
		"-o", outPath,
	}

	if req.Format == "docx" && req.Template != "" {
		args = append(args, "--reference-doc="+req.Template)
	}
	if req.Format == "pdf" {
		args = append(args, "--pdf-engine=xelatex")
	}

	args = append(args, tmpFile.Name())

	cmd := exec.Command("pandoc", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pandoc failed: %w", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return nil, fmt.Errorf("stat output: %w", err)
	}

	return &Result{
		Path:   outPath,
		Size:   info.Size(),
		Format: req.Format,
	}, nil
}

// OutputPath computes the output file path by replacing the .md extension
// with the target format extension.
func OutputPath(source, format, outDir string) string {
	base := filepath.Base(source)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(outDir, stem+"."+format)
}

// FindMatchingCoverLetter looks for a cover letter file that matches a
// resume file's date and company slug pattern.
func FindMatchingCoverLetter(vaultRoot, resumePath string) string {
	base := filepath.Base(resumePath)
	// Resume naming: YYYY-MM-DD_company-slug_candidate_resume.md
	// Cover naming:  YYYY-MM-DD_company-slug_candidate_cover.md
	parts := strings.SplitN(base, "_", 3)
	if len(parts) < 2 {
		return ""
	}
	prefix := parts[0] + "_" + parts[1] // date_company-slug

	coverDir := filepath.Join(vaultRoot, "cover-letters")
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".md") {
			return filepath.Join(coverDir, e.Name())
		}
	}
	return ""
}
