// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package scoring

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// ScoredExperience holds the scoring result for a single experience file
// ranked against a job posting.
type ScoredExperience struct {
	File             string   `json:"file"`
	Role             string   `json:"role"`
	Company          string   `json:"company"`
	Score            float64  `json:"score"`
	NormalizedScore  int      `json:"normalized_score"`
	MatchedRequired  []string `json:"matched_required"`
	MatchedPreferred []string `json:"matched_preferred"`
	MatchedTags      []string `json:"matched_tags"`
	Boosted          bool     `json:"boosted"`
	Penalized        bool     `json:"penalized"`
}

// Scoring weights.
const (
	weightRequired  = 2.0
	weightPreferred = 1.0
	weightTag       = 0.5
	boostHighlight  = 0.10  // +10%
	penaltyAge      = -0.30 // -30%
	agePenaltyYears = 10
)

// Rank scores all experience files against a job posting and returns
// results sorted by score descending. Hidden experiences are excluded.
func Rank(job *vault.Job, experiences []*vault.Experience) []ScoredExperience {
	return RankAt(job, experiences, time.Now().Year())
}

// RankAt is like Rank but accepts an explicit current year for testability.
func RankAt(job *vault.Job, experiences []*vault.Experience, currentYear int) []ScoredExperience {
	requiredSet := lowercaseSet(job.RequiredSkills)
	preferredSet := lowercaseSet(job.PreferredSkills)
	tagSet := lowercaseSet(job.Tags)

	// Remove overlaps: required takes priority over preferred and tags,
	// preferred takes priority over tags.
	for k := range requiredSet {
		delete(preferredSet, k)
		delete(tagSet, k)
	}
	for k := range preferredSet {
		delete(tagSet, k)
	}

	var results []ScoredExperience

	for _, exp := range experiences {
		if strings.EqualFold(exp.Visibility, "hidden") {
			continue
		}

		se := scoreExperience(exp, requiredSet, preferredSet, tagSet, currentYear)
		results = append(results, se)
	}

	// Find max score for normalization.
	maxScore := 0.0
	for _, r := range results {
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}

	// Normalize to 0-100.
	for i := range results {
		if maxScore > 0 {
			results[i].NormalizedScore = int(results[i].Score / maxScore * 100)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func scoreExperience(
	exp *vault.Experience,
	requiredSet, preferredSet, tagSet map[string]bool,
	currentYear int,
) ScoredExperience {
	expKeywords := lowercaseSet(append(exp.Tags, exp.Skills...))

	var matchedReq, matchedPref, matchedTags []string
	score := 0.0

	for kw := range expKeywords {
		if requiredSet[kw] {
			matchedReq = append(matchedReq, kw)
			score += weightRequired
		} else if preferredSet[kw] {
			matchedPref = append(matchedPref, kw)
			score += weightPreferred
		} else if tagSet[kw] {
			matchedTags = append(matchedTags, kw)
			score += weightTag
		}
	}

	sort.Strings(matchedReq)
	sort.Strings(matchedPref)
	sort.Strings(matchedTags)

	boosted := false
	penalized := false

	// Highlight boost.
	if exp.Highlight && score > 0 {
		score *= (1 + boostHighlight)
		boosted = true
	}

	// Age penalty.
	if score > 0 {
		endYear, err := vault.ExtractYear(exp.End)
		if err == nil && currentYear-endYear > agePenaltyYears {
			score *= (1 + penaltyAge)
			penalized = true
		}
	}

	return ScoredExperience{
		File:             filepath.Base(exp.FilePath),
		Role:             exp.Role,
		Company:          exp.Company,
		Score:            score,
		MatchedRequired:  matchedReq,
		MatchedPreferred: matchedPref,
		MatchedTags:      matchedTags,
		Boosted:          boosted,
		Penalized:        penalized,
	}
}

// lowercaseSet builds a set from a string slice with all values lowercased.
func lowercaseSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[strings.ToLower(item)] = true
	}
	return s
}
