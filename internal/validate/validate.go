// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package validate

import (
	"regexp"
	"strings"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// Result holds the outcome of validating a generated resume.
type Result struct {
	WordCount        int      `json:"word_count"`
	WordCountOK      bool     `json:"word_count_ok"`
	HeadingErrors    []string `json:"heading_errors"`
	UnknownSkills    []string `json:"unknown_skills"`
	UnknownCompanies []string `json:"unknown_companies"`
	ContactMatch     bool     `json:"contact_match"`
	Warnings         []string `json:"warnings"`
}

var headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+)`)

// Resume validates a generated resume's markdown body against vault data.
// The body should have frontmatter already stripped.
func Resume(body string, v *vault.Vault) Result {
	var r Result

	r.WordCount = countWords(body)
	r.WordCountOK = r.WordCount >= 600 && r.WordCount <= 800
	if !r.WordCountOK {
		if r.WordCount < 600 {
			r.Warnings = append(r.Warnings, "word count below 600 minimum")
		} else {
			r.Warnings = append(r.Warnings, "word count above 800 maximum")
		}
	}

	r.HeadingErrors = checkHeadings(body)
	r.UnknownSkills = checkSkills(body, v.Skills)
	r.UnknownCompanies = checkCompanies(body, v.Experiences)
	r.ContactMatch = checkContact(body, v.Contact)

	return r
}

// countWords counts whitespace-delimited words in the body text,
// excluding markdown headings markers but including heading text.
func countWords(body string) int {
	count := 0
	for _, word := range strings.Fields(body) {
		// Skip pure markdown heading markers.
		if strings.Trim(word, "#") == "" {
			continue
		}
		count++
	}
	return count
}

// checkHeadings validates markdown heading hierarchy.
func checkHeadings(body string) []string {
	var errors []string
	h1Count := 0
	prevLevel := 0

	for _, line := range strings.Split(body, "\n") {
		m := headingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		level := len(m[1])

		if level == 1 {
			h1Count++
		}

		// Check for skipped heading levels (e.g., h1 -> h3 with no h2).
		if prevLevel > 0 && level > prevLevel+1 {
			errors = append(errors, "heading level skipped: h"+
				itoa(prevLevel)+" followed by h"+itoa(level)+
				" (\""+strings.TrimSpace(m[2])+"\")")
		}
		prevLevel = level
	}

	if h1Count == 0 {
		errors = append(errors, "no h1 heading found")
	} else if h1Count > 1 {
		errors = append(errors, "multiple h1 headings found (expected exactly 1)")
	}

	return errors
}

// checkSkills extracts skills from the "Core Competencies" line and checks
// them against the vault skills inventory.
func checkSkills(body string, skills []vault.SkillEntry) []string {
	knownSkills := make(map[string]bool, len(skills))
	for _, s := range skills {
		knownSkills[strings.ToLower(s.Skill)] = true
	}

	competencies := extractCompetencies(body)
	var unknown []string
	for _, skill := range competencies {
		if !knownSkills[strings.ToLower(skill)] {
			unknown = append(unknown, skill)
		}
	}
	return unknown
}

// extractCompetencies finds the line after a "Core Competencies" or
// "Technical Skills" heading and splits it on " | ".
func extractCompetencies(body string) []string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if headingRe.MatchString(trimmed) {
			text := headingRe.FindStringSubmatch(trimmed)[2]
			lower := strings.ToLower(text)
			if lower == "core competencies" || lower == "technical skills" {
				// The competencies are on the next non-empty line.
				for j := i + 1; j < len(lines); j++ {
					next := strings.TrimSpace(lines[j])
					if next == "" {
						continue
					}
					return splitCompetencies(next)
				}
			}
		}
	}
	return nil
}

// splitCompetencies splits a pipe-delimited competencies line.
func splitCompetencies(line string) []string {
	parts := strings.Split(line, "|")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// checkCompanies extracts company names from h3 headings and checks
// them against vault experience records.
// Resume format: ### Role Title\n**Company Name** | Location | Dates
func checkCompanies(body string, experiences []*vault.Experience) []string {
	knownCompanies := make(map[string]bool, len(experiences))
	for _, e := range experiences {
		knownCompanies[strings.ToLower(e.Company)] = true
	}

	companies := extractCompanyNames(body)
	var unknown []string
	seen := make(map[string]bool)
	for _, company := range companies {
		lower := strings.ToLower(company)
		if !knownCompanies[lower] && !seen[lower] {
			unknown = append(unknown, company)
			seen[lower] = true
		}
	}
	return unknown
}

// extractCompanyNames finds company names from bold-formatted lines
// that follow h3 headings: **Company Name** | Location | Dates
var boldCompanyRe = regexp.MustCompile(`^\*\*(.+?)\*\*`)

func extractCompanyNames(body string) []string {
	lines := strings.Split(body, "\n")
	var companies []string
	inH3 := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if headingRe.MatchString(trimmed) {
			m := headingRe.FindStringSubmatch(trimmed)
			inH3 = len(m[1]) == 3
			continue
		}
		if inH3 {
			if m := boldCompanyRe.FindStringSubmatch(trimmed); m != nil {
				// Strip parenthetical notes like "(Contract via SykeTech LTD)".
				company := m[1]
				if idx := strings.Index(company, "("); idx > 0 {
					company = strings.TrimSpace(company[:idx])
				}
				companies = append(companies, company)
				inH3 = false
			}
		}
	}
	return companies
}

// checkContact verifies that the resume body contains the contact info
// from the vault's contact profile.
func checkContact(body string, contact *vault.Contact) bool {
	if contact == nil {
		return false
	}
	// Check that key contact fields appear in the body.
	checks := []string{contact.Email, contact.Phone}
	for _, check := range checks {
		if check != "" && !strings.Contains(body, check) {
			return false
		}
	}
	return true
}

func itoa(n int) string {
	return string(rune('0' + n))
}
