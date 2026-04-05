package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Vault is the root object providing access to all vault data.
type Vault struct {
	Root         string
	Contact      *Contact
	Skills       []SkillEntry
	Experiences  []*Experience
	Jobs         []*Job
	Resumes      []*Resume
	CoverLetters []*CoverLetter
	Training     []*Training
}

// Open loads the vault from a root directory path.
// It validates that the root contains profile/contact.md, then eagerly loads
// all vault data into memory.
func Open(root string) (*Vault, error) {
	// Validate vault root by checking for the contact file.
	contactPath := filepath.Join(root, "profile", "contact.md")
	if _, err := os.Stat(contactPath); err != nil {
		return nil, fmt.Errorf("not a vault (missing profile/contact.md): %w", err)
	}

	v := &Vault{Root: root}

	var err error

	v.Contact, err = LoadContact(contactPath)
	if err != nil {
		return nil, fmt.Errorf("load contact: %w", err)
	}

	skillsPath := filepath.Join(root, "profile", "skills.md")
	if _, statErr := os.Stat(skillsPath); statErr == nil {
		v.Skills, err = LoadSkills(skillsPath)
		if err != nil {
			return nil, fmt.Errorf("load skills: %w", err)
		}
	}

	expDir := filepath.Join(root, "experience")
	if _, statErr := os.Stat(expDir); statErr == nil {
		v.Experiences, err = LoadAllExperiences(expDir)
		if err != nil {
			return nil, fmt.Errorf("load experiences: %w", err)
		}
	}

	jobsDir := filepath.Join(root, "jobs", "target")
	if _, statErr := os.Stat(jobsDir); statErr == nil {
		v.Jobs, err = LoadAllJobs(jobsDir)
		if err != nil {
			return nil, fmt.Errorf("load jobs: %w", err)
		}
	}

	resumesDir := filepath.Join(root, "resumes", "generated")
	v.Resumes, err = LoadAllResumes(resumesDir)
	if err != nil {
		return nil, fmt.Errorf("load resumes: %w", err)
	}

	coverDir := filepath.Join(root, "cover-letters")
	if _, statErr := os.Stat(coverDir); statErr == nil {
		entries, readErr := os.ReadDir(coverDir)
		if readErr == nil {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}
				cl, loadErr := LoadCoverLetter(filepath.Join(coverDir, entry.Name()))
				if loadErr != nil {
					return nil, fmt.Errorf("load cover letter %s: %w", entry.Name(), loadErr)
				}
				v.CoverLetters = append(v.CoverLetters, cl)
			}
		}
	}

	trainingDir := filepath.Join(root, "training")
	v.Training, err = LoadAllTraining(trainingDir)
	if err != nil {
		return nil, fmt.Errorf("load training: %w", err)
	}

	return v, nil
}

// LoadJob loads a specific job file. The path can be absolute or relative to the vault root.
func (v *Vault) LoadJob(path string) (*Job, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(v.Root, path)
	}
	return LoadJob(path)
}

// LatestJob returns the most recent job file by filename (date-prefixed filenames
// sort lexicographically in chronological order).
func (v *Vault) LatestJob() (*Job, error) {
	if len(v.Jobs) == 0 {
		return nil, fmt.Errorf("no job files loaded")
	}

	sorted := make([]*Job, len(v.Jobs))
	copy(sorted, v.Jobs)
	sort.Slice(sorted, func(i, j int) bool {
		return filepath.Base(sorted[i].FilePath) > filepath.Base(sorted[j].FilePath)
	})

	return sorted[0], nil
}

// LatestResume returns the most recent generated resume by filename.
func (v *Vault) LatestResume() (*Resume, error) {
	if len(v.Resumes) == 0 {
		return nil, fmt.Errorf("no resume files loaded")
	}

	sorted := make([]*Resume, len(v.Resumes))
	copy(sorted, v.Resumes)
	sort.Slice(sorted, func(i, j int) bool {
		return filepath.Base(sorted[i].FilePath) > filepath.Base(sorted[j].FilePath)
	})

	return sorted[0], nil
}
