// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// SkillEntry represents one row of the skills inventory table.
type SkillEntry struct {
	Skill       string
	Proficiency string
	LastUsed    string
	Years       string
	Category    string
}

// ParseSkillsTable parses the markdown skills table from content (body without frontmatter).
// Expects a pipe-delimited table with columns: Skill, Proficiency, Last Used, Years, Category.
func ParseSkillsTable(content []byte) ([]SkillEntry, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Find the header row.
	headerFound := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "| Skill") && strings.Contains(line, "Proficiency") {
			headerFound = true
			break
		}
	}

	if !headerFound {
		return nil, fmt.Errorf("skills table header not found")
	}

	// Skip the separator row (|---|---|...).
	if !scanner.Scan() {
		return nil, fmt.Errorf("expected separator row after header")
	}

	// Parse data rows until we hit an empty line or a heading.
	var entries []SkillEntry
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Stop at headings or empty lines.
		if line == "" || strings.HasPrefix(line, "#") {
			break
		}

		// Skip lines that don't look like table rows.
		if !strings.Contains(line, "|") {
			continue
		}

		entry, err := parseSkillRow(line)
		if err != nil {
			continue // Skip malformed rows.
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan skills table: %w", err)
	}

	return entries, nil
}

// LoadSkills loads the skills inventory from a markdown file.
// The file has YAML frontmatter (just an "updated" field) followed by the skills table.
func LoadSkills(path string) ([]SkillEntry, error) {
	var meta struct {
		Updated string `yaml:"updated"`
	}
	body, err := loadAndParse(path, &meta)
	if err != nil {
		return nil, err
	}

	return ParseSkillsTable(body)
}

func parseSkillRow(line string) (SkillEntry, error) {
	// Split on | and trim each cell.
	parts := strings.Split(line, "|")

	// A properly formatted row has empty strings at the start and end
	// due to leading/trailing |, plus 5 data cells.
	var cells []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			cells = append(cells, trimmed)
		}
	}

	if len(cells) < 5 {
		return SkillEntry{}, fmt.Errorf("expected 5 cells, got %d", len(cells))
	}

	return SkillEntry{
		Skill:       cells[0],
		Proficiency: cells[1],
		LastUsed:    cells[2],
		Years:       cells[3],
		Category:    cells[4],
	}, nil
}
