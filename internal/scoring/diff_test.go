// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package scoring

import (
	"testing"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

func TestDiff_NoChange(t *testing.T) {
	job := bigcoJob()
	exps := testExperiences()
	ranked := RankAt(job, exps, 2026)
	ms := Score(job, exps, ranked)

	// Diff against same data — no change.
	d := Diff(job, exps, ranked, ms)

	if d.Delta != 0 {
		t.Errorf("expected 0 delta for unchanged data, got %d", d.Delta)
	}
	if d.OldScore != d.NewScore {
		t.Errorf("old=%d new=%d should be equal", d.OldScore, d.NewScore)
	}
	if len(d.ImprovedSkills) != 0 {
		t.Errorf("expected no improved skills, got %v", d.ImprovedSkills)
	}
}

func TestDiff_ImprovedAfterTagAddition(t *testing.T) {
	job := bigcoJob() // required: Ceph, Linux, ZFS

	// Initial: only ceph and zfs matched (no linux).
	initialExps := testExperiences()
	initialRanked := RankAt(job, initialExps, 2026)
	initialMatch := Score(job, initialExps, initialRanked)

	// Verify linux is initially a miss.
	found := false
	for _, s := range initialMatch.RequiredMisses {
		if s == "linux" {
			found = true
		}
	}
	if !found {
		t.Fatal("linux should be a required miss initially")
	}

	// Enriched: add "linux" tag to the CloudStart experience.
	enrichedExps := make([]*vault.Experience, len(initialExps))
	for i, exp := range initialExps {
		cp := *exp
		enrichedExps[i] = &cp
	}
	for _, exp := range enrichedExps {
		if exp.Company == "CloudStart Inc" {
			exp.Tags = append(exp.Tags, "linux")
		}
	}

	d := Diff(job, enrichedExps, initialRanked, initialMatch)

	if d.Delta <= 0 {
		t.Errorf("expected positive delta after adding linux tag, got %d", d.Delta)
	}
	if d.NewScore <= d.OldScore {
		t.Errorf("new score (%d) should exceed old score (%d)", d.NewScore, d.OldScore)
	}

	// "linux" should appear in improved skills.
	foundLinux := false
	for _, s := range d.ImprovedSkills {
		if s == "linux" {
			foundLinux = true
		}
	}
	if !foundLinux {
		t.Errorf("expected 'linux' in improved skills, got %v", d.ImprovedSkills)
	}
}

func TestDiff_EmptyPrevious(t *testing.T) {
	job := bigcoJob()
	exps := testExperiences()

	// Previous had no results at all.
	prevMatch := MatchScore{Overall: 0}

	d := Diff(job, exps, nil, prevMatch)

	if d.OldScore != 0 {
		t.Errorf("old score should be 0, got %d", d.OldScore)
	}
	if d.NewScore <= 0 {
		t.Error("new score should be positive")
	}
	if d.Delta != d.NewScore {
		t.Errorf("delta should equal new score when old=0, got delta=%d new=%d", d.Delta, d.NewScore)
	}
}

func TestDiff_AllExperiencesHidden(t *testing.T) {
	job := bigcoJob()
	hidden := []*vault.Experience{
		{Role: "A", Company: "X", Visibility: "hidden", FilePath: "/a.md"},
	}
	prevMatch := MatchScore{Overall: 50}

	d := Diff(job, hidden, nil, prevMatch)

	if d.NewScore != 0 {
		t.Errorf("expected 0 new score with all hidden, got %d", d.NewScore)
	}
	if d.Delta != -50 {
		t.Errorf("expected delta -50, got %d", d.Delta)
	}
}

func TestDiff_PreferredSkillImprovement(t *testing.T) {
	job := &vault.Job{
		Title: "Test", Company: "Co",
		RequiredSkills:  []string{"Go"},
		PreferredSkills: []string{"Rust", "Python"},
		FilePath:        "/job.md",
	}

	// Initial: only Go matched.
	initialExps := []*vault.Experience{{
		Role: "Dev", Company: "A",
		Skills: []string{"Go"}, Visibility: "resume",
		FilePath: "/a.md",
	}}
	initialRanked := RankAt(job, initialExps, 2026)
	initialMatch := Score(job, initialExps, initialRanked)

	// Enriched: add Rust.
	enrichedExps := []*vault.Experience{{
		Role: "Dev", Company: "A",
		Skills: []string{"Go", "Rust"}, Visibility: "resume",
		FilePath: "/a.md",
	}}

	d := Diff(job, enrichedExps, initialRanked, initialMatch)

	foundRust := false
	for _, s := range d.ImprovedSkills {
		if s == "rust" {
			foundRust = true
		}
	}
	if !foundRust {
		t.Errorf("expected 'rust' in improved skills, got %v", d.ImprovedSkills)
	}
}
