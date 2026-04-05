// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Compensation is the normalized compensation data from any of the three
// formats found in vault job files.
type Compensation struct {
	Min      int
	Max      int
	Currency string
	Equity   bool
	Raw      string // original string representation, if from a string field
}

// compensationData is the structured YAML object form of compensation.
type compensationData struct {
	Min      int    `yaml:"min"`
	Max      int    `yaml:"max"`
	Currency string `yaml:"currency"`
	Equity   bool   `yaml:"equity"`
}

// Job represents a jobs/target/*.md vault file.
type Job struct {
	Title     string `yaml:"title"`
	Company   string `yaml:"company"`
	CompanySlug string `yaml:"company_slug,omitempty"`
	Location  string `yaml:"location"`
	Type      string `yaml:"type,omitempty"`
	Mode      string `yaml:"mode,omitempty"`
	Remote    *bool  `yaml:"remote,omitempty"`
	Seniority string `yaml:"seniority,omitempty"`
	Domain    string `yaml:"domain,omitempty"`
	JobID     string `yaml:"job_id,omitempty"`
	Status    string `yaml:"status,omitempty"`
	Parsed    string `yaml:"parsed,omitempty"`
	Posted    string `yaml:"posted,omitempty"`
	Deadline  string `yaml:"deadline,omitempty"`

	// Source URL — three different field names in practice.
	Source    string `yaml:"source,omitempty"`
	SourceURL string `yaml:"source_url,omitempty"`
	URL       string `yaml:"url,omitempty"`

	// Compensation — three different formats.
	CompensationObj   *compensationData `yaml:"compensation,omitempty"`
	Salary            string            `yaml:"salary,omitempty"`
	SalaryRange       string            `yaml:"salary_range,omitempty"`
	CompensationNotes string            `yaml:"compensation_notes,omitempty"`

	RequiredSkills  []string `yaml:"required_skills,omitempty"`
	PreferredSkills []string `yaml:"preferred_skills,omitempty"`
	CultureSignals  []string `yaml:"culture_signals,omitempty"`
	Tags            []string `yaml:"tags,omitempty"`

	// Normalized fields (populated post-unmarshal, not from YAML).
	Comp        *Compensation `yaml:"-"`
	ResolvedURL string        `yaml:"-"`
	FilePath    string        `yaml:"-"`
	Body        []byte        `yaml:"-"`
}

// salaryRegex matches patterns like "$150,000 - $200,000 USD" or "$152,000–$287,500".
// Handles hyphen (-), en-dash (–), and em-dash (—).
var salaryRegex = regexp.MustCompile(`\$?([\d,]+)\s*[-–—]\s*\$?([\d,]+)`)

// LoadJob loads a job file from the given path.
func LoadJob(path string) (*Job, error) {
	var job Job
	body, err := loadAndParse(path, &job)
	if err != nil {
		return nil, err
	}

	if job.Title == "" || job.Company == "" {
		return nil, fmt.Errorf("job %s: missing required field (title or company)", path)
	}

	job.FilePath = path
	job.Body = body
	job.normalize()
	return &job, nil
}

// LoadAllJobs loads all .md files from the given directory (should be jobs/target/).
func LoadAllJobs(dir string) ([]*Job, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read jobs dir %s: %w", dir, err)
	}

	var jobs []*Job
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		job, err := LoadJob(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// normalize populates the Comp and ResolvedURL fields from raw YAML fields.
func (j *Job) normalize() {
	j.normalizeCompensation()
	j.normalizeURL()
}

func (j *Job) normalizeCompensation() {
	switch {
	case j.CompensationObj != nil:
		j.Comp = &Compensation{
			Min:      j.CompensationObj.Min,
			Max:      j.CompensationObj.Max,
			Currency: j.CompensationObj.Currency,
			Equity:   j.CompensationObj.Equity,
		}
	case j.Salary != "":
		if c := parseSalaryString(j.Salary); c != nil {
			j.Comp = c
		}
	case j.SalaryRange != "":
		if c := parseSalaryString(j.SalaryRange); c != nil {
			j.Comp = c
		}
	}
}

func (j *Job) normalizeURL() {
	switch {
	case j.SourceURL != "":
		j.ResolvedURL = j.SourceURL
	case j.Source != "":
		j.ResolvedURL = j.Source
	case j.URL != "":
		j.ResolvedURL = j.URL
	}
}

// parseSalaryString extracts min/max from strings like "$200,000 - $280,000 USD".
func parseSalaryString(s string) *Compensation {
	matches := salaryRegex.FindStringSubmatch(s)
	if len(matches) < 3 {
		return nil
	}

	minVal := parseIntFromCommaStr(matches[1])
	maxVal := parseIntFromCommaStr(matches[2])

	currency := "USD"
	upper := strings.ToUpper(s)
	if strings.Contains(upper, "EUR") {
		currency = "EUR"
	} else if strings.Contains(upper, "GBP") {
		currency = "GBP"
	}

	return &Compensation{
		Min:      minVal,
		Max:      maxVal,
		Currency: currency,
		Raw:      s,
	}
}

func parseIntFromCommaStr(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	v, _ := strconv.Atoi(s)
	return v
}
