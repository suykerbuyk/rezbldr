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

func TestInstallCreatesNewFile(t *testing.T) {
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

func TestInstallPreservesExistingData(t *testing.T) {
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

	// Check rezbldr was added.
	getProjectMCPServer(t, config, "/home/user/project", "rezbldr")

	// Check other-tool preserved.
	getProjectMCPServer(t, config, "/home/user/project", "other-tool")

	// Check top-level key preserved.
	if config["numStartups"] != float64(42) {
		t.Errorf("numStartups not preserved")
	}

	// Check project-level key preserved.
	projects := config["projects"].(map[string]interface{})
	project := projects["/home/user/project"].(map[string]interface{})
	if project["hasTrustDialogAccepted"] != true {
		t.Errorf("hasTrustDialogAccepted not preserved")
	}
}

func TestInstallWithVaultPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)

	if err := Register("/usr/local/bin/rezbldr", configPath, "/home/user/project", "/home/user/vault"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	config := readJSON(t, configPath)
	srv := getProjectMCPServer(t, config, "/home/user/project", "rezbldr")

	args := srv["args"].([]interface{})
	if len(args) != 3 || args[0] != "serve" || args[1] != "--vault" || args[2] != "/home/user/vault" {
		t.Errorf("args = %v, want [serve --vault /home/user/vault]", args)
	}
}

func TestInstallWithoutVaultPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)

	if err := Register("/usr/local/bin/rezbldr", configPath, "/home/user/project", ""); err != nil {
		t.Fatalf("Register: %v", err)
	}

	config := readJSON(t, configPath)
	srv := getProjectMCPServer(t, config, "/home/user/project", "rezbldr")

	args := srv["args"].([]interface{})
	if len(args) != 1 || args[0] != "serve" {
		t.Errorf("args = %v, want [serve]", args)
	}
}

func TestInstallIdempotent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)

	if err := Register("/usr/local/bin/rezbldr", configPath, "/project", "/vault"); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	first, _ := os.ReadFile(configPath)

	if err := Register("/usr/local/bin/rezbldr", configPath, "/project", "/vault"); err != nil {
		t.Fatalf("second Register: %v", err)
	}

	second, _ := os.ReadFile(configPath)

	if string(first) != string(second) {
		t.Errorf("Register is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

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

func TestUninstallRemovesEmptyMCPServers(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)

	existing := map[string]interface{}{
		"projects": map[string]interface{}{
			"/project": map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"rezbldr": map[string]interface{}{
						"command": "/usr/local/bin/rezbldr",
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

	if _, found := project["mcpServers"]; found {
		t.Error("mcpServers should have been removed when empty")
	}
	if project["keepMe"] != true {
		t.Error("keepMe key should be preserved")
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

func TestUninstallCleansUpLegacy(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFile)
	legacyDir := filepath.Join(dir, "legacy")
	os.MkdirAll(legacyDir, 0o755)

	// Put rezbldr in legacy settings.json.
	legacy := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"rezbldr": map[string]interface{}{
				"command": "/usr/local/bin/rezbldr",
			},
		},
	}
	data, _ := json.MarshalIndent(legacy, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(legacyDir, legacySettingsFile), data, 0o644)

	if err := Uninstall(configPath, "/project", legacyDir, ""); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	settings := readJSON(t, filepath.Join(legacyDir, legacySettingsFile))
	if _, found := settings["mcpServers"]; found {
		t.Error("legacy mcpServers should have been removed")
	}
}

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
