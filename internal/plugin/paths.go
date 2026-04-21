// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package plugin installs rezbldr as a Claude Code plugin. Claude Code bug
// #2682 causes MCP servers registered via mcpServers in settings.json or
// ~/.claude.json to fail tool registration; plugin-bundled servers use a
// separate, working code path. This package generates the marketplace +
// plugin manifest files, updates ~/.claude/settings.json with the expected
// extraKnownMarketplaces and enabledPlugins entries, and belt-and-suspenders
// injects the same data into Claude Code's internal cache and registry
// files at ~/.claude/plugins/.
package plugin

import (
	"os"
	"path/filepath"
)

// Naming constants baked into the plugin identity. Changing these after
// release breaks existing installations; treat as stable.
const (
	MarketplaceName = "rezbldr-local"
	PluginName      = "rezbldr"
	PluginKey       = PluginName + "@" + MarketplaceName
)

// Paths bundles every filesystem location touched by the plugin installer.
// Construct with FromHome for test-isolated paths or Default for the running
// user's real home directory.
type Paths struct {
	Home string

	// Marketplace source tree under ~/.local/share/rezbldr/claude-plugin/.
	MarketplaceRoot     string
	MarketplaceManifest string
	PluginRoot          string
	PluginManifest      string
	McpJson             string

	// Claude Code state under ~/.claude/.
	ClaudeDir         string
	Settings          string
	PluginsDir        string
	KnownMarketplaces string
	InstalledPlugins  string
}

// FromHome returns a Paths instance rooted at the supplied home directory.
// The marketplace tree is placed under `<home>/.local/share/rezbldr/claude-plugin`
// (XDG default; callers that honor XDG_DATA_HOME should resolve it before
// calling and pass the resulting path via FromDataHome).
func FromHome(home string) Paths {
	return FromDataHome(home, filepath.Join(home, ".local", "share"))
}

// FromDataHome returns a Paths instance with an explicit XDG data-home
// location, enabling XDG_DATA_HOME support and hermetic tests.
func FromDataHome(home, dataHome string) Paths {
	marketplaceRoot := filepath.Join(dataHome, "rezbldr", "claude-plugin")
	pluginRoot := filepath.Join(marketplaceRoot, PluginName)
	claudeDir := filepath.Join(home, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")

	return Paths{
		Home:                home,
		MarketplaceRoot:     marketplaceRoot,
		MarketplaceManifest: filepath.Join(marketplaceRoot, ".claude-plugin", "marketplace.json"),
		PluginRoot:          pluginRoot,
		PluginManifest:      filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"),
		McpJson:             filepath.Join(pluginRoot, ".mcp.json"),
		ClaudeDir:           claudeDir,
		Settings:            filepath.Join(claudeDir, "settings.json"),
		PluginsDir:          pluginsDir,
		KnownMarketplaces:   filepath.Join(pluginsDir, "known_marketplaces.json"),
		InstalledPlugins:    filepath.Join(pluginsDir, "installed_plugins.json"),
	}
}

// Default returns Paths for the current user's home directory, honoring
// XDG_DATA_HOME if set.
func Default() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	return FromDataHome(home, dataHome), nil
}

// CacheVersionDir returns the version-scoped cache directory Claude Code
// uses to hold a resolved plugin:
// `<home>/.claude/plugins/cache/rezbldr-local/rezbldr/<version>`.
func (p Paths) CacheVersionDir(version string) string {
	return filepath.Join(p.PluginsDir, "cache", MarketplaceName, PluginName, version)
}

// CachePluginManifest returns the cache-side plugin.json path for a version.
func (p Paths) CachePluginManifest(version string) string {
	return filepath.Join(p.CacheVersionDir(version), ".claude-plugin", "plugin.json")
}

// CacheMcpJson returns the cache-side .mcp.json path for a version.
func (p Paths) CacheMcpJson(version string) string {
	return filepath.Join(p.CacheVersionDir(version), ".mcp.json")
}

// pathExists reports whether the file or directory at path is accessible.
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
