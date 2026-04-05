// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package validate

import (
	"strings"
	"testing"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

func testVault() *vault.Vault {
	return &vault.Vault{
		Contact: &vault.Contact{
			Name:     "Jane Doe",
			Email:    "jane@example.com",
			Phone:    "+1-555-123-4567",
			Location: "Denver, CO",
			LinkedIn: "linkedin.com/in/jane-doe",
		},
		Skills: []vault.SkillEntry{
			{Skill: "Go", Proficiency: "Expert"},
			{Skill: "Python", Proficiency: "Advanced"},
			{Skill: "Kubernetes", Proficiency: "Advanced"},
			{Skill: "Docker", Proficiency: "Expert"},
			{Skill: "AWS", Proficiency: "Advanced"},
		},
		Experiences: []*vault.Experience{
			{Company: "Acme Corp", Role: "Senior Engineer"},
			{Company: "Globex Inc", Role: "Staff Engineer"},
			{Company: "Initech", Role: "Lead Developer"},
		},
	}
}

const validResume = `# Jane Doe

jane@example.com | +1-555-123-4567 | Denver, CO

## Professional Summary

` + loremWords600 + `

## Core Competencies

Go | Python | Kubernetes | Docker | AWS

## Professional Experience

### Senior Engineer
**Acme Corp** | Denver, CO | 2020 – Present

- Built distributed systems in Go serving 10M requests per day.
- Designed Kubernetes deployment pipelines for microservices architecture.

### Staff Engineer
**Globex Inc** | Remote | 2017 – 2020

- Led platform team of 8 engineers building internal developer tools.
- Migrated monolithic Python application to Go microservices.

## Education

**University of Colorado** — B.S. Computer Science (2015)
`

// loremWords600 generates filler to reach the word count minimum.
const loremWords600 = `Experienced software engineer with deep expertise in distributed systems cloud infrastructure and platform engineering. Proven track record of designing and deploying large scale production systems that serve millions of users. Skilled in Go Python Kubernetes Docker and AWS with a focus on reliability performance and developer experience. Passionate about building tools that make engineering teams more productive and shipping high quality software quickly. Strong communicator who bridges technical and business stakeholders to align on priorities and deliver measurable outcomes. Known for taking ownership of complex problems from investigation through implementation and driving projects to completion across organizational boundaries. Experienced in both startup and enterprise environments with the ability to adapt communication and technical approach to the audience. Committed to engineering excellence through code review testing mentoring and continuous improvement. Experienced software engineer with deep expertise in distributed systems cloud infrastructure and platform engineering. Proven track record of designing and deploying large scale production systems that serve millions of users. Skilled in Go Python Kubernetes Docker and AWS with a focus on reliability performance and developer experience. Passionate about building tools that make engineering teams more productive and shipping high quality software quickly. Strong communicator who bridges technical and business stakeholders to align on priorities and deliver measurable outcomes. Known for taking ownership of complex problems from investigation through implementation and driving projects to completion across organizational boundaries. Experienced in both startup and enterprise environments with the ability to adapt communication and technical approach to the audience. Committed to engineering excellence through code review testing mentoring and continuous improvement. Experienced software engineer with deep expertise in distributed systems cloud infrastructure and platform engineering. Proven track record of designing and deploying large scale production systems that serve millions of users. Skilled in Go Python Kubernetes Docker and AWS with a focus on reliability performance and developer experience. Passionate about building tools that make engineering teams more productive and shipping high quality software quickly. Strong communicator who bridges technical and business stakeholders to align on priorities and deliver measurable outcomes. Known for taking ownership of complex problems from investigation through implementation and driving projects to completion across organizational boundaries. Experienced in both startup and enterprise environments with the ability to adapt communication and technical approach to the audience. Committed to engineering excellence through code review testing mentoring and continuous improvement. Experienced software engineer with deep expertise in distributed systems cloud infrastructure and platform engineering. Proven track record of designing and deploying large scale production systems that serve millions of users. Skilled in Go Python Kubernetes Docker and AWS with a focus on reliability performance and developer experience. Passionate about building tools that make engineering teams more productive and shipping high quality software quickly. Strong communicator who bridges technical and business stakeholders to align on priorities and deliver measurable outcomes. Known for taking ownership of complex problems from investigation through implementation and driving projects to completion across organizational boundaries. Experienced in both startup and enterprise environments with the ability to adapt communication and technical approach to the audience. Committed to engineering excellence through code review testing and continuous improvement.`

func TestCountWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"simple", "one two three", 3},
		{"with headings", "# Title\n\nsome words here", 4},
		{"heading markers excluded", "## Section\n\n### Sub", 2},
		{"multiline", "word1\nword2\nword3", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countWords(tt.input)
			if got != tt.want {
				t.Errorf("countWords() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCheckHeadings(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int // number of errors expected
	}{
		{"valid hierarchy", "# Title\n\n## Section\n\n### Sub", 0},
		{"no h1", "## Section\n\n### Sub", 1},
		{"multiple h1", "# One\n\n# Two\n\n## Sub", 1},
		{"skipped level", "# Title\n\n### Sub", 1},
		{"no headings", "just plain text", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := checkHeadings(tt.body)
			if len(errors) != tt.wantCount {
				t.Errorf("checkHeadings() returned %d errors, want %d: %v",
					len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestCheckSkills(t *testing.T) {
	skills := []vault.SkillEntry{
		{Skill: "Go"},
		{Skill: "Python"},
		{Skill: "Kubernetes"},
	}

	tests := []struct {
		name        string
		body        string
		wantUnknown []string
	}{
		{
			"all known",
			"## Core Competencies\n\nGo | Python | Kubernetes",
			nil,
		},
		{
			"one unknown",
			"## Core Competencies\n\nGo | Python | Rust",
			[]string{"Rust"},
		},
		{
			"case insensitive",
			"## Core Competencies\n\ngo | python | kubernetes",
			nil,
		},
		{
			"no competencies section",
			"## Professional Experience\n\nSome text",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkSkills(tt.body, skills)
			if len(got) != len(tt.wantUnknown) {
				t.Errorf("checkSkills() = %v, want %v", got, tt.wantUnknown)
			}
		})
	}
}

func TestCheckCompanies(t *testing.T) {
	experiences := []*vault.Experience{
		{Company: "Acme Corp"},
		{Company: "Globex Inc"},
	}

	tests := []struct {
		name        string
		body        string
		wantUnknown []string
	}{
		{
			"all known",
			"### Senior Engineer\n**Acme Corp** | Denver | 2020\n\n### Staff\n**Globex Inc** | Remote | 2018",
			nil,
		},
		{
			"one unknown",
			"### Engineer\n**Unknown Corp** | Denver | 2020",
			[]string{"Unknown Corp"},
		},
		{
			"parenthetical stripped",
			"### Architect\n**Acme Corp (Contract via Agency)** | Denver | 2020",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkCompanies(tt.body, experiences)
			if len(got) != len(tt.wantUnknown) {
				t.Errorf("checkCompanies() = %v, want %v", got, tt.wantUnknown)
			}
		})
	}
}

func TestCheckContact(t *testing.T) {
	contact := &vault.Contact{
		Email: "jane@example.com",
		Phone: "+1-555-123-4567",
	}

	tests := []struct {
		name string
		body string
		want bool
	}{
		{"both present", "jane@example.com | +1-555-123-4567 | Denver", true},
		{"email missing", "+1-555-123-4567 | Denver", false},
		{"phone missing", "jane@example.com | Denver", false},
		{"nil contact", "anything", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := contact
			if tt.name == "nil contact" {
				c = nil
			}
			got := checkContact(tt.body, c)
			if got != tt.want {
				t.Errorf("checkContact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResumeIntegration(t *testing.T) {
	v := testVault()
	result := Resume(validResume, v)

	if !result.WordCountOK {
		t.Errorf("word count %d not in 600-800 range", result.WordCount)
	}
	if len(result.HeadingErrors) > 0 {
		t.Errorf("unexpected heading errors: %v", result.HeadingErrors)
	}
	if len(result.UnknownSkills) > 0 {
		t.Errorf("unexpected unknown skills: %v", result.UnknownSkills)
	}
	if len(result.UnknownCompanies) > 0 {
		t.Errorf("unexpected unknown companies: %v", result.UnknownCompanies)
	}
	if !result.ContactMatch {
		t.Error("expected contact match")
	}
}

func TestResumeWithProblems(t *testing.T) {
	v := testVault()
	// Short resume with unknown skill and company.
	body := `# Jane Doe

jane@example.com | +1-555-123-4567 | Denver, CO

## Professional Summary

Short resume.

## Core Competencies

Go | Python | Haskell

## Professional Experience

### Engineer
**Mystery Corp** | Denver, CO | 2020 – Present

- Did things.
`
	result := Resume(body, v)

	if result.WordCountOK {
		t.Error("expected word count to be out of range")
	}
	if !strings.Contains(result.UnknownSkills[0], "Haskell") {
		t.Errorf("expected Haskell as unknown skill, got %v", result.UnknownSkills)
	}
	if !strings.Contains(result.UnknownCompanies[0], "Mystery Corp") {
		t.Errorf("expected Mystery Corp as unknown company, got %v", result.UnknownCompanies)
	}
}
