// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

import (
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	v, err := Open(vaultPath())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if v.Contact == nil {
		t.Fatal("expected non-nil Contact")
	}
	assertEqual(t, "Contact.Name", v.Contact.Name, "Test User")

	if len(v.Skills) != 10 {
		t.Errorf("Skills count: got %d, want 10", len(v.Skills))
	}

	if len(v.Experiences) != 4 {
		t.Errorf("Experiences count: got %d, want 4", len(v.Experiences))
	}

	if len(v.Jobs) != 4 {
		t.Errorf("Jobs count: got %d, want 4", len(v.Jobs))
	}

	if len(v.Resumes) != 1 {
		t.Errorf("Resumes count: got %d, want 1", len(v.Resumes))
	}

	if len(v.CoverLetters) != 1 {
		t.Errorf("CoverLetters count: got %d, want 1", len(v.CoverLetters))
	}

	if len(v.Training) != 1 {
		t.Errorf("Training count: got %d, want 1", len(v.Training))
	}
}

func TestOpen_InvalidRoot(t *testing.T) {
	_, err := Open("/nonexistent/vault")
	if err == nil {
		t.Fatal("expected error for nonexistent vault root")
	}
}

func TestOpen_MissingContact(t *testing.T) {
	// Use a directory that exists but has no profile/contact.md.
	_, err := Open(vaultPath("experience"))
	if err == nil {
		t.Fatal("expected error for vault without profile/contact.md")
	}
}

func TestVault_LatestJob(t *testing.T) {
	v, err := Open(vaultPath())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	job, err := v.LatestJob()
	if err != nil {
		t.Fatalf("LatestJob: %v", err)
	}

	// 2026-04-03 is the latest date prefix in our fixtures.
	assertEqual(t, "LatestJob.Title", job.Title, "Staff Software Engineer, Storage")
}

func TestVault_LatestResume(t *testing.T) {
	v, err := Open(vaultPath())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	r, err := v.LatestResume()
	if err != nil {
		t.Fatalf("LatestResume: %v", err)
	}

	assertEqual(t, "LatestResume.Model", r.Model, "claude-opus-4-6")
}

func TestVault_LoadJob_Relative(t *testing.T) {
	v, err := Open(vaultPath())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	job, err := v.LoadJob("jobs/target/2026-04-01-bigco-senior-storage-engineer.md")
	if err != nil {
		t.Fatalf("LoadJob relative: %v", err)
	}

	assertEqual(t, "Title", job.Title, "Senior Storage Engineer")
}

func TestVault_LoadJob_Absolute(t *testing.T) {
	v, err := Open(vaultPath())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Build an actual absolute path for the test.
	absPath, err := filepath.Abs(vaultPath("jobs", "target", "2026-04-01-bigco-senior-storage-engineer.md"))
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	job, err := v.LoadJob(absPath)
	if err != nil {
		t.Fatalf("LoadJob absolute: %v", err)
	}

	assertEqual(t, "Title", job.Title, "Senior Storage Engineer")
}
