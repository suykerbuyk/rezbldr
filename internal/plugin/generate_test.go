// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readJSONFile(t *testing.T, path string) map[string]any {
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
		t.Fatalf("parsing %s: %v", path, err)
	}
	return m
}

func testConfig() Config {
	return Config{
		Version:    "0.2.0",
		BinaryPath: "/tmp/fake-home/.local/bin/rezbldr",
	}
}

func TestGenerate_CreatesAllThreeFiles(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)

	if err := Generate(paths, testConfig()); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	for _, path := range []string{paths.MarketplaceManifest, paths.PluginManifest, paths.McpJson} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist: %v", path, err)
		}
	}
}

func TestGenerate_MarketplaceManifestShape(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	if err := Generate(paths, testConfig()); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	m := readJSONFile(t, paths.MarketplaceManifest)
	if m["$schema"] != marketplaceSchemaURL {
		t.Errorf("$schema = %v, want %v", m["$schema"], marketplaceSchemaURL)
	}
	if m["name"] != MarketplaceName {
		t.Errorf("name = %v, want %v", m["name"], MarketplaceName)
	}
	if m["description"] != marketplaceDesc {
		t.Errorf("description = %v, want %v", m["description"], marketplaceDesc)
	}

	owner, ok := m["owner"].(map[string]any)
	if !ok {
		t.Fatalf("owner missing or wrong type: %T", m["owner"])
	}
	if owner["name"] != ownerName || owner["email"] != ownerEmail {
		t.Errorf("owner = %v, want name=%q email=%q", owner, ownerName, ownerEmail)
	}

	plugins, ok := m["plugins"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("plugins = %v, want slice of length 1", m["plugins"])
	}
	p := plugins[0].(map[string]any)
	if p["name"] != PluginName || p["source"] != "./"+PluginName {
		t.Errorf("plugin ref = %v, want name=%q source=./%s", p, PluginName, PluginName)
	}
}

func TestGenerate_PluginManifestShape(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := testConfig()
	if err := Generate(paths, cfg); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	m := readJSONFile(t, paths.PluginManifest)
	if m["name"] != PluginName {
		t.Errorf("name = %v, want %v", m["name"], PluginName)
	}
	if m["version"] != cfg.Version {
		t.Errorf("version = %v, want %v", m["version"], cfg.Version)
	}
	if m["description"] != pluginDesc {
		t.Errorf("description = %v, want %v", m["description"], pluginDesc)
	}
}

func TestGenerate_PluginHasAuthor(t *testing.T) {
	// Matches vibe-vault iteration 94 requirement: Claude Code's plugin loader
	// expects an "author" object in plugin.json.
	home := t.TempDir()
	paths := FromHome(home)
	if err := Generate(paths, testConfig()); err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	m := readJSONFile(t, paths.PluginManifest)
	author, ok := m["author"].(map[string]any)
	if !ok {
		t.Fatalf("author missing or wrong type: %T", m["author"])
	}
	if author["name"] != ownerName {
		t.Errorf("author.name = %v, want %v", author["name"], ownerName)
	}
}

func TestGenerate_McpManifestShape(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := testConfig()
	cfg.ExtraArgs = []string{"--vault", "/vault/path"}
	if err := Generate(paths, cfg); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	m := readJSONFile(t, paths.McpJson)
	entry, ok := m[PluginName].(map[string]any)
	if !ok {
		t.Fatalf("%s entry missing: %v", PluginName, m)
	}
	if entry["command"] != cfg.BinaryPath {
		t.Errorf("command = %v, want %v", entry["command"], cfg.BinaryPath)
	}
	args, ok := entry["args"].([]any)
	if !ok {
		t.Fatalf("args missing: %v", entry)
	}
	wantArgs := []any{"serve", "--vault", "/vault/path"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

func TestGenerate_McpManifestNoExtraArgs(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	if err := Generate(paths, testConfig()); err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	m := readJSONFile(t, paths.McpJson)
	entry := m[PluginName].(map[string]any)
	args := entry["args"].([]any)
	if !reflect.DeepEqual(args, []any{"serve"}) {
		t.Errorf("args = %v, want [serve]", args)
	}
}

func TestGenerate_Idempotent(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := testConfig()

	if err := Generate(paths, cfg); err != nil {
		t.Fatalf("first Generate error = %v", err)
	}
	first, err := os.ReadFile(paths.PluginManifest)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if err := Generate(paths, cfg); err != nil {
		t.Fatalf("second Generate error = %v", err)
	}
	second, err := os.ReadFile(paths.PluginManifest)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("Generate not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestGenerate_VersionUpdate(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := testConfig()

	cfg.Version = "0.1.0"
	if err := Generate(paths, cfg); err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	m := readJSONFile(t, paths.PluginManifest)
	if m["version"] != "0.1.0" {
		t.Fatalf("initial version = %v, want 0.1.0", m["version"])
	}

	cfg.Version = "0.2.0"
	if err := Generate(paths, cfg); err != nil {
		t.Fatalf("Generate (update) error = %v", err)
	}
	m = readJSONFile(t, paths.PluginManifest)
	if m["version"] != "0.2.0" {
		t.Errorf("updated version = %v, want 0.2.0", m["version"])
	}
}

func TestGenerate_MissingBinaryFallback(t *testing.T) {
	// Empty BinaryPath should resolve via exec.LookPath or os.Executable,
	// both of which return something non-empty in test binary contexts.
	home := t.TempDir()
	paths := FromHome(home)
	cfg := Config{Version: "0.0.1"} // no BinaryPath
	if err := Generate(paths, cfg); err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	m := readJSONFile(t, paths.McpJson)
	entry := m[PluginName].(map[string]any)
	command, ok := entry["command"].(string)
	if !ok || command == "" {
		t.Errorf("command should have been resolved to non-empty string, got %v", entry["command"])
	}
}

func TestGenerate_DefaultVersionWhenMissing(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := Config{BinaryPath: "/tmp/rezbldr"} // no Version
	if err := Generate(paths, cfg); err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	m := readJSONFile(t, paths.PluginManifest)
	if m["version"] != "0.0.0-dev" {
		t.Errorf("default version = %v, want 0.0.0-dev", m["version"])
	}
}

func TestGenerate_CreatesParentDirs(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	// No pre-existing directories should be required.
	if err := Generate(paths, testConfig()); err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	// Both claude-plugin/.claude-plugin/ and rezbldr/.claude-plugin/ should exist.
	subdirs := []string{
		filepath.Dir(paths.MarketplaceManifest),
		filepath.Dir(paths.PluginManifest),
	}
	for _, d := range subdirs {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("expected %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}
}

func TestRemoveMarketplace(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	if err := Generate(paths, testConfig()); err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if _, err := os.Stat(paths.MarketplaceRoot); err != nil {
		t.Fatalf("MarketplaceRoot should exist before removal: %v", err)
	}
	if err := RemoveMarketplace(paths); err != nil {
		t.Fatalf("RemoveMarketplace error = %v", err)
	}
	if _, err := os.Stat(paths.MarketplaceRoot); !os.IsNotExist(err) {
		t.Errorf("expected MarketplaceRoot removed, stat err = %v", err)
	}
}

func TestRemoveMarketplace_NotPresentIsNoop(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	// Never created — RemoveMarketplace should still succeed.
	if err := RemoveMarketplace(paths); err != nil {
		t.Errorf("RemoveMarketplace when absent: %v", err)
	}
}

func TestResolveBinary_Explicit(t *testing.T) {
	got, err := resolveBinary("/opt/override/rezbldr")
	if err != nil {
		t.Fatalf("resolveBinary error = %v", err)
	}
	if got != "/opt/override/rezbldr" {
		t.Errorf("got %q, want explicit path", got)
	}
}

func TestResolveBinary_LookPath(t *testing.T) {
	// Plant a fake rezbldr in a tempdir and set PATH to just that dir.
	dir := t.TempDir()
	fake := filepath.Join(dir, PluginName)
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("writing fake: %v", err)
	}
	t.Setenv("PATH", dir)

	got, err := resolveBinary("")
	if err != nil {
		t.Fatalf("resolveBinary error = %v", err)
	}
	// exec.LookPath may return the bare name or a relative path; we only care
	// that the basename matches.
	if filepath.Base(got) != PluginName {
		t.Errorf("got %q, want basename %q", got, PluginName)
	}
}

func TestResolveBinary_FallbackToExecutable(t *testing.T) {
	// Point PATH at an empty dir so LookPath fails; resolveBinary falls back
	// to os.Executable (the go test binary itself).
	t.Setenv("PATH", t.TempDir())
	got, err := resolveBinary("")
	if err != nil {
		t.Fatalf("resolveBinary error = %v", err)
	}
	if got == "" {
		t.Error("got empty, want os.Executable fallback path")
	}
}

func TestWriteJSON_UnwritableParent(t *testing.T) {
	// Make parent directory a regular file so MkdirAll fails.
	dir := t.TempDir()
	blockingFile := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blockingFile, []byte("block"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	target := filepath.Join(blockingFile, "child", "file.json")
	if err := writeJSON(target, map[string]string{"k": "v"}); err == nil {
		t.Error("expected writeJSON to fail when parent path is a regular file")
	}
}

func TestGenerate_PropagatesWriteErrors(t *testing.T) {
	// Place a regular file at the location where MarketplaceManifest's
	// parent directory should be created — MkdirAll will fail.
	home := t.TempDir()
	paths := FromHome(home)
	blocker := filepath.Dir(paths.MarketplaceRoot)
	if err := os.MkdirAll(blocker, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Replace MarketplaceRoot's future location with a file.
	if err := os.WriteFile(paths.MarketplaceRoot, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := Generate(paths, testConfig()); err == nil {
		t.Error("expected Generate to fail when marketplace dir cannot be created")
	}
}
