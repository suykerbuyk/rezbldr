package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resume represents a resumes/generated/*.md vault file.
type Resume struct {
	JobFile         string   `yaml:"job_file"`
	Generated       string   `yaml:"generated"`
	Model           string   `yaml:"model"`
	Status          string   `yaml:"status"`
	ExperienceFiles []string `yaml:"experience_files"`
	WordCount       int      `yaml:"word_count"`
	Version         int      `yaml:"version"`

	FilePath string `yaml:"-"`
	Body     []byte `yaml:"-"`
}

// LoadResume loads a generated resume from the given path.
func LoadResume(path string) (*Resume, error) {
	var r Resume
	body, err := loadAndParse(path, &r)
	if err != nil {
		return nil, err
	}
	r.FilePath = path
	r.Body = body
	return &r, nil
}

// LoadAllResumes loads all .md files from the given directory.
func LoadAllResumes(dir string) ([]*Resume, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read resumes dir %s: %w", dir, err)
	}

	var resumes []*Resume
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		r, err := LoadResume(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		resumes = append(resumes, r)
	}

	return resumes, nil
}
