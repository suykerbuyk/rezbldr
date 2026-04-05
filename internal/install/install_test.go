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
	// Verify trailing newline.
	if !strings.HasSuffix(string(data), "\n") {
		t.Errorf("file %s does not end with newline", path)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parsing JSON from %s: %v", path, err)
	}
	return m
}

func getMCPServer(t *testing.T, settings map[string]interface{}, name string) map[string]interface{} {
	t.Helper()
	servers, ok := settings["mcpServers"].(map[string]interface{})
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

	if err := Register("/usr/local/bin/rezbldr", dir, ""); err != nil {
		t.Fatalf("Install: %v", err)
	}

	path := filepath.Join(dir, settingsFile)
	settings := readJSON(t, path)
	srv := getMCPServer(t, settings, "rezbldr")

	if srv["command"] != "/usr/local/bin/rezbldr" {
		t.Errorf("command = %v, want /usr/local/bin/rezbldr", srv["command"])
	}

	args := srv["args"].([]interface{})
	if len(args) != 1 || args[0] != "serve" {
		t.Errorf("args = %v, want [serve]", args)
	}
}

func TestInstallPreservesExistingServers(t *testing.T) {
	dir := t.TempDir()

	// Seed with an existing MCP server.
	existing := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"other-tool": map[string]interface{}{
				"command": "/usr/bin/other",
				"args":    []interface{}{},
			},
		},
		"someOtherKey": "preserved",
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(dir, settingsFile), data, 0o644)

	if err := Register("/usr/local/bin/rezbldr", dir, ""); err != nil {
		t.Fatalf("Install: %v", err)
	}

	settings := readJSON(t, filepath.Join(dir, settingsFile))

	// Check rezbldr was added.
	getMCPServer(t, settings, "rezbldr")

	// Check other-tool preserved.
	getMCPServer(t, settings, "other-tool")

	// Check other top-level key preserved.
	if settings["someOtherKey"] != "preserved" {
		t.Errorf("someOtherKey not preserved")
	}
}

func TestInstallWithVaultPath(t *testing.T) {
	dir := t.TempDir()

	if err := Register("/usr/local/bin/rezbldr", dir, "/home/user/vault"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	settings := readJSON(t, filepath.Join(dir, settingsFile))
	srv := getMCPServer(t, settings, "rezbldr")

	args := srv["args"].([]interface{})
	if len(args) != 3 || args[0] != "serve" || args[1] != "--vault" || args[2] != "/home/user/vault" {
		t.Errorf("args = %v, want [serve --vault /home/user/vault]", args)
	}
}

func TestInstallWithoutVaultPath(t *testing.T) {
	dir := t.TempDir()

	if err := Register("/usr/local/bin/rezbldr", dir, ""); err != nil {
		t.Fatalf("Install: %v", err)
	}

	settings := readJSON(t, filepath.Join(dir, settingsFile))
	srv := getMCPServer(t, settings, "rezbldr")

	args := srv["args"].([]interface{})
	if len(args) != 1 || args[0] != "serve" {
		t.Errorf("args = %v, want [serve]", args)
	}
}

func TestInstallIdempotent(t *testing.T) {
	dir := t.TempDir()

	if err := Register("/usr/local/bin/rezbldr", dir, "/vault"); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	first, _ := os.ReadFile(filepath.Join(dir, settingsFile))

	if err := Register("/usr/local/bin/rezbldr", dir, "/vault"); err != nil {
		t.Fatalf("second Install: %v", err)
	}

	second, _ := os.ReadFile(filepath.Join(dir, settingsFile))

	if string(first) != string(second) {
		t.Errorf("Install is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestUninstallRemovesRezbldr(t *testing.T) {
	dir := t.TempDir()

	existing := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"rezbldr": map[string]interface{}{
				"command": "/usr/local/bin/rezbldr",
			},
			"other": map[string]interface{}{
				"command": "/usr/bin/other",
			},
		},
		"keepMe": true,
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(dir, settingsFile), data, 0o644)

	if err := Uninstall(dir, ""); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	settings := readJSON(t, filepath.Join(dir, settingsFile))

	servers := settings["mcpServers"].(map[string]interface{})
	if _, found := servers["rezbldr"]; found {
		t.Error("rezbldr should have been removed")
	}
	if _, found := servers["other"]; !found {
		t.Error("other server should be preserved")
	}
	if settings["keepMe"] != true {
		t.Error("keepMe key should be preserved")
	}
}

func TestUninstallRemovesEmptyMCPServers(t *testing.T) {
	dir := t.TempDir()

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
	os.WriteFile(filepath.Join(dir, settingsFile), data, 0o644)

	if err := Uninstall(dir, ""); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	settings := readJSON(t, filepath.Join(dir, settingsFile))

	if _, found := settings["mcpServers"]; found {
		t.Error("mcpServers should have been removed when empty")
	}
	if settings["keepMe"] != true {
		t.Error("keepMe key should be preserved")
	}
}

func TestUninstallNotPresent(t *testing.T) {
	dir := t.TempDir()

	// No settings files at all — should succeed with "not found" message.
	if err := Uninstall(dir, ""); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
}

func TestUninstallFallsBackToSettingsJSON(t *testing.T) {
	dir := t.TempDir()

	// Put rezbldr in settings.json (not settings.local.json).
	existing := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"rezbldr": map[string]interface{}{
				"command": "/usr/local/bin/rezbldr",
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(dir, fallbackFile), data, 0o644)

	if err := Uninstall(dir, ""); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	settings := readJSON(t, filepath.Join(dir, fallbackFile))
	if _, found := settings["mcpServers"]; found {
		t.Error("mcpServers should have been removed")
	}
}

func TestCopyBinary(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "bin", "rezbldr")

	// CopyBinary copies the test binary (os.Executable) to dst.
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

	// Second copy should succeed (overwrite).
	if err := CopyBinary(dst); err != nil {
		t.Fatalf("second CopyBinary: %v", err)
	}
	info2, _ := os.Stat(dst)

	if info1.Size() != info2.Size() {
		t.Errorf("size mismatch after second copy: %d vs %d", info1.Size(), info2.Size())
	}
}
