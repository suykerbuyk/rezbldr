// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

import (
	"path/filepath"
	"testing"
)

func vaultPath(parts ...string) string {
	elems := append([]string{"testdata", "vault"}, parts...)
	return filepath.Join(elems...)
}

// --- Contact ---

func TestLoadContact(t *testing.T) {
	c, err := LoadContact(vaultPath("profile", "contact.md"))
	if err != nil {
		t.Fatalf("LoadContact: %v", err)
	}

	assertEqual(t, "Name", c.Name, "Test User")
	assertEqual(t, "Email", c.Email, "test@example.com")
	assertEqual(t, "Phone", c.Phone, "+1-555-555-1234")
	assertEqual(t, "Location", c.Location, "Denver, CO")
	assertEqual(t, "LinkedIn", c.LinkedIn, "https://www.linkedin.com/in/testuser/")
	assertEqual(t, "GitHub", c.GitHub, "https://github.com/testuser")
	assertEqual(t, "Tagline", c.Tagline, "Building great software")
	assertEqual(t, "InternationalTeams", c.InternationalTeams, "Germany, Japan")

	if len(c.Languages) != 2 {
		t.Errorf("Languages: got %d, want 2", len(c.Languages))
	}
}

func TestLoadContact_NotFound(t *testing.T) {
	_, err := LoadContact("/nonexistent/contact.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- Experience ---

func TestLoadExperience(t *testing.T) {
	exp, err := LoadExperience(vaultPath("experience", "2017-acme-solutions-architect.md"))
	if err != nil {
		t.Fatalf("LoadExperience: %v", err)
	}

	assertEqual(t, "Role", exp.Role, "Solutions Architect")
	assertEqual(t, "Company", exp.Company, "Acme Corp")
	assertEqual(t, "CompanySlug", exp.CompanySlug, "acme")
	assertEqual(t, "Start", exp.Start, "2017")
	assertEqual(t, "End", exp.End, "2023")
	assertEqual(t, "EmploymentType", exp.EmploymentType, "full-time")
	assertEqual(t, "Domain", exp.Domain, "storage")
	assertEqual(t, "Visibility", exp.Visibility, "resume")

	if !exp.Highlight {
		t.Error("expected Highlight to be true")
	}
	if exp.Current {
		t.Error("expected Current to be false")
	}
	if len(exp.Tags) != 4 {
		t.Errorf("Tags: got %d, want 4", len(exp.Tags))
	}
	if len(exp.Skills) != 4 {
		t.Errorf("Skills: got %d, want 4", len(exp.Skills))
	}
	if len(exp.Body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestLoadExperience_Military(t *testing.T) {
	exp, err := LoadExperience(vaultPath("experience", "1990-military-electronics-tech.md"))
	if err != nil {
		t.Fatalf("LoadExperience: %v", err)
	}

	assertEqual(t, "EmploymentType", exp.EmploymentType, "military")
	assertEqual(t, "Start", exp.Start, "1990")
	assertEqual(t, "Domain", exp.Domain, "defense")

	if exp.Highlight {
		t.Error("expected Highlight to be false")
	}
}

func TestLoadExperience_CurrentWithYearMonth(t *testing.T) {
	exp, err := LoadExperience(vaultPath("experience", "2024-startup-staff-engineer.md"))
	if err != nil {
		t.Fatalf("LoadExperience: %v", err)
	}

	assertEqual(t, "Start", exp.Start, "2024-08")
	assertEqual(t, "End", exp.End, "Present")

	if !exp.Current {
		t.Error("expected Current to be true")
	}
}

func TestLoadAllExperiences(t *testing.T) {
	exps, err := LoadAllExperiences(vaultPath("experience"))
	if err != nil {
		t.Fatalf("LoadAllExperiences: %v", err)
	}

	if len(exps) != 4 {
		t.Errorf("experience count: got %d, want 4", len(exps))
	}
}

// --- Job ---

func TestLoadJob_CompensationObject(t *testing.T) {
	job, err := LoadJob(vaultPath("jobs", "target", "2026-04-01-bigco-senior-storage-engineer.md"))
	if err != nil {
		t.Fatalf("LoadJob: %v", err)
	}

	assertEqual(t, "Title", job.Title, "Senior Storage Engineer")
	assertEqual(t, "Company", job.Company, "BigCo Industries")
	assertEqual(t, "CompanySlug", job.CompanySlug, "bigco")
	assertEqual(t, "Mode", job.Mode, "Hybrid")
	assertEqual(t, "Domain", job.Domain, "storage")
	assertEqual(t, "Status", job.Status, "targeting")
	assertEqual(t, "ResolvedURL", job.ResolvedURL, "https://example.com/jobs/12345")

	if job.Comp == nil {
		t.Fatal("expected non-nil Comp")
	}
	if job.Comp.Min != 150000 {
		t.Errorf("Comp.Min: got %d, want 150000", job.Comp.Min)
	}
	if job.Comp.Max != 200000 {
		t.Errorf("Comp.Max: got %d, want 200000", job.Comp.Max)
	}
	if !job.Comp.Equity {
		t.Error("expected Comp.Equity to be true")
	}

	if len(job.RequiredSkills) != 3 {
		t.Errorf("RequiredSkills: got %d, want 3", len(job.RequiredSkills))
	}
	if len(job.PreferredSkills) != 3 {
		t.Errorf("PreferredSkills: got %d, want 3", len(job.PreferredSkills))
	}
}

func TestLoadJob_SalaryString(t *testing.T) {
	job, err := LoadJob(vaultPath("jobs", "target", "2026-04-03-startup-staff-engineer.md"))
	if err != nil {
		t.Fatalf("LoadJob: %v", err)
	}

	assertEqual(t, "ResolvedURL", job.ResolvedURL, "https://startupco.com/careers/staff-storage")

	if job.Remote == nil || !*job.Remote {
		t.Error("expected Remote to be true")
	}

	if job.Comp == nil {
		t.Fatal("expected non-nil Comp")
	}
	if job.Comp.Min != 200000 {
		t.Errorf("Comp.Min: got %d, want 200000", job.Comp.Min)
	}
	if job.Comp.Max != 280000 {
		t.Errorf("Comp.Max: got %d, want 280000", job.Comp.Max)
	}
	assertEqual(t, "Comp.Raw", job.Comp.Raw, "$200,000 - $280,000 USD")
}

func TestLoadJob_SalaryRange(t *testing.T) {
	job, err := LoadJob(vaultPath("jobs", "target", "2026-04-02-megacorp-systems-architect.md"))
	if err != nil {
		t.Fatalf("LoadJob: %v", err)
	}

	assertEqual(t, "JobID", job.JobID, "JR99999")
	assertEqual(t, "ResolvedURL", job.ResolvedURL, "https://megacorp.com/jobs/JR99999")

	if job.Comp == nil {
		t.Fatal("expected non-nil Comp")
	}
	if job.Comp.Min != 160000 {
		t.Errorf("Comp.Min: got %d, want 160000", job.Comp.Min)
	}
	if job.Comp.Max != 220000 {
		t.Errorf("Comp.Max: got %d, want 220000", job.Comp.Max)
	}

	// PreferredSkills is explicitly empty [].
	if job.PreferredSkills == nil {
		t.Error("expected non-nil PreferredSkills (explicit empty array)")
	}
	if len(job.PreferredSkills) != 0 {
		t.Errorf("PreferredSkills: got %d, want 0", len(job.PreferredSkills))
	}
}

func TestLoadJob_Minimal(t *testing.T) {
	job, err := LoadJob(vaultPath("jobs", "target", "2026-03-28-smallco-storage-admin.md"))
	if err != nil {
		t.Fatalf("LoadJob: %v", err)
	}

	assertEqual(t, "Title", job.Title, "Storage Administrator")
	assertEqual(t, "Company", job.Company, "SmallCo")

	if job.Comp != nil {
		t.Errorf("expected nil Comp for minimal job, got %+v", job.Comp)
	}
	if job.ResolvedURL != "" {
		t.Errorf("expected empty ResolvedURL, got %q", job.ResolvedURL)
	}
	if job.RequiredSkills != nil {
		t.Errorf("expected nil RequiredSkills, got %v", job.RequiredSkills)
	}
}

func TestLoadAllJobs(t *testing.T) {
	jobs, err := LoadAllJobs(vaultPath("jobs", "target"))
	if err != nil {
		t.Fatalf("LoadAllJobs: %v", err)
	}

	if len(jobs) != 4 {
		t.Errorf("job count: got %d, want 4", len(jobs))
	}
}

// --- Skills ---

func TestLoadSkills(t *testing.T) {
	skills, err := LoadSkills(vaultPath("profile", "skills.md"))
	if err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}

	if len(skills) != 10 {
		t.Fatalf("skills count: got %d, want 10", len(skills))
	}

	// Spot-check first entry.
	first := skills[0]
	assertEqual(t, "first.Skill", first.Skill, "Go")
	assertEqual(t, "first.Proficiency", first.Proficiency, "Advanced")
	assertEqual(t, "first.LastUsed", first.LastUsed, "2026")
	assertEqual(t, "first.Years", first.Years, "5")
	assertEqual(t, "first.Category", first.Category, "Languages")

	// Spot-check last entry.
	last := skills[9]
	assertEqual(t, "last.Skill", last.Skill, "Terraform")
	assertEqual(t, "last.Proficiency", last.Proficiency, "Novice")

	// Check a "20+" years entry.
	c := skills[2] // C
	assertEqual(t, "C.Years", c.Years, "20+")
}

// --- Resume ---

func TestLoadResume(t *testing.T) {
	r, err := LoadResume(vaultPath("resumes", "generated", "test_user_2026-04-01-bigco_resume.md"))
	if err != nil {
		t.Fatalf("LoadResume: %v", err)
	}

	assertEqual(t, "JobFile", r.JobFile, "jobs/target/2026-04-01-bigco-senior-storage-engineer.md")
	assertEqual(t, "Model", r.Model, "claude-opus-4-6")
	assertEqual(t, "Status", r.Status, "draft")

	if r.WordCount != 650 {
		t.Errorf("WordCount: got %d, want 650", r.WordCount)
	}
	if r.Version != 1 {
		t.Errorf("Version: got %d, want 1", r.Version)
	}
	if len(r.ExperienceFiles) != 2 {
		t.Errorf("ExperienceFiles: got %d, want 2", len(r.ExperienceFiles))
	}
	if len(r.Body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestLoadAllResumes(t *testing.T) {
	resumes, err := LoadAllResumes(vaultPath("resumes", "generated"))
	if err != nil {
		t.Fatalf("LoadAllResumes: %v", err)
	}
	if len(resumes) != 1 {
		t.Errorf("resume count: got %d, want 1", len(resumes))
	}
}

func TestLoadAllResumes_NonexistentDir(t *testing.T) {
	resumes, err := LoadAllResumes("/nonexistent/dir")
	if err != nil {
		t.Fatalf("expected nil error for nonexistent dir, got %v", err)
	}
	if resumes != nil {
		t.Errorf("expected nil resumes, got %d", len(resumes))
	}
}

// --- Cover Letter ---

func TestLoadCoverLetter(t *testing.T) {
	cl, err := LoadCoverLetter(vaultPath("cover-letters", "test_user_2026-04-01-bigco_cover.md"))
	if err != nil {
		t.Fatalf("LoadCoverLetter: %v", err)
	}

	assertEqual(t, "JobFile", cl.JobFile, "jobs/target/2026-04-01-bigco-senior-storage-engineer.md")
	assertEqual(t, "ResumeFile", cl.ResumeFile, "resumes/generated/test_user_2026-04-01-bigco_resume.md")
	assertEqual(t, "Model", cl.Model, "claude-opus-4-6")
	assertEqual(t, "Status", cl.Status, "draft")

	if len(cl.Body) == 0 {
		t.Error("expected non-empty body")
	}
}

// --- Training ---

func TestLoadTraining(t *testing.T) {
	tr, err := LoadTraining(vaultPath("training", "kubernetes.md"))
	if err != nil {
		t.Fatalf("LoadTraining: %v", err)
	}

	assertEqual(t, "Skill", tr.Skill, "Kubernetes")
	assertEqual(t, "Category", tr.Category, "DevOps")
	assertEqual(t, "Priority", tr.Priority, "high")
	assertEqual(t, "Status", tr.Status, "not-started")

	if len(tr.SurfacedBy) != 2 {
		t.Fatalf("SurfacedBy: got %d, want 2", len(tr.SurfacedBy))
	}
	assertEqual(t, "SurfacedBy[0].Company", tr.SurfacedBy[0].Company, "StartupCo")
	assertEqual(t, "SurfacedBy[0].Requirement", tr.SurfacedBy[0].Requirement, "required")
	assertEqual(t, "SurfacedBy[1].Requirement", tr.SurfacedBy[1].Requirement, "preferred")

	if len(tr.RelatedSkills) != 3 {
		t.Errorf("RelatedSkills: got %d, want 3", len(tr.RelatedSkills))
	}

	if len(tr.Body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestLoadAllTraining(t *testing.T) {
	training, err := LoadAllTraining(vaultPath("training"))
	if err != nil {
		t.Fatalf("LoadAllTraining: %v", err)
	}
	if len(training) != 1 {
		t.Errorf("training count: got %d, want 1", len(training))
	}
}

// --- Salary Parsing ---

func TestParseSalaryString(t *testing.T) {
	tests := []struct {
		input   string
		wantMin int
		wantMax int
	}{
		{"$200,000 - $280,000 USD", 200000, 280000},
		{"$152,000–$287,500", 152000, 287500},       // en-dash
		{"$160,000—$220,000", 160000, 220000},       // em-dash
		{"150000-200000", 150000, 200000},            // no $ or commas
		{"$100,000 - $150,000 EUR", 100000, 150000},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c := parseSalaryString(tt.input)
			if c == nil {
				t.Fatal("expected non-nil compensation")
			}
			if c.Min != tt.wantMin {
				t.Errorf("Min: got %d, want %d", c.Min, tt.wantMin)
			}
			if c.Max != tt.wantMax {
				t.Errorf("Max: got %d, want %d", c.Max, tt.wantMax)
			}
		})
	}
}

func TestParseSalaryString_Invalid(t *testing.T) {
	c := parseSalaryString("competitive")
	if c != nil {
		t.Errorf("expected nil for unparseable salary, got %+v", c)
	}
}

// --- test helpers ---

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", field, got, want)
	}
}
