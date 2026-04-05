package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SurfacedBy records which job posting surfaced a training need.
type SurfacedBy struct {
	Job         string `yaml:"job"`
	Company     string `yaml:"company"`
	Requirement string `yaml:"requirement"`
	Date        string `yaml:"date"`
}

// Training represents a training/*.md vault file.
type Training struct {
	Skill         string       `yaml:"skill"`
	Category      string       `yaml:"category"`
	Priority      string       `yaml:"priority"`
	Status        string       `yaml:"status"`
	SurfacedBy    []SurfacedBy `yaml:"surfaced_by"`
	RelatedSkills []string     `yaml:"related_skills"`
	Created       string       `yaml:"created"`
	Updated       string       `yaml:"updated"`

	FilePath string `yaml:"-"`
	Body     []byte `yaml:"-"`
}

// LoadTraining loads a training file from the given path.
func LoadTraining(path string) (*Training, error) {
	var tr Training
	body, err := loadAndParse(path, &tr)
	if err != nil {
		return nil, err
	}
	tr.FilePath = path
	tr.Body = body
	return &tr, nil
}

// LoadAllTraining loads all .md files from the given directory.
func LoadAllTraining(dir string) ([]*Training, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read training dir %s: %w", dir, err)
	}

	var training []*Training
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		tr, err := LoadTraining(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		training = append(training, tr)
	}

	return training, nil
}
