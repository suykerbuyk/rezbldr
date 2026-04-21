// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstall_FullInstall(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}

	if err := Install(paths, cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Marketplace files.
	if !hasMarketplaceFiles(paths) {
		t.Error("marketplace files not fully created")
	}

	// Settings entries.
	has, err := HasSettingsEntries(paths)
	if err != nil || !has {
		t.Errorf("HasSettingsEntries = (%v, %v), want (true, nil)", has, err)
	}

	// Cache.
	if !HasCacheInstalled(paths, "0.2.0") {
		t.Error("cache not installed")
	}

	// Registries.
	has, err = HasMarketplaceRegistered(paths)
	if err != nil || !has {
		t.Errorf("HasMarketplaceRegistered = (%v, %v)", has, err)
	}
	has, err = HasInstalledPluginRegistered(paths)
	if err != nil || !has {
		t.Errorf("HasInstalledPluginRegistered = (%v, %v)", has, err)
	}
}

func TestInstall_RemovesStaleLegacyEntry(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)

	// Pre-seed a stale iteration-12 entry.
	writeSettingsForTest(t, paths.Settings, map[string]any{
		settingsKeyMcpServers: map[string]any{
			PluginName: map[string]any{"command": "/stale/path", "args": []any{"serve"}},
		},
	})

	cfg := Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}
	if err := Install(paths, cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}

	has, err := HasLegacyMcpServer(paths)
	if err != nil {
		t.Fatalf("HasLegacyMcpServer: %v", err)
	}
	if has {
		t.Error("stale mcpServers.rezbldr should have been removed by Install")
	}
}

func TestInstall_Idempotent(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}

	if err := Install(paths, cfg); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := Install(paths, cfg); err != nil {
		t.Fatalf("second: %v", err)
	}

	// Still healthy after repeated install.
	if !HealthCheck(paths).Healthy() {
		t.Error("health check failed after second install")
	}
}

func TestUninstall_RemovesEverything(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}
	if err := Install(paths, cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if err := Uninstall(paths); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if hasMarketplaceFiles(paths) {
		t.Error("marketplace files still present")
	}
	if has, _ := HasSettingsEntries(paths); has {
		t.Error("settings entries still present")
	}
	if HasCacheInstalled(paths, "0.2.0") {
		t.Error("cache still present")
	}
	if has, _ := HasMarketplaceRegistered(paths); has {
		t.Error("marketplace still in known_marketplaces.json")
	}
	if has, _ := HasInstalledPluginRegistered(paths); has {
		t.Error("plugin still in installed_plugins.json")
	}
}

func TestUninstall_AlsoRemovesStaleLegacyEntry(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	writeSettingsForTest(t, paths.Settings, map[string]any{
		settingsKeyMcpServers: map[string]any{
			PluginName: map[string]any{"command": "/stale"},
		},
	})
	if err := Uninstall(paths); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if has, _ := HasLegacyMcpServer(paths); has {
		t.Error("legacy entry still present after Uninstall")
	}
}

func TestUninstall_NothingInstalledIsNoop(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	if err := Uninstall(paths); err != nil {
		t.Errorf("Uninstall on empty state: %v", err)
	}
}

func TestHealthCheck_FreshlyInstalled(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	cfg := Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}
	if err := Install(paths, cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}
	s := HealthCheck(paths)
	if !s.Healthy() {
		t.Errorf("not healthy after install: %+v", s)
	}
}

func TestHealthCheck_UninstalledReportsAllFalse(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	s := HealthCheck(paths)
	if s.Healthy() {
		t.Error("empty state reports healthy")
	}
	if s.MarketplaceFiles || s.SettingsEntries || s.CacheInstalled ||
		s.MarketplaceInReg || s.InstalledPluginInReg || s.LegacyMcpServer {
		t.Errorf("expected all flags false, got %+v", s)
	}
}

func TestHealthCheck_DetectsLegacyEntry(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	writeSettingsForTest(t, paths.Settings, map[string]any{
		settingsKeyMcpServers: map[string]any{
			PluginName: map[string]any{"command": "/stale"},
		},
	})
	s := HealthCheck(paths)
	if !s.LegacyMcpServer {
		t.Error("HealthCheck did not detect legacy mcpServers entry")
	}
	if s.Healthy() {
		t.Error("Healthy() should be false when legacy entry present")
	}
}

func TestHealthCheck_PropagatesReadError(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	// Create a settings file that is valid JSON for some calls but not for others.
	// We simulate an error by making settings.json invalid JSON.
	_ = os.MkdirAll(filepath.Dir(paths.Settings), 0o755)
	_ = os.WriteFile(paths.Settings, []byte("{invalid"), 0o644)

	s := HealthCheck(paths)
	if s.FirstError == nil {
		t.Error("expected FirstError to be populated on invalid JSON")
	}
	if s.Healthy() {
		t.Error("Healthy() should be false when FirstError set")
	}
}

func TestHealthCheck_PartialInstall(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	// Only generate marketplace files; skip everything else.
	if err := Generate(paths, Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	s := HealthCheck(paths)
	if !s.MarketplaceFiles {
		t.Error("MarketplaceFiles should be true")
	}
	if s.SettingsEntries || s.CacheInstalled || s.MarketplaceInReg || s.InstalledPluginInReg {
		t.Errorf("other flags should be false: %+v", s)
	}
	if s.Healthy() {
		t.Error("partial install should not report healthy")
	}
}

func TestHealthCheck_NonFatalReadErrorsStopAtFirst(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	// Make two registry files invalid to verify only one error is surfaced.
	_ = os.MkdirAll(filepath.Dir(paths.KnownMarketplaces), 0o755)
	_ = os.WriteFile(paths.KnownMarketplaces, []byte("{broken"), 0o600)
	// Invalid installed_plugins.json too.
	_ = os.WriteFile(paths.InstalledPlugins, []byte("{broken"), 0o600)

	s := HealthCheck(paths)
	if s.FirstError == nil {
		t.Error("expected FirstError to be set when registry files are invalid")
	}
}

// Sanity test that HealthCheck sees a realistic multi-plugin install.
func TestHealthCheck_SiblingPluginsIgnored(t *testing.T) {
	home := t.TempDir()
	paths := FromHome(home)
	// Install ourselves.
	cfg := Config{Version: "0.2.0", BinaryPath: "/bin/rezbldr"}
	if err := Install(paths, cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}
	// Add a sibling to both registries.
	m, _ := readRegistryFile(paths.KnownMarketplaces)
	m["other-local"] = map[string]any{
		"source":          map[string]any{"source": "directory", "path": "/x"},
		"installLocation": "/x",
		"lastUpdated":     "2026-01-01T00:00:00Z",
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(paths.KnownMarketplaces, append(data, '\n'), 0o600)

	doc, _ := readInstalledPluginsDoc(paths.InstalledPlugins)
	doc.Plugins["other@other-local"] = []installedPluginVersionV2{{Scope: "user", Version: "1.0", InstallPath: "/y", InstalledAt: "2026-01-01T00:00:00Z", LastUpdated: "2026-01-01T00:00:00Z"}}
	data, _ = json.MarshalIndent(doc, "", "  ")
	_ = os.WriteFile(paths.InstalledPlugins, append(data, '\n'), 0o600)

	s := HealthCheck(paths)
	if !s.Healthy() {
		t.Errorf("sibling plugins should not affect Healthy; got %+v", s)
	}
}
