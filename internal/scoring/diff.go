// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package scoring

import (
	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// ScoreDiff captures the change in match score after vault edits.
type ScoreDiff struct {
	OldScore       int      `json:"old_score"`
	NewScore       int      `json:"new_score"`
	Delta          int      `json:"delta"`
	ImprovedSkills []string `json:"improved_skills"`
}

// Diff computes the score change between a previous ranking and the current
// vault state. It re-ranks and re-scores, then compares against the previous
// results to identify improvements.
func Diff(job *vault.Job, experiences []*vault.Experience, previousRanked []ScoredExperience, previousScore MatchScore) ScoreDiff {
	newRanked := Rank(job, experiences)
	newMatch := Score(job, experiences, newRanked)

	// Collect previous skill hits.
	prevReqHits := stringSet(previousScore.RequiredHits)
	prevPrefHits := stringSet(previousScore.PreferredHits)

	// Find newly matched skills.
	var improved []string
	for _, s := range newMatch.RequiredHits {
		if !prevReqHits[s] {
			improved = append(improved, s)
		}
	}
	for _, s := range newMatch.PreferredHits {
		if !prevPrefHits[s] {
			improved = append(improved, s)
		}
	}

	return ScoreDiff{
		OldScore:       previousScore.Overall,
		NewScore:       newMatch.Overall,
		Delta:          newMatch.Overall - previousScore.Overall,
		ImprovedSkills: improved,
	}
}

func stringSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
