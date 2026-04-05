// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package scoring

import (
	"strings"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// MatchScore represents the overall match quality between a candidate's
// experience vault and a job posting.
type MatchScore struct {
	Overall           int      `json:"overall"`
	RequiredCoverage  float64  `json:"required_coverage"`
	PreferredCoverage float64  `json:"preferred_coverage"`
	DomainMatch       float64  `json:"domain_match"`
	SeniorityMatch    float64  `json:"seniority_match"`
	RequiredHits      []string `json:"required_hits"`
	RequiredMisses    []string `json:"required_misses"`
	PreferredHits     []string `json:"preferred_hits"`
	PreferredMisses   []string `json:"preferred_misses"`
}

// Overall score component weights.
const (
	weightRequiredCoverage  = 0.60
	weightPreferredCoverage = 0.20
	weightDomain            = 0.10
	weightSeniority         = 0.10
)

// Score computes the overall match score for a job against ranked experience results.
func Score(job *vault.Job, experiences []*vault.Experience, ranked []ScoredExperience) MatchScore {
	ms := MatchScore{}

	// Collect all matched skills across ranked experiences.
	reqHitSet := make(map[string]bool)
	prefHitSet := make(map[string]bool)
	for _, r := range ranked {
		for _, s := range r.MatchedRequired {
			reqHitSet[s] = true
		}
		for _, s := range r.MatchedPreferred {
			prefHitSet[s] = true
		}
	}

	// Required coverage.
	reqLower := lowercaseSlice(job.RequiredSkills)
	ms.RequiredHits, ms.RequiredMisses = splitHitsMisses(reqLower, reqHitSet)
	if len(reqLower) > 0 {
		ms.RequiredCoverage = float64(len(ms.RequiredHits)) / float64(len(reqLower))
	}

	// Preferred coverage.
	prefLower := lowercaseSlice(job.PreferredSkills)
	ms.PreferredHits, ms.PreferredMisses = splitHitsMisses(prefLower, prefHitSet)
	if len(prefLower) > 0 {
		ms.PreferredCoverage = float64(len(ms.PreferredHits)) / float64(len(prefLower))
	}

	// Domain match.
	ms.DomainMatch = computeDomainMatch(job.Domain, experiences, ranked)

	// Seniority match.
	ms.SeniorityMatch = computeSeniorityMatch(job.Seniority, experiences, ranked)

	// Overall weighted score.
	ms.Overall = int((ms.RequiredCoverage*weightRequiredCoverage +
		ms.PreferredCoverage*weightPreferredCoverage +
		ms.DomainMatch*weightDomain +
		ms.SeniorityMatch*weightSeniority) * 100)

	return ms
}

// computeDomainMatch checks if any scoring experience has a matching domain.
func computeDomainMatch(jobDomain string, experiences []*vault.Experience, ranked []ScoredExperience) float64 {
	if jobDomain == "" {
		return 0
	}
	jd := strings.ToLower(jobDomain)

	// Build set of experiences that scored > 0.
	scoringFiles := make(map[string]bool)
	for _, r := range ranked {
		if r.Score > 0 {
			scoringFiles[r.File] = true
		}
	}

	bestMatch := 0.0
	for _, exp := range experiences {
		if exp.Domain == "" {
			continue
		}
		if !scoringFiles[fileBase(exp.FilePath)] {
			continue
		}

		ed := strings.ToLower(exp.Domain)
		if ed == jd {
			return 1.0
		}
		if strings.Contains(ed, jd) || strings.Contains(jd, ed) {
			if bestMatch < 0.5 {
				bestMatch = 0.5
			}
		}
	}
	return bestMatch
}

// seniorityLevels maps normalized seniority labels to ordinal ranks.
var seniorityLevels = map[string]int{
	"junior":     0,
	"mid":        1,
	"mid-senior": 2,
	"senior":     3,
	"staff":      4,
	"principal":  5,
	"fellow":     6,
}

// roleSeniorityKeywords maps role title keywords to seniority levels.
var roleSeniorityKeywords = map[string]string{
	"junior":    "junior",
	"staff":     "staff",
	"senior":    "senior",
	"principal": "principal",
	"fellow":    "fellow",
	"lead":      "senior",
	"architect": "staff",
	"director":  "principal",
}

// computeSeniorityMatch compares the job's seniority against the highest
// seniority among scoring experiences.
func computeSeniorityMatch(jobSeniority string, experiences []*vault.Experience, ranked []ScoredExperience) float64 {
	if jobSeniority == "" {
		return 0
	}

	jobLevel, ok := parseSeniority(jobSeniority)
	if !ok {
		return 0
	}

	// Build set of experiences that scored > 0.
	scoringFiles := make(map[string]bool)
	for _, r := range ranked {
		if r.Score > 0 {
			scoringFiles[r.File] = true
		}
	}

	// Find highest seniority among scoring experiences.
	bestLevel := -1
	for _, exp := range experiences {
		if !scoringFiles[fileBase(exp.FilePath)] {
			continue
		}
		level := inferSeniority(exp.Role)
		if level > bestLevel {
			bestLevel = level
		}
	}

	if bestLevel < 0 {
		return 0
	}

	diff := jobLevel - bestLevel
	if diff < 0 {
		diff = -diff
	}

	switch diff {
	case 0:
		return 1.0
	case 1:
		return 0.5
	default:
		return 0.0
	}
}

// parseSeniority normalizes a seniority string to its ordinal level.
func parseSeniority(s string) (int, bool) {
	s = strings.ToLower(strings.TrimSpace(s))

	// Direct match.
	if level, ok := seniorityLevels[s]; ok {
		return level, true
	}

	// Handle compound forms like "mid-senior level".
	s = strings.TrimSuffix(s, " level")
	if level, ok := seniorityLevels[s]; ok {
		return level, true
	}

	// Try keyword extraction.
	for keyword, label := range roleSeniorityKeywords {
		if strings.Contains(s, keyword) {
			return seniorityLevels[label], true
		}
	}

	return 0, false
}

// inferSeniority extracts a seniority level from a role title.
func inferSeniority(role string) int {
	r := strings.ToLower(role)

	bestLevel := -1
	for keyword, label := range roleSeniorityKeywords {
		if strings.Contains(r, keyword) {
			level := seniorityLevels[label]
			if level > bestLevel {
				bestLevel = level
			}
		}
	}

	// Default: assume mid-level if no keywords found.
	if bestLevel < 0 {
		return seniorityLevels["mid"]
	}
	return bestLevel
}

func splitHitsMisses(items []string, hitSet map[string]bool) (hits, misses []string) {
	for _, item := range items {
		if hitSet[item] {
			hits = append(hits, item)
		} else {
			misses = append(misses, item)
		}
	}
	return hits, misses
}

func lowercaseSlice(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = strings.ToLower(s)
	}
	return out
}

func fileBase(path string) string {
	// Simple basename extraction.
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
