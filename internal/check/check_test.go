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

func TestCheckConfigFile_NotFound(t *testing.T) {
	found, _ := checkConfigFile("/nonexistent/.claude.json", "/project")
	if found {
		t.Error("expected not found for nonexistent file")
	}
}

func TestCheckConfigFile_WithRezbldr(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	content := `{"projects":{"/project":{"mcpServers":{"rezbldr":{"command":"/usr/local/bin/rezbldr"}}}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found, cmd := checkConfigFile(p, "/project")
	if !found {
		t.Error("expected to find rezbldr")
	}
	if cmd != "/usr/local/bin/rezbldr" {
		t.Errorf("expected command /usr/local/bin/rezbldr, got %s", cmd)
	}
}

func TestCheckConfigFile_WrongProject(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	content := `{"projects":{"/other":{"mcpServers":{"rezbldr":{"command":"/usr/local/bin/rezbldr"}}}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found, _ := checkConfigFile(p, "/project")
	if found {
		t.Error("expected not to find rezbldr for different project")
	}
}

func TestCheckConfigFile_WithoutRezbldr(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	content := `{"projects":{"/project":{"mcpServers":{"other":{"command":"other"}}}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found, _ := checkConfigFile(p, "/project")
	if found {
		t.Error("expected not to find rezbldr")
	}
}

func TestCheckConfigFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	os.WriteFile(p, []byte("not json"), 0o644)

	found, _ := checkConfigFile(p, "/project")
	if found {
		t.Error("expected not found for invalid JSON")
	}
}

func TestCheckClaudeConfig_Registered(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")

	// Create a binary so the stat check passes.
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)
	binPath := filepath.Join(binDir, "rezbldr")
	os.WriteFile(binPath, []byte("binary"), 0o755)

	content := `{"projects":{"/project":{"mcpServers":{"rezbldr":{"command":"` + binPath + `"}}}}}`
	os.WriteFile(p, []byte(content), 0o644)

	r := CheckClaudeConfig(p, "/project")
	if r.Status != "ok" {
		t.Errorf("expected ok, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckClaudeConfig_BinaryMissing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	content := `{"projects":{"/project":{"mcpServers":{"rezbldr":{"command":"/nonexistent/rezbldr"}}}}}`
	os.WriteFile(p, []byte(content), 0o644)

	r := CheckClaudeConfig(p, "/project")
	if r.Status != "warn" {
		t.Errorf("expected warn, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckClaudeConfig_NotRegistered(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	content := `{"projects":{}}`
	os.WriteFile(p, []byte(content), 0o644)

	r := CheckClaudeConfig(p, "/project")
	if r.Status != "warn" {
		t.Errorf("expected warn, got %s: %s", r.Status, r.Detail)
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
