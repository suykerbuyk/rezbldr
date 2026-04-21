// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Errorf("file %s does not end with newline", path)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parsing JSON from %s: %v", path, err)
	}
	return m
}

// --- MigrateProjectScoped tests ---

func TestMigrateProjectScoped(t *testing.T) {
	dir := t.TempDir()
	claudeJsonPath := filepath.Join(dir, ".claude.json")

	existing := map[string]any{
		"projects": map[string]any{
			"/project-a": map[string]any{
				"mcpServers": map[string]any{
					"rezbldr": map[string]any{"command": "x"},
					"other":   map[string]any{"command": "y"},
				},
				"keepMe": true,
			},
			"/project-b": map[string]any{
				"mcpServers": map[string]any{
					"rezbldr": map[string]any{"command": "x"},
				},
			},
			"/project-c": map[string]any{
				"mcpServers": map[string]any{
					"other": map[string]any{"command": "y"},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(claudeJsonPath, data, 0o644)

	cleaned, err := MigrateProjectScoped(claudeJsonPath)
	if err != nil {
		t.Fatalf("MigrateProjectScoped: %v", err)
	}

	if len(cleaned) != 2 {
		t.Errorf("expected 2 cleaned projects, got %d: %v", len(cleaned), cleaned)
	}

	config := readJSON(t, claudeJsonPath)
	projects := config["projects"].(map[string]any)

	// project-a: rezbldr removed, other preserved.
	projA := projects["/project-a"].(map[string]any)
	servers := projA["mcpServers"].(map[string]any)
	if _, found := servers["rezbldr"]; found {
		t.Error("rezbldr should have been removed from project-a")
	}
	if _, found := servers["other"]; !found {
		t.Error("other should be preserved in project-a")
	}
	if projA["keepMe"] != true {
		t.Error("keepMe should be preserved in project-a")
	}

	// project-b: mcpServers should be removed (was only rezbldr).
	projB := projects["/project-b"].(map[string]any)
	if _, found := projB["mcpServers"]; found {
		t.Error("mcpServers should be removed from project-b (was only rezbldr)")
	}

	// project-c: untouched.
	projC := projects["/project-c"].(map[string]any)
	serversC := projC["mcpServers"].(map[string]any)
	if _, found := serversC["other"]; !found {
		t.Error("other should be preserved in project-c")
	}
}

func TestMigrateProjectScopedNoFile(t *testing.T) {
	cleaned, err := MigrateProjectScoped("/nonexistent/.claude.json")
	if err != nil {
		t.Fatalf("should be no-op for missing file: %v", err)
	}
	if len(cleaned) != 0 {
		t.Errorf("expected empty, got %v", cleaned)
	}
}

func TestMigrateProjectScopedNoRezbldr(t *testing.T) {
	dir := t.TempDir()
	claudeJsonPath := filepath.Join(dir, ".claude.json")

	existing := map[string]any{
		"projects": map[string]any{
			"/project": map[string]any{
				"mcpServers": map[string]any{
					"other": map[string]any{"command": "y"},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(claudeJsonPath, data, 0o644)

	cleaned, err := MigrateProjectScoped(claudeJsonPath)
	if err != nil {
		t.Fatalf("MigrateProjectScoped: %v", err)
	}
	if len(cleaned) != 0 {
		t.Errorf("expected empty, got %v", cleaned)
	}
}

// --- CleanupLegacyMcpJson tests ---

func TestCleanupLegacyMcpJson(t *testing.T) {
	dir := t.TempDir()

	mcpJson := map[string]any{
		"mcpServers": map[string]any{
			"rezbldr": map[string]any{"command": "/old/path/rezbldr"},
			"other":   map[string]any{"command": "/usr/bin/other"},
		},
	}
	data, _ := json.MarshalIndent(mcpJson, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644)

	if err := CleanupLegacyMcpJson(dir); err != nil {
		t.Fatalf("CleanupLegacyMcpJson: %v", err)
	}

	config := readJSON(t, filepath.Join(dir, ".mcp.json"))
	servers := config["mcpServers"].(map[string]any)
	if _, found := servers["rezbldr"]; found {
		t.Error("rezbldr should have been removed from .mcp.json")
	}
	if _, found := servers["other"]; !found {
		t.Error("other should be preserved in .mcp.json")
	}
}

func TestCleanupLegacyMcpJsonNoFile(t *testing.T) {
	dir := t.TempDir()
	if err := CleanupLegacyMcpJson(dir); err != nil {
		t.Fatalf("should be no-op for missing file: %v", err)
	}
}

func TestCleanupLegacyMcpJsonNoRezbldr(t *testing.T) {
	dir := t.TempDir()

	mcpJson := map[string]any{
		"mcpServers": map[string]any{
			"other": map[string]any{"command": "/usr/bin/other"},
		},
	}
	data, _ := json.MarshalIndent(mcpJson, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644)

	if err := CleanupLegacyMcpJson(dir); err != nil {
		t.Fatalf("should be no-op when rezbldr not present: %v", err)
	}
}

func TestCleanupLegacyMcpJsonEmptiesMap(t *testing.T) {
	dir := t.TempDir()
	mcpJson := map[string]any{
		"mcpServers": map[string]any{
			"rezbldr": map[string]any{"command": "/old"},
		},
	}
	data, _ := json.MarshalIndent(mcpJson, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644)

	if err := CleanupLegacyMcpJson(dir); err != nil {
		t.Fatalf("CleanupLegacyMcpJson: %v", err)
	}
	config := readJSON(t, filepath.Join(dir, ".mcp.json"))
	if _, found := config["mcpServers"]; found {
		t.Error("empty mcpServers should have been removed")
	}
}

// --- CopyBinary tests ---

func TestCopyBinary(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "bin", "rezbldr")

	if err := CopyBinary(dst); err != nil {
		t.Fatalf("CopyBinary: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("binary not found at %s: %v", dst, err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("binary at %s is not executable (mode %s)", dst, info.Mode())
	}
}

func TestCopyBinaryIdempotent(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "bin", "rezbldr")

	if err := CopyBinary(dst); err != nil {
		t.Fatalf("first CopyBinary: %v", err)
	}
	info1, _ := os.Stat(dst)

	if err := CopyBinary(dst); err != nil {
		t.Fatalf("second CopyBinary: %v", err)
	}
	info2, _ := os.Stat(dst)

	if info1.Size() != info2.Size() {
		t.Errorf("size mismatch after second copy: %d vs %d", info1.Size(), info2.Size())
	}
}

func TestCopyBinary_ParentIsFile(t *testing.T) {
	// When a regular file occupies the path where a parent directory should
	// be created, MkdirAll fails and CopyBinary should surface that error.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	dst := filepath.Join(blocker, "child", "rezbldr")
	if err := CopyBinary(dst); err == nil {
		t.Error("expected CopyBinary to fail when parent path is a regular file")
	}
}

func TestWriteSettingsFile_MkdirFailure(t *testing.T) {
	// writeSettingsFile should propagate a MkdirAll failure.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	target := filepath.Join(blocker, "sub", "out.json")
	if err := writeSettingsFile(target, map[string]any{"k": "v"}); err == nil {
		t.Error("expected writeSettingsFile to fail when parent path is a regular file")
	}
}

func TestReadSettingsFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.json")
	if err := os.WriteFile(p, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, _, err := readSettingsFile(p); err == nil {
		t.Error("expected readSettingsFile to error on invalid JSON")
	}
}

func TestMigrateProjectScoped_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(p, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := MigrateProjectScoped(p); err == nil {
		t.Error("expected MigrateProjectScoped to error on invalid JSON")
	}
}

func TestCopyBinary_AlreadyAtDest(t *testing.T) {
	// When dst is a symlink pointing at the running binary, CopyBinary
	// should detect the self-link and exit early without re-copying.
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Skipf("os.Executable unavailable: %v", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		t.Skipf("EvalSymlinks(os.Executable) failed: %v", err)
	}
	dst := filepath.Join(dir, "rezbldr")
	if err := os.Symlink(exe, dst); err != nil {
		t.Skipf("cannot create symlink in temp dir: %v", err)
	}
	before, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if err := CopyBinary(dst); err != nil {
		t.Fatalf("CopyBinary self-link: %v", err)
	}
	after, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("lstat after: %v", err)
	}
	// ModTime shouldn't have changed if CopyBinary was a no-op.
	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("CopyBinary modified the destination despite src==dst")
	}
}
