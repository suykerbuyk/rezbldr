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

func readJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Errorf("file %s does not end with newline", path)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parsing JSON from %s: %v", path, err)
	}
	return m
}

func getProjectMCPServer(t *testing.T, config map[string]interface{}, projectDir, name string) map[string]interface{} {
	t.Helper()
	projects, ok := config["projects"].(map[string]interface{})
	if !ok {
		t.Fatal("projects key missing or not a map")
	}
	project, ok := projects[projectDir].(map[string]interface{})
	if !ok {
		t.Fatalf("projects[%s] missing or not a map", projectDir)
	}
	servers, ok := project["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers key missing or not a map")
	}
	srv, ok := servers[name].(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers.%s missing or not a map", name)
	}
	return srv
}

func getGlobalMCPServer(t *testing.T, config map[string]interface{}, name string) map[string]interface{} {
	t.Helper()
	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers key missing or not a map")
	}
	srv, ok := servers[name].(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers.%s missing or not a map", name)
	}
	return srv
}

// --- Legacy Register tests (still used by deprecated Install) ---

func TestRegisterCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)

	if err := Register("/usr/local/bin/rezbldr", configPath, "/home/user/project", ""); err != nil {
		t.Fatalf("Register: %v", err)
	}

	config := readJSON(t, configPath)
	srv := getProjectMCPServer(t, config, "/home/user/project", "rezbldr")

	if srv["command"] != "/usr/local/bin/rezbldr" {
		t.Errorf("command = %v, want /usr/local/bin/rezbldr", srv["command"])
	}
	if srv["type"] != "stdio" {
		t.Errorf("type = %v, want stdio", srv["type"])
	}

	args := srv["args"].([]interface{})
	if len(args) != 1 || args[0] != "serve" {
		t.Errorf("args = %v, want [serve]", args)
	}
}

func TestRegisterPreservesExistingData(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)

	existing := map[string]interface{}{
		"numStartups": float64(42),
		"projects": map[string]interface{}{
			"/home/user/project": map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"other-tool": map[string]interface{}{
						"command": "/usr/bin/other",
						"args":    []interface{}{},
					},
				},
				"hasTrustDialogAccepted": true,
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(configPath, data, 0o644)

	if err := Register("/usr/local/bin/rezbldr", configPath, "/home/user/project", ""); err != nil {
		t.Fatalf("Register: %v", err)
	}

	config := readJSON(t, configPath)
	getProjectMCPServer(t, config, "/home/user/project", "rezbldr")
	getProjectMCPServer(t, config, "/home/user/project", "other-tool")

	if config["numStartups"] != float64(42) {
		t.Errorf("numStartups not preserved")
	}

	projects := config["projects"].(map[string]interface{})
	project := projects["/home/user/project"].(map[string]interface{})
	if project["hasTrustDialogAccepted"] != true {
		t.Errorf("hasTrustDialogAccepted not preserved")
	}
}

// --- RegisterGlobal tests ---

func TestRegisterGlobalCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	if err := RegisterGlobal("/usr/local/bin/rezbldr", settingsPath, ""); err != nil {
		t.Fatalf("RegisterGlobal: %v", err)
	}

	config := readJSON(t, settingsPath)
	srv := getGlobalMCPServer(t, config, "rezbldr")

	if srv["command"] != "/usr/local/bin/rezbldr" {
		t.Errorf("command = %v, want /usr/local/bin/rezbldr", srv["command"])
	}
	// Should NOT have type or env keys (vibe-vault convention).
	if _, ok := srv["type"]; ok {
		t.Error("stanza should not have 'type' key")
	}
	if _, ok := srv["env"]; ok {
		t.Error("stanza should not have 'env' key")
	}

	args := srv["args"].([]interface{})
	if len(args) != 1 || args[0] != "serve" {
		t.Errorf("args = %v, want [serve]", args)
	}
}

func TestRegisterGlobalWithVault(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	if err := RegisterGlobal("/usr/local/bin/rezbldr", settingsPath, "/home/user/vault"); err != nil {
		t.Fatalf("RegisterGlobal: %v", err)
	}

	config := readJSON(t, settingsPath)
	srv := getGlobalMCPServer(t, config, "rezbldr")

	args := srv["args"].([]interface{})
	if len(args) != 3 || args[0] != "serve" || args[1] != "--vault" || args[2] != "/home/user/vault" {
		t.Errorf("args = %v, want [serve --vault /home/user/vault]", args)
	}
}

func TestRegisterGlobalPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	existing := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"vibe-vault": map[string]interface{}{
				"command": "/usr/local/bin/vv",
				"args":    []interface{}{"serve"},
			},
		},
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Read"},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(settingsPath, data, 0o644)

	if err := RegisterGlobal("/usr/local/bin/rezbldr", settingsPath, ""); err != nil {
		t.Fatalf("RegisterGlobal: %v", err)
	}

	config := readJSON(t, settingsPath)
	// rezbldr added.
	getGlobalMCPServer(t, config, "rezbldr")
	// vibe-vault preserved.
	getGlobalMCPServer(t, config, "vibe-vault")
	// permissions preserved.
	if _, ok := config["permissions"]; !ok {
		t.Error("permissions key not preserved")
	}
}

func TestRegisterGlobalIdempotent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	if err := RegisterGlobal("/usr/local/bin/rezbldr", settingsPath, "/vault"); err != nil {
		t.Fatalf("first RegisterGlobal: %v", err)
	}
	first, _ := os.ReadFile(settingsPath)

	if err := RegisterGlobal("/usr/local/bin/rezbldr", settingsPath, "/vault"); err != nil {
		t.Fatalf("second RegisterGlobal: %v", err)
	}
	second, _ := os.ReadFile(settingsPath)

	if string(first) != string(second) {
		t.Errorf("RegisterGlobal is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// --- UnregisterGlobal tests ---

func TestUnregisterGlobal(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	existing := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"rezbldr": map[string]interface{}{
				"command": "/usr/local/bin/rezbldr",
			},
			"other": map[string]interface{}{
				"command": "/usr/bin/other",
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(settingsPath, data, 0o644)

	if err := UnregisterGlobal(settingsPath); err != nil {
		t.Fatalf("UnregisterGlobal: %v", err)
	}

	config := readJSON(t, settingsPath)
	servers := config["mcpServers"].(map[string]interface{})
	if _, found := servers["rezbldr"]; found {
		t.Error("rezbldr should have been removed")
	}
	if _, found := servers["other"]; !found {
		t.Error("other server should be preserved")
	}
}

func TestUnregisterGlobalRemovesEmptyMCPServers(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	existing := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"rezbldr": map[string]interface{}{
				"command": "/usr/local/bin/rezbldr",
			},
		},
		"keepMe": true,
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(settingsPath, data, 0o644)

	if err := UnregisterGlobal(settingsPath); err != nil {
		t.Fatalf("UnregisterGlobal: %v", err)
	}

	config := readJSON(t, settingsPath)
	if _, found := config["mcpServers"]; found {
		t.Error("mcpServers should have been removed when empty")
	}
	if config["keepMe"] != true {
		t.Error("keepMe should be preserved")
	}
}

func TestUnregisterGlobalNoFile(t *testing.T) {
	if err := UnregisterGlobal("/nonexistent/settings.json"); err != nil {
		t.Fatalf("UnregisterGlobal should be no-op for missing file: %v", err)
	}
}

func TestUnregisterGlobalNotPresent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	os.WriteFile(settingsPath, []byte("{}\n"), 0o644)

	if err := UnregisterGlobal(settingsPath); err != nil {
		t.Fatalf("UnregisterGlobal should be no-op when not present: %v", err)
	}
}

// --- MigrateProjectScoped tests ---

func TestMigrateProjectScoped(t *testing.T) {
	dir := t.TempDir()
	claudeJsonPath := filepath.Join(dir, ".claude.json")

	existing := map[string]interface{}{
		"projects": map[string]interface{}{
			"/project-a": map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"rezbldr": map[string]interface{}{"command": "x"},
					"other":   map[string]interface{}{"command": "y"},
				},
				"keepMe": true,
			},
			"/project-b": map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"rezbldr": map[string]interface{}{"command": "x"},
				},
			},
			"/project-c": map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"other": map[string]interface{}{"command": "y"},
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
	projects := config["projects"].(map[string]interface{})

	// project-a: rezbldr removed, other preserved.
	projA := projects["/project-a"].(map[string]interface{})
	servers := projA["mcpServers"].(map[string]interface{})
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
	projB := projects["/project-b"].(map[string]interface{})
	if _, found := projB["mcpServers"]; found {
		t.Error("mcpServers should be removed from project-b (was only rezbldr)")
	}

	// project-c: untouched.
	projC := projects["/project-c"].(map[string]interface{})
	serversC := projC["mcpServers"].(map[string]interface{})
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

	existing := map[string]interface{}{
		"projects": map[string]interface{}{
			"/project": map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"other": map[string]interface{}{"command": "y"},
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

	mcpJson := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"rezbldr": map[string]interface{}{"command": "/old/path/rezbldr"},
			"other":   map[string]interface{}{"command": "/usr/bin/other"},
		},
	}
	data, _ := json.MarshalIndent(mcpJson, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644)

	if err := CleanupLegacyMcpJson(dir); err != nil {
		t.Fatalf("CleanupLegacyMcpJson: %v", err)
	}

	config := readJSON(t, filepath.Join(dir, ".mcp.json"))
	servers := config["mcpServers"].(map[string]interface{})
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

	mcpJson := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"other": map[string]interface{}{"command": "/usr/bin/other"},
		},
	}
	data, _ := json.MarshalIndent(mcpJson, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644)

	if err := CleanupLegacyMcpJson(dir); err != nil {
		t.Fatalf("should be no-op when rezbldr not present: %v", err)
	}
}

// --- Uninstall tests ---

func TestUninstallRemovesRezbldr(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)

	existing := map[string]interface{}{
		"projects": map[string]interface{}{
			"/project": map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"rezbldr": map[string]interface{}{
						"command": "/usr/local/bin/rezbldr",
					},
					"other": map[string]interface{}{
						"command": "/usr/bin/other",
					},
				},
				"keepMe": true,
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(configPath, data, 0o644)

	legacyDir := filepath.Join(dir, "legacy")
	os.MkdirAll(legacyDir, 0o755)

	if err := Uninstall(configPath, "/project", legacyDir, ""); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	config := readJSON(t, configPath)
	projects := config["projects"].(map[string]interface{})
	project := projects["/project"].(map[string]interface{})
	servers := project["mcpServers"].(map[string]interface{})

	if _, found := servers["rezbldr"]; found {
		t.Error("rezbldr should have been removed")
	}
	if _, found := servers["other"]; !found {
		t.Error("other server should be preserved")
	}
	if project["keepMe"] != true {
		t.Error("keepMe key should be preserved")
	}
}

func TestUninstallCleansGlobalAndLegacy(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)
	legacyDir := filepath.Join(dir, "legacy")
	os.MkdirAll(legacyDir, 0o755)

	// Put rezbldr in legacy settings.json (also the global location).
	legacy := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"rezbldr": map[string]interface{}{"command": "/usr/local/bin/rezbldr"},
		},
	}
	data, _ := json.MarshalIndent(legacy, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(legacyDir, legacySettingsFile), data, 0o644)

	// Put rezbldr in .mcp.json.
	mcpJson := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"rezbldr": map[string]interface{}{"command": "/old/rezbldr"},
		},
	}
	data, _ = json.MarshalIndent(mcpJson, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(legacyDir, ".mcp.json"), data, 0o644)

	if err := Uninstall(configPath, "/project", legacyDir, ""); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	// settings.json cleaned.
	settings := readJSON(t, filepath.Join(legacyDir, legacySettingsFile))
	if _, found := settings["mcpServers"]; found {
		t.Error("legacy settings.json mcpServers should have been removed")
	}

	// .mcp.json cleaned.
	mcpConfig := readJSON(t, filepath.Join(legacyDir, ".mcp.json"))
	if _, found := mcpConfig["mcpServers"]; found {
		t.Error(".mcp.json mcpServers should have been removed")
	}
}

func TestUninstallNotPresent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "nonexistent.json")
	legacyDir := filepath.Join(dir, "legacy")

	if err := Uninstall(configPath, "/project", legacyDir, ""); err != nil {
		t.Fatalf("Uninstall: %v", err)
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
