// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package scoring

import (
	"testing"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// testExperiences returns the standard set of test experiences matching
// the testdata/vault/experience/ fixtures.
func testExperiences() []*vault.Experience {
	return []*vault.Experience{
		{
			Role: "Solutions Architect", Company: "Acme Corp",
			CompanySlug: "acme", Start: "2017", End: "2023",
			Tags: []string{"storage", "architecture", "ceph", "zfs"},
			Skills: []string{"Ceph", "ZFS", "Python", "Go"},
			Domain: "storage", Highlight: true, Visibility: "resume",
			FilePath: "/vault/experience/2017-acme-solutions-architect.md",
		},
		{
			Role: "Staff Software Engineer", Company: "CloudStart Inc",
			CompanySlug: "cloudstart", Start: "2024-08", End: "Present",
			Current: true,
			Tags:   []string{"storage", "kubernetes", "cloud"},
			Skills: []string{"Kubernetes", "Go", "Ceph", "AWS"},
			Domain: "cloud/storage", Highlight: true, Visibility: "resume",
			FilePath: "/vault/experience/2024-startup-staff-engineer.md",
		},
		{
			Role: "Software Engineer", Company: "Hidden Corp",
			CompanySlug: "hidden", Start: "2010", End: "2012",
			Tags: []string{"backend"}, Skills: []string{"Python"},
			Domain: "web", Highlight: false, Visibility: "hidden",
			FilePath: "/vault/experience/2010-hidden-corp-engineer.md",
		},
		{
			Role: "Electronics Technician", Company: "US Air Force",
			CompanySlug: "usaf", Start: "1990", End: "1994",
			Tags: []string{"electronics", "embedded"}, Skills: []string{"C"},
			Domain: "defense", Highlight: false, Visibility: "resume",
			FilePath: "/vault/experience/1990-military-electronics-tech.md",
		},
	}
}

func bigcoJob() *vault.Job {
	return &vault.Job{
		Title: "Senior Storage Engineer", Company: "BigCo Industries",
		Seniority: "Senior", Domain: "storage",
		RequiredSkills:  []string{"Ceph", "Linux", "ZFS"},
		PreferredSkills: []string{"Go", "Python", "Kubernetes"},
		Tags:            []string{"storage", "ceph", "zfs", "linux"},
		FilePath:        "/vault/jobs/target/2026-04-01-bigco-senior-storage-engineer.md",
	}
}

func startupJob() *vault.Job {
	return &vault.Job{
		Title: "Staff Software Engineer, Storage", Company: "StartupCo",
		Seniority: "Staff", Domain: "storage",
		RequiredSkills:  []string{"Ceph", "Kubernetes", "Go"},
		PreferredSkills: []string{"AWS", "Terraform"},
		Tags:            []string{"storage", "cloud", "kubernetes"},
		FilePath:        "/vault/jobs/target/2026-04-03-startup-staff-engineer.md",
	}
}

func TestRank_BigCoJob(t *testing.T) {
	results := RankAt(bigcoJob(), testExperiences(), 2026)

	// Hidden experience should be excluded.
	for _, r := range results {
		if r.Company == "Hidden Corp" {
			t.Fatal("hidden experience should be excluded from results")
		}
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results (excluding hidden), got %d", len(results))
	}

	// Acme should rank first (most matches + highlight boost).
	if results[0].Company != "Acme Corp" {
		t.Errorf("expected Acme Corp first, got %s", results[0].Company)
	}

	// CloudStart should rank second.
	if results[1].Company != "CloudStart Inc" {
		t.Errorf("expected CloudStart Inc second, got %s", results[1].Company)
	}

	// Military should rank last (no matches).
	if results[2].Company != "US Air Force" {
		t.Errorf("expected US Air Force last, got %s", results[2].Company)
	}
}

func TestRank_BigCoScores(t *testing.T) {
	results := RankAt(bigcoJob(), testExperiences(), 2026)

	// Acme: required{ceph, zfs}=4.0, preferred{go, python}=2.0, tags{storage}=0.5
	// Raw: 6.5, highlight boost: 6.5*1.10 = 7.15
	acme := findResult(results, "Acme Corp")
	if acme == nil {
		t.Fatal("Acme Corp not found")
	}
	assertFloat(t, "Acme score", 7.15, acme.Score, 0.01)
	if !acme.Boosted {
		t.Error("Acme should be boosted (highlight)")
	}
	if acme.Penalized {
		t.Error("Acme should not be penalized (end=2023, <10yr)")
	}
	assertStringSlice(t, "Acme required", []string{"ceph", "zfs"}, acme.MatchedRequired)
	assertStringSlice(t, "Acme preferred", []string{"go", "python"}, acme.MatchedPreferred)
	assertStringSlice(t, "Acme tags", []string{"storage"}, acme.MatchedTags)

	// CloudStart: required{ceph}=2.0, preferred{go, kubernetes}=2.0, tags{storage}=0.5
	// Raw: 4.5, highlight boost: 4.5*1.10 = 4.95
	cloud := findResult(results, "CloudStart Inc")
	if cloud == nil {
		t.Fatal("CloudStart Inc not found")
	}
	assertFloat(t, "CloudStart score", 4.95, cloud.Score, 0.01)
	if !cloud.Boosted {
		t.Error("CloudStart should be boosted")
	}

	// Military: no matches, score 0.
	mil := findResult(results, "US Air Force")
	if mil == nil {
		t.Fatal("US Air Force not found")
	}
	if mil.Score != 0 {
		t.Errorf("military score should be 0, got %f", mil.Score)
	}
	if mil.Boosted || mil.Penalized {
		t.Error("military should not be boosted or penalized (score=0)")
	}
}

func TestRank_Normalization(t *testing.T) {
	results := RankAt(bigcoJob(), testExperiences(), 2026)

	acme := findResult(results, "Acme Corp")
	if acme.NormalizedScore != 100 {
		t.Errorf("top scorer should normalize to 100, got %d", acme.NormalizedScore)
	}

	cloud := findResult(results, "CloudStart Inc")
	// 4.95/7.15 * 100 ≈ 69
	if cloud.NormalizedScore < 68 || cloud.NormalizedScore > 70 {
		t.Errorf("CloudStart normalized score expected ~69, got %d", cloud.NormalizedScore)
	}

	mil := findResult(results, "US Air Force")
	if mil.NormalizedScore != 0 {
		t.Errorf("zero-score should normalize to 0, got %d", mil.NormalizedScore)
	}
}

func TestRank_StartupJob(t *testing.T) {
	results := RankAt(startupJob(), testExperiences(), 2026)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// CloudStart should rank first for startup job (all 3 required skills).
	if results[0].Company != "CloudStart Inc" {
		t.Errorf("expected CloudStart first, got %s", results[0].Company)
	}
}

func TestRank_AgePenalty(t *testing.T) {
	old := &vault.Experience{
		Role: "Old Role", Company: "OldCo",
		Start: "2005", End: "2010",
		Tags: []string{"storage"}, Skills: []string{"Ceph"},
		Highlight: false, Visibility: "resume",
		FilePath: "/vault/experience/old.md",
	}

	job := &vault.Job{
		Title:          "Test",
		Company:        "TestCo",
		RequiredSkills: []string{"Ceph"},
		Tags:           []string{"storage"},
	}

	results := RankAt(job, []*vault.Experience{old}, 2026)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if !r.Penalized {
		t.Error("experience ending in 2010 should be penalized in 2026 (>10yr)")
	}

	// Raw: required{ceph}=2.0, tags{storage}=0.5 => 2.5 * 0.70 = 1.75
	assertFloat(t, "penalized score", 1.75, r.Score, 0.01)
}

func TestRank_EmptyJob(t *testing.T) {
	job := &vault.Job{Title: "Empty", Company: "Co"}
	results := RankAt(job, testExperiences(), 2026)

	for _, r := range results {
		if r.Score != 0 {
			t.Errorf("all scores should be 0 with empty job skills, got %f for %s", r.Score, r.Company)
		}
	}
}

func TestRank_NoExperiences(t *testing.T) {
	results := RankAt(bigcoJob(), nil, 2026)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil experiences, got %d", len(results))
	}
}

func TestRank_AllHidden(t *testing.T) {
	hidden := []*vault.Experience{
		{Role: "A", Company: "X", Visibility: "hidden", FilePath: "/a.md"},
		{Role: "B", Company: "Y", Visibility: "Hidden", FilePath: "/b.md"},
	}
	results := RankAt(bigcoJob(), hidden, 2026)
	if len(results) != 0 {
		t.Errorf("expected 0 results when all hidden, got %d", len(results))
	}
}

func TestRank_NoDuplicateCounting(t *testing.T) {
	// "ceph" appears in both required_skills and tags for bigco job.
	// An experience matching "ceph" should only count it as required (higher weight).
	exp := &vault.Experience{
		Role: "Ceph Admin", Company: "CephCo",
		Tags: []string{"ceph"}, Skills: []string{},
		Visibility: "resume", FilePath: "/vault/experience/ceph.md",
	}

	results := RankAt(bigcoJob(), []*vault.Experience{exp}, 2026)
	r := results[0]

	if len(r.MatchedRequired) != 1 || r.MatchedRequired[0] != "ceph" {
		t.Errorf("ceph should match as required, got required=%v", r.MatchedRequired)
	}
	if len(r.MatchedTags) != 0 {
		t.Errorf("ceph should not also match as tag, got tags=%v", r.MatchedTags)
	}
}

// --- Helpers ---

func findResult(results []ScoredExperience, company string) *ScoredExperience {
	for i := range results {
		if results[i].Company == company {
			return &results[i]
		}
	}
	return nil
}

func assertFloat(t *testing.T, name string, expected, actual, tolerance float64) {
	t.Helper()
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("%s: expected %f, got %f (tolerance %f)", name, expected, actual, tolerance)
	}
}

func assertStringSlice(t *testing.T, name string, expected, actual []string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("%s: expected %v, got %v", name, expected, actual)
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("%s: expected %v, got %v", name, expected, actual)
			return
		}
	}
}
