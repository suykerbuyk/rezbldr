// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package plugin

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestConstants(t *testing.T) {
	if MarketplaceName != "rezbldr-local" {
		t.Errorf("MarketplaceName = %q, want %q", MarketplaceName, "rezbldr-local")
	}
	if PluginName != "rezbldr" {
		t.Errorf("PluginName = %q, want %q", PluginName, "rezbldr")
	}
	if PluginKey != "rezbldr@rezbldr-local" {
		t.Errorf("PluginKey = %q, want %q", PluginKey, "rezbldr@rezbldr-local")
	}
}

func TestFromHome(t *testing.T) {
	home := "/tmp/fake-home"
	p := FromHome(home)

	wantDataHome := filepath.Join(home, ".local", "share", "rezbldr", "claude-plugin")
	if p.MarketplaceRoot != wantDataHome {
		t.Errorf("MarketplaceRoot = %q, want %q", p.MarketplaceRoot, wantDataHome)
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Home", p.Home, home},
		{"MarketplaceManifest", p.MarketplaceManifest,
			filepath.Join(home, ".local", "share", "rezbldr", "claude-plugin", ".claude-plugin", "marketplace.json")},
		{"PluginRoot", p.PluginRoot,
			filepath.Join(home, ".local", "share", "rezbldr", "claude-plugin", "rezbldr")},
		{"PluginManifest", p.PluginManifest,
			filepath.Join(home, ".local", "share", "rezbldr", "claude-plugin", "rezbldr", ".claude-plugin", "plugin.json")},
		{"McpJson", p.McpJson,
			filepath.Join(home, ".local", "share", "rezbldr", "claude-plugin", "rezbldr", ".mcp.json")},
		{"ClaudeDir", p.ClaudeDir, filepath.Join(home, ".claude")},
		{"Settings", p.Settings, filepath.Join(home, ".claude", "settings.json")},
		{"PluginsDir", p.PluginsDir, filepath.Join(home, ".claude", "plugins")},
		{"KnownMarketplaces", p.KnownMarketplaces,
			filepath.Join(home, ".claude", "plugins", "known_marketplaces.json")},
		{"InstalledPlugins", p.InstalledPlugins,
			filepath.Join(home, ".claude", "plugins", "installed_plugins.json")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestFromDataHome_ExplicitDataHome(t *testing.T) {
	home := "/tmp/fake-home"
	dataHome := "/custom/data/home"
	p := FromDataHome(home, dataHome)

	wantMarket := filepath.Join(dataHome, "rezbldr", "claude-plugin")
	if p.MarketplaceRoot != wantMarket {
		t.Errorf("MarketplaceRoot = %q, want %q", p.MarketplaceRoot, wantMarket)
	}
	// Claude Code state still rooted at home, not dataHome.
	wantClaude := filepath.Join(home, ".claude")
	if p.ClaudeDir != wantClaude {
		t.Errorf("ClaudeDir = %q, want %q", p.ClaudeDir, wantClaude)
	}
}

func TestDefault(t *testing.T) {
	// Default() uses os.UserHomeDir; we just verify it returns populated paths.
	p, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if p.Home == "" {
		t.Error("Default().Home is empty")
	}
	if p.Settings == "" || !strings.HasSuffix(p.Settings, "settings.json") {
		t.Errorf("Default().Settings = %q, want a path ending in settings.json", p.Settings)
	}
	if p.MarketplaceRoot == "" {
		t.Error("Default().MarketplaceRoot is empty")
	}
}

func TestDefault_HonorsXDG_DATA_HOME(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/xdg/data")
	p, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	wantRoot := filepath.Join("/xdg/data", "rezbldr", "claude-plugin")
	if p.MarketplaceRoot != wantRoot {
		t.Errorf("MarketplaceRoot = %q, want %q", p.MarketplaceRoot, wantRoot)
	}
}

func TestDefault_EmptyXDG_DATA_HOMEFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	p, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	// Should contain ~/.local/share somewhere in the path.
	if !strings.Contains(p.MarketplaceRoot, filepath.Join(".local", "share")) {
		t.Errorf("MarketplaceRoot = %q, expected to contain %q",
			p.MarketplaceRoot, filepath.Join(".local", "share"))
	}
}

func TestCacheVersionPaths(t *testing.T) {
	p := FromHome("/tmp/fake-home")
	version := "0.2.0"

	wantVerDir := filepath.Join("/tmp/fake-home", ".claude", "plugins", "cache",
		MarketplaceName, PluginName, version)
	if got := p.CacheVersionDir(version); got != wantVerDir {
		t.Errorf("CacheVersionDir(%q) = %q, want %q", version, got, wantVerDir)
	}

	wantManifest := filepath.Join(wantVerDir, ".claude-plugin", "plugin.json")
	if got := p.CachePluginManifest(version); got != wantManifest {
		t.Errorf("CachePluginManifest(%q) = %q, want %q", version, got, wantManifest)
	}

	wantMcp := filepath.Join(wantVerDir, ".mcp.json")
	if got := p.CacheMcpJson(version); got != wantMcp {
		t.Errorf("CacheMcpJson(%q) = %q, want %q", version, got, wantMcp)
	}
}

func TestCacheVersionPaths_DifferentVersions(t *testing.T) {
	p := FromHome("/tmp/fake-home")
	v1 := p.CacheVersionDir("0.1.0")
	v2 := p.CacheVersionDir("0.2.0")
	if v1 == v2 {
		t.Error("CacheVersionDir returned identical paths for different versions")
	}
}
