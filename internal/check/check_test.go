// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package check

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// makeVault creates a minimal valid vault structure in a temp directory.
func makeVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"profile", "jobs/target", "resumes"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "profile", "contact.md"), []byte("---\nname: Test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCheckGo(t *testing.T) {
	r := checkGo()
	if r.Status != "ok" {
		t.Errorf("expected ok, got %s", r.Status)
	}
	if r.Detail != runtime.Version() {
		t.Errorf("expected %s, got %s", runtime.Version(), r.Detail)
	}
}

func TestCheckVaultPath_Valid(t *testing.T) {
	dir := makeVault(t)
	r := checkVaultPath(dir)
	if r.Status != "ok" {
		t.Errorf("expected ok, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckVaultPath_Missing(t *testing.T) {
	r := checkVaultPath("/nonexistent/path/xyz")
	if r.Status != "fail" {
		t.Errorf("expected fail, got %s", r.Status)
	}
}

func TestCheckVaultStructure_Valid(t *testing.T) {
	dir := makeVault(t)
	r := checkVaultStructure(dir)
	if r.Status != "ok" {
		t.Errorf("expected ok, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckVaultStructure_MissingDirs(t *testing.T) {
	dir := t.TempDir()
	// Only create profile, skip jobs/target and resumes.
	os.MkdirAll(filepath.Join(dir, "profile"), 0o755)

	r := checkVaultStructure(dir)
	if r.Status != "fail" {
		t.Errorf("expected fail, got %s", r.Status)
	}
	if r.Detail == "" {
		t.Error("expected detail about missing directories")
	}
}

func TestCheckVaultStructure_AllMissing(t *testing.T) {
	dir := t.TempDir()
	r := checkVaultStructure(dir)
	if r.Status != "fail" {
		t.Errorf("expected fail, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckContactFile_Valid(t *testing.T) {
	dir := makeVault(t)
	r := checkContactFile(dir)
	if r.Status != "ok" {
		t.Errorf("expected ok, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckContactFile_Missing(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "profile"), 0o755)
	// Don't create contact.md.

	r := checkContactFile(dir)
	if r.Status != "fail" {
		t.Errorf("expected fail, got %s", r.Status)
	}
}

func TestCheckSettingsFile_NotFound(t *testing.T) {
	found, _ := checkSettingsFile("/nonexistent/settings.json")
	if found {
		t.Error("expected not found for nonexistent file")
	}
}

func TestCheckSettingsFile_WithRezbldr(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	content := `{"mcpServers":{"rezbldr":{"command":"rezbldr"}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found, file := checkSettingsFile(p)
	if !found {
		t.Error("expected to find rezbldr")
	}
	if file != p {
		t.Errorf("expected file %s, got %s", p, file)
	}
}

func TestCheckSettingsFile_WithoutRezbldr(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	content := `{"mcpServers":{"other":{"command":"other"}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found, _ := checkSettingsFile(p)
	if found {
		t.Error("expected not to find rezbldr")
	}
}

func TestCheckSettingsFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	os.WriteFile(p, []byte("not json"), 0o644)

	found, _ := checkSettingsFile(p)
	if found {
		t.Error("expected not found for invalid JSON")
	}
}

func TestRun_ValidVault(t *testing.T) {
	dir := makeVault(t)
	results := Run(dir)

	// Should have 7 results.
	if len(results) != 7 {
		t.Fatalf("expected 7 results, got %d", len(results))
	}

	// Check that vault-related checks pass.
	for _, r := range results {
		switch r.Name {
		case "go", "vault", "vault-structure", "contact":
			if r.Status != "ok" {
				t.Errorf("check %q: expected ok, got %s: %s", r.Name, r.Status, r.Detail)
			}
		}
	}
}

func TestRun_InvalidVault(t *testing.T) {
	results := Run("/nonexistent/vault/path")

	failures := 0
	for _, r := range results {
		if r.Status == "fail" {
			failures++
		}
	}
	// vault, vault-structure, contact should all fail.
	if failures < 3 {
		t.Errorf("expected at least 3 failures for invalid vault, got %d", failures)
	}
}
