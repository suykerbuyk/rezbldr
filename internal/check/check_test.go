// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package check

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	installer "github.com/suykerbuyk/claude-plugin-installer"
)

func testIdentity() installer.Identity {
	return identity()
}

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

// --- Plugin-path install checks ---

func TestCheckPluginAt_FreshInstall(t *testing.T) {
	home := t.TempDir()
	paths := installer.FromHome(home, testIdentity())
	cfg := installer.Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}
	if err := installer.Install(paths, cfg); err != nil {
		t.Fatalf("setup Install: %v", err)
	}

	results := CheckPluginAt(paths)
	if len(results) != 3 {
		t.Fatalf("expected 3 plugin results, got %d: %+v", len(results), results)
	}
	for _, r := range results {
		if r.Status != "ok" {
			t.Errorf("%s: expected ok, got %s (%s)", r.Name, r.Status, r.Detail)
		}
	}
}

func TestCheckPluginAt_NothingInstalled(t *testing.T) {
	home := t.TempDir()
	paths := installer.FromHome(home, testIdentity())
	results := CheckPluginAt(paths)
	if len(results) != 3 {
		t.Fatalf("expected 3 plugin results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != "fail" {
			t.Errorf("%s: expected fail (nothing installed), got %s: %s", r.Name, r.Status, r.Detail)
		}
	}
}

func TestCheckPluginAt_PartialInstall(t *testing.T) {
	// Only generate manifests; leave settings + cache absent.
	home := t.TempDir()
	paths := installer.FromHome(home, testIdentity())
	if err := installer.Generate(paths, installer.Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	results := CheckPluginAt(paths)
	statusBy := make(map[string]string)
	for _, r := range results {
		statusBy[r.Name] = r.Status
	}
	if statusBy["mcp-plugin-files"] != "ok" {
		t.Errorf("mcp-plugin-files: expected ok, got %s", statusBy["mcp-plugin-files"])
	}
	if statusBy["mcp-plugin-settings"] != "fail" {
		t.Errorf("mcp-plugin-settings: expected fail, got %s", statusBy["mcp-plugin-settings"])
	}
	if statusBy["mcp-plugin-cache"] != "fail" {
		t.Errorf("mcp-plugin-cache: expected fail, got %s", statusBy["mcp-plugin-cache"])
	}
}

func TestCheckPluginAt_HealthCheckReadError(t *testing.T) {
	home := t.TempDir()
	paths := installer.FromHome(home, testIdentity())
	// Write an invalid settings.json so HealthCheck surfaces an error.
	_ = os.MkdirAll(filepath.Dir(paths.Settings), 0o755)
	_ = os.WriteFile(paths.Settings, []byte("{not json"), 0o644)

	results := CheckPluginAt(paths)
	if len(results) != 1 {
		t.Fatalf("expected 1 failure result, got %+v", results)
	}
	if results[0].Status != "fail" || results[0].Name != "mcp-plugin" {
		t.Errorf("expected mcp-plugin fail, got %+v", results[0])
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

func TestFindProjectScopedEntries_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	os.WriteFile(p, []byte("{not json"), 0o644)
	found := findProjectScopedEntries(p)
	if len(found) != 0 {
		t.Errorf("expected empty on invalid JSON, got %v", found)
	}
}

func TestFindMcpJsonEntry(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".mcp.json")

	// Missing file.
	if found, _ := findMcpJsonEntry(p); found {
		t.Error("expected not found for missing file")
	}

	// Present.
	os.WriteFile(p, []byte(`{"mcpServers":{"rezbldr":{"command":"x"}}}`), 0o644)
	found, err := findMcpJsonEntry(p)
	if err != nil || !found {
		t.Errorf("expected (true, nil) when present, got (%v, %v)", found, err)
	}

	// Absent.
	os.WriteFile(p, []byte(`{"mcpServers":{"other":{"command":"x"}}}`), 0o644)
	found, err = findMcpJsonEntry(p)
	if err != nil || found {
		t.Errorf("expected (false, nil) when absent, got (%v, %v)", found, err)
	}
}

// --- Run integration ---

func TestRun_ValidVault(t *testing.T) {
	dir := makeVault(t)
	results := Run(dir)

	// Expect 10 results: go, pandoc, git, vault, vault-structure, contact,
	// mcp-plugin-files, mcp-plugin-settings, mcp-plugin-cache, mcp-legacy.
	if len(results) != 10 {
		names := make([]string, len(results))
		for i, r := range results {
			names[i] = r.Name
		}
		t.Fatalf("expected 10 results, got %d: %v", len(results), names)
	}

	// Vault-related checks should pass.
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
	// At minimum vault, vault-structure, contact should fail.
	if failures < 3 {
		t.Errorf("expected at least 3 failures for invalid vault, got %d", failures)
	}
}
