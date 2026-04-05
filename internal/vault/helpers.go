// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// loadAndParse reads a file, extracts YAML frontmatter, and unmarshals it
// into the target struct. Returns the markdown body and any error.
func loadAndParse(path string, target any) (body []byte, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	fm, body, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter in %s: %w", path, err)
	}

	if fm == nil {
		return body, nil
	}

	if err := yaml.Unmarshal(fm, target); err != nil {
		return nil, fmt.Errorf("unmarshal frontmatter in %s: %w", path, err)
	}

	return body, nil
}

// ExtractYear pulls the year from flexible date strings.
// Handles: "1984", "2024-08", "2026-04-02", "2026-04-03T14:30:00Z".
// Returns 0 and an error for "Present", empty strings, or unparseable input.
func ExtractYear(dateStr string) (int, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return 0, fmt.Errorf("empty date string")
	}
	if strings.EqualFold(dateStr, "present") {
		return 0, fmt.Errorf("date is %q (current)", dateStr)
	}

	// Take the first 4 characters as the year.
	if len(dateStr) < 4 {
		return 0, fmt.Errorf("date string too short: %q", dateStr)
	}

	year, err := strconv.Atoi(dateStr[:4])
	if err != nil {
		return 0, fmt.Errorf("cannot parse year from %q: %w", dateStr, err)
	}

	return year, nil
}
