// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

// CoverLetter represents a cover-letters/*.md vault file.
type CoverLetter struct {
	JobFile    string `yaml:"job_file"`
	ResumeFile string `yaml:"resume_file"`
	Generated  string `yaml:"generated"`
	Model      string `yaml:"model"`
	Status     string `yaml:"status"`

	FilePath string `yaml:"-"`
	Body     []byte `yaml:"-"`
}

// LoadCoverLetter loads a cover letter from the given path.
func LoadCoverLetter(path string) (*CoverLetter, error) {
	var cl CoverLetter
	body, err := loadAndParse(path, &cl)
	if err != nil {
		return nil, err
	}
	cl.FilePath = path
	cl.Body = body
	return &cl, nil
}
