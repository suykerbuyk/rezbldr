// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Experience represents an experience/*.md vault file.
type Experience struct {
	Role           string   `yaml:"role"`
	Company        string   `yaml:"company"`
	CompanySlug    string   `yaml:"company_slug"`
	Start          string   `yaml:"start"`
	End            string   `yaml:"end"`
	Current        bool     `yaml:"current"`
	Location       string   `yaml:"location"`
	EmploymentType string   `yaml:"employment_type"`
	Tags           []string `yaml:"tags"`
	Skills         []string `yaml:"skills"`
	Domain         string   `yaml:"domain"`
	Highlight      bool     `yaml:"highlight"`
	Visibility     string   `yaml:"visibility"`
	Created        string   `yaml:"created"`
	Updated        string   `yaml:"updated"`

	// Populated by loader, not from YAML.
	FilePath string `yaml:"-"`
	Body     []byte `yaml:"-"`
}

// LoadExperience loads an experience file from the given path.
func LoadExperience(path string) (*Experience, error) {
	var exp Experience
	body, err := loadAndParse(path, &exp)
	if err != nil {
		return nil, err
	}

	if exp.Role == "" || exp.Company == "" {
		return nil, fmt.Errorf("experience %s: missing required field (role or company)", path)
	}

	exp.FilePath = path
	exp.Body = body
	return &exp, nil
}

// LoadAllExperiences loads all .md files from the given directory.
func LoadAllExperiences(dir string) ([]*Experience, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read experience dir %s: %w", dir, err)
	}

	var experiences []*Experience
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		exp, err := LoadExperience(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		experiences = append(experiences, exp)
	}

	return experiences, nil
}
