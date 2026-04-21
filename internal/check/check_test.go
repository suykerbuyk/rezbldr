// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package check

import (
	"encoding/json"
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

	r := checkContactFile(dir)
	if r.Status != "fail" {
		t.Errorf("expected fail, got %s", r.Status)
	}
}

// --- Global registration checks ---

func TestFindGlobalMCPServer_NotFound(t *testing.T) {
	found, _ := findGlobalMCPServer("/nonexistent/settings.json")
	if found {
		t.Error("expected not found for nonexistent file")
	}
}

func TestFindGlobalMCPServer_WithRezbldr(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	content := `{"mcpServers":{"rezbldr":{"command":"/usr/local/bin/rezbldr","args":["serve"]}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found, cmd := findGlobalMCPServer(p)
	if !found {
		t.Error("expected to find rezbldr")
	}
	if cmd != "/usr/local/bin/rezbldr" {
		t.Errorf("expected command /usr/local/bin/rezbldr, got %s", cmd)
	}
}

func TestFindGlobalMCPServer_WithoutRezbldr(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	content := `{"mcpServers":{"other":{"command":"other"}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found, _ := findGlobalMCPServer(p)
	if found {
		t.Error("expected not to find rezbldr")
	}
}

func TestFindGlobalMCPServer_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	os.WriteFile(p, []byte("not json"), 0o644)

	found, _ := findGlobalMCPServer(p)
	if found {
		t.Error("expected not found for invalid JSON")
	}
}

func TestCheckGlobalConfig_Registered(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)
	binPath := filepath.Join(binDir, "rezbldr")
	os.WriteFile(binPath, []byte("binary"), 0o755)

	content := `{"mcpServers":{"rezbldr":{"command":"` + binPath + `","args":["serve"]}}}`
	os.WriteFile(p, []byte(content), 0o644)

	r := CheckGlobalConfig(p)
	if r.Status != "ok" {
		t.Errorf("expected ok, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckGlobalConfig_BinaryMissing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	content := `{"mcpServers":{"rezbldr":{"command":"/nonexistent/rezbldr","args":["serve"]}}}`
	os.WriteFile(p, []byte(content), 0o644)

	r := CheckGlobalConfig(p)
	if r.Status != "warn" {
		t.Errorf("expected warn, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckGlobalConfig_NotRegistered(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	content := `{"mcpServers":{}}`
	os.WriteFile(p, []byte(content), 0o644)

	r := CheckGlobalConfig(p)
	if r.Status != "warn" {
		t.Errorf("expected warn, got %s: %s", r.Status, r.Detail)
	}
}

// --- Legacy registration checks ---

func TestFindProjectScopedEntries_Found(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	content := `{"projects":{"/project-a":{"mcpServers":{"rezbldr":{"command":"x"}}},"/project-b":{"mcpServers":{"other":{"command":"y"}}}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found := findProjectScopedEntries(p)
	if len(found) != 1 || found[0] != "/project-a" {
		t.Errorf("expected [/project-a], got %v", found)
	}
}

func TestFindProjectScopedEntries_None(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	content := `{"projects":{"/project":{"mcpServers":{"other":{"command":"y"}}}}}`
	os.WriteFile(p, []byte(content), 0o644)

	found := findProjectScopedEntries(p)
	if len(found) != 0 {
		t.Errorf("expected empty, got %v", found)
	}
}

func TestFindProjectScopedEntries_NoFile(t *testing.T) {
	found := findProjectScopedEntries("/nonexistent/.claude.json")
	if len(found) != 0 {
		t.Errorf("expected empty, got %v", found)
	}
}

func TestRun_ValidVault(t *testing.T) {
	dir := makeVault(t)
	results := Run(dir)

	// Should have 8 results (go, pandoc, git, vault, vault-structure, contact, mcp-global, mcp-legacy).
	if len(results) != 8 {
		names := make([]string, len(results))
		for i, r := range results {
			names[i] = r.Name
		}
		t.Fatalf("expected 8 results, got %d: %v", len(results), names)
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

// helper for install tests that need JSON reading
func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
