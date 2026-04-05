// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package scoring

import (
	"testing"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

func TestScore_BigCoJob(t *testing.T) {
	job := bigcoJob()
	exps := testExperiences()
	ranked := RankAt(job, exps, 2026)

	ms := Score(job, exps, ranked)

	// Required: Ceph, Linux, ZFS — we match ceph and zfs but not linux.
	if len(ms.RequiredHits) != 2 {
		t.Errorf("expected 2 required hits, got %d: %v", len(ms.RequiredHits), ms.RequiredHits)
	}
	if len(ms.RequiredMisses) != 1 {
		t.Errorf("expected 1 required miss, got %d: %v", len(ms.RequiredMisses), ms.RequiredMisses)
	}
	assertFloat(t, "required coverage", 2.0/3.0, ms.RequiredCoverage, 0.01)

	// Preferred: Go, Python, Kubernetes — all matched across experiences.
	if len(ms.PreferredHits) != 3 {
		t.Errorf("expected 3 preferred hits, got %d: %v", len(ms.PreferredHits), ms.PreferredHits)
	}
	assertFloat(t, "preferred coverage", 1.0, ms.PreferredCoverage, 0.01)

	// Domain: job=storage, acme=storage → exact match.
	assertFloat(t, "domain match", 1.0, ms.DomainMatch, 0.01)

	// Seniority: job=Senior, best scoring exp role has "Staff" (cloudstart) → 1 level off → 0.5.
	// Actually "Solutions Architect" maps to "staff" level too via "architect" keyword.
	// So best is staff(4) vs senior(3) → diff=1 → 0.5.
	assertFloat(t, "seniority match", 0.5, ms.SeniorityMatch, 0.01)

	// Overall: req(0.667*0.60) + pref(1.0*0.20) + domain(1.0*0.10) + seniority(0.5*0.10)
	// = 0.400 + 0.200 + 0.100 + 0.050 = 0.750 → 75
	if ms.Overall != 75 {
		t.Errorf("expected overall 75, got %d", ms.Overall)
	}
}

func TestScore_StartupJob(t *testing.T) {
	job := startupJob()
	exps := testExperiences()
	ranked := RankAt(job, exps, 2026)

	ms := Score(job, exps, ranked)

	// Required: Ceph, Kubernetes, Go — all matched by CloudStart.
	assertFloat(t, "required coverage", 1.0, ms.RequiredCoverage, 0.01)

	// Preferred: AWS, Terraform — AWS matched by CloudStart, Terraform not.
	assertFloat(t, "preferred coverage", 0.5, ms.PreferredCoverage, 0.01)

	// Domain: job has no domain set in startupJob() helper... let me check.
	// startupJob sets Domain: "storage". CloudStart has "cloud/storage" → partial 0.5,
	// Acme has "storage" → exact 1.0.
	assertFloat(t, "domain match", 1.0, ms.DomainMatch, 0.01)
}

func TestScore_EmptyJob(t *testing.T) {
	job := &vault.Job{Title: "Empty", Company: "Co"}
	ranked := RankAt(job, testExperiences(), 2026)
	ms := Score(job, testExperiences(), ranked)

	if ms.Overall != 0 {
		t.Errorf("expected 0 overall for empty job, got %d", ms.Overall)
	}
	if ms.RequiredCoverage != 0 {
		t.Errorf("expected 0 required coverage, got %f", ms.RequiredCoverage)
	}
}

func TestScore_NoRankedResults(t *testing.T) {
	job := bigcoJob()
	ms := Score(job, nil, nil)

	if ms.RequiredCoverage != 0 {
		t.Errorf("expected 0 required coverage with no results, got %f", ms.RequiredCoverage)
	}
	if ms.Overall != 0 {
		t.Errorf("expected 0 overall with no results, got %d", ms.Overall)
	}
}

func TestDomainMatch_Partial(t *testing.T) {
	exps := []*vault.Experience{
		{
			Role: "Engineer", Company: "CloudCo",
			Domain: "cloud/storage", Visibility: "resume",
			FilePath: "/vault/experience/cloud.md",
		},
	}
	ranked := []ScoredExperience{
		{File: "cloud.md", Score: 1.0},
	}

	dm := computeDomainMatch("storage", exps, ranked)
	if dm != 0.5 {
		t.Errorf("expected partial domain match 0.5, got %f", dm)
	}
}

func TestDomainMatch_Empty(t *testing.T) {
	dm := computeDomainMatch("", nil, nil)
	if dm != 0 {
		t.Errorf("expected 0 for empty job domain, got %f", dm)
	}
}

func TestSeniorityMatch_ExactAndOff(t *testing.T) {
	tests := []struct {
		name     string
		jobSen   string
		role     string
		expected float64
	}{
		{"exact senior", "Senior", "Senior Engineer", 1.0},
		{"staff vs senior", "Senior", "Staff Engineer", 0.5},
		{"junior vs senior", "Junior", "Senior Engineer", 0.0},
		{"staff vs staff", "Staff", "Staff Engineer", 1.0},
		{"mid-senior level", "Mid-Senior level", "Senior Engineer", 0.5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exps := []*vault.Experience{
				{Role: tc.role, Company: "Co", Visibility: "resume", FilePath: "/exp.md"},
			}
			ranked := []ScoredExperience{{File: "exp.md", Score: 1.0}}
			result := computeSeniorityMatch(tc.jobSen, exps, ranked)
			assertFloat(t, tc.name, tc.expected, result, 0.01)
		})
	}
}

func TestSeniorityMatch_EmptyJobSeniority(t *testing.T) {
	result := computeSeniorityMatch("", nil, nil)
	if result != 0 {
		t.Errorf("expected 0 for empty job seniority, got %f", result)
	}
}
