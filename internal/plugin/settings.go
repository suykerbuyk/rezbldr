// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// settingsKeyExtraKnownMarketplaces and settingsKeyEnabledPlugins are the
// top-level keys Claude Code reads from ~/.claude/settings.json when loading
// plugins from an external marketplace directory.
const (
	settingsKeyExtraKnownMarketplaces = "extraKnownMarketplaces"
	settingsKeyEnabledPlugins         = "enabledPlugins"
	settingsKeyMcpServers             = "mcpServers"
)

// AddSettingsEntries adds the rezbldr marketplace and plugin entries to
// ~/.claude/settings.json, preserving any existing keys and sibling plugins.
// If the settings file does not exist, it is created with 0o644.
func AddSettingsEntries(paths Paths) error {
	settings, err := readSettings(paths.Settings)
	if err != nil {
		return fmt.Errorf("reading %s: %w", paths.Settings, err)
	}

	markets := getStringMap(settings, settingsKeyExtraKnownMarketplaces)
	markets[MarketplaceName] = map[string]any{
		"source": map[string]any{
			"path":   paths.MarketplaceRoot,
			"source": "directory",
		},
	}
	settings[settingsKeyExtraKnownMarketplaces] = markets

	plugins := getStringMap(settings, settingsKeyEnabledPlugins)
	plugins[PluginKey] = true
	settings[settingsKeyEnabledPlugins] = plugins

	return writeSettings(paths.Settings, settings)
}

// RemoveSettingsEntries removes the rezbldr marketplace and plugin entries
// from ~/.claude/settings.json. Sibling entries (other marketplaces, other
// plugins) are preserved. If the file does not exist or contains no rezbldr
// entries, RemoveSettingsEntries is a no-op.
func RemoveSettingsEntries(paths Paths) error {
	if _, err := os.Stat(paths.Settings); os.IsNotExist(err) {
		return nil
	}
	settings, err := readSettings(paths.Settings)
	if err != nil {
		return fmt.Errorf("reading %s: %w", paths.Settings, err)
	}

	modified := false
	if markets, ok := settings[settingsKeyExtraKnownMarketplaces].(map[string]any); ok {
		if _, found := markets[MarketplaceName]; found {
			delete(markets, MarketplaceName)
			if len(markets) == 0 {
				delete(settings, settingsKeyExtraKnownMarketplaces)
			} else {
				settings[settingsKeyExtraKnownMarketplaces] = markets
			}
			modified = true
		}
	}
	if plugins, ok := settings[settingsKeyEnabledPlugins].(map[string]any); ok {
		if _, found := plugins[PluginKey]; found {
			delete(plugins, PluginKey)
			if len(plugins) == 0 {
				delete(settings, settingsKeyEnabledPlugins)
			} else {
				settings[settingsKeyEnabledPlugins] = plugins
			}
			modified = true
		}
	}

	if !modified {
		return nil
	}
	return writeSettings(paths.Settings, settings)
}

// RemoveLegacyMcpServer removes a stale `mcpServers.rezbldr` entry from
// ~/.claude/settings.json — the iteration-12 registration location that
// triggered Claude Code bug #2682. Returns whether anything was removed.
func RemoveLegacyMcpServer(paths Paths) (bool, error) {
	if _, err := os.Stat(paths.Settings); os.IsNotExist(err) {
		return false, nil
	}
	settings, err := readSettings(paths.Settings)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", paths.Settings, err)
	}

	servers, ok := settings[settingsKeyMcpServers].(map[string]any)
	if !ok {
		return false, nil
	}
	if _, found := servers[PluginName]; !found {
		return false, nil
	}

	delete(servers, PluginName)
	if len(servers) == 0 {
		delete(settings, settingsKeyMcpServers)
	} else {
		settings[settingsKeyMcpServers] = servers
	}
	if err := writeSettings(paths.Settings, settings); err != nil {
		return false, err
	}
	return true, nil
}

// HasLegacyMcpServer reports whether settings.json contains a stale
// mcpServers.rezbldr entry. Used by `rezbldr check`.
func HasLegacyMcpServer(paths Paths) (bool, error) {
	if _, err := os.Stat(paths.Settings); os.IsNotExist(err) {
		return false, nil
	}
	settings, err := readSettings(paths.Settings)
	if err != nil {
		return false, err
	}
	servers, ok := settings[settingsKeyMcpServers].(map[string]any)
	if !ok {
		return false, nil
	}
	_, found := servers[PluginName]
	return found, nil
}

// HasSettingsEntries reports whether both the marketplace and plugin settings
// entries are present and point to the expected locations. Used by
// `rezbldr check`.
func HasSettingsEntries(paths Paths) (bool, error) {
	if _, err := os.Stat(paths.Settings); os.IsNotExist(err) {
		return false, nil
	}
	settings, err := readSettings(paths.Settings)
	if err != nil {
		return false, err
	}

	markets, ok := settings[settingsKeyExtraKnownMarketplaces].(map[string]any)
	if !ok {
		return false, nil
	}
	entry, ok := markets[MarketplaceName].(map[string]any)
	if !ok {
		return false, nil
	}
	source, ok := entry["source"].(map[string]any)
	if !ok {
		return false, nil
	}
	if source["path"] != paths.MarketplaceRoot {
		return false, nil
	}
	if source["source"] != "directory" {
		return false, nil
	}

	plugins, ok := settings[settingsKeyEnabledPlugins].(map[string]any)
	if !ok {
		return false, nil
	}
	enabled, _ := plugins[PluginKey].(bool)
	return enabled, nil
}

// getStringMap returns settings[key] as a map[string]any, creating a new
// one if the key is missing or not a map.
func getStringMap(settings map[string]any, key string) map[string]any {
	if m, ok := settings[key].(map[string]any); ok {
		return m
	}
	return make(map[string]any)
}

// readSettings parses the JSON file at path into a map. If the file does not
// exist, returns an empty map and nil error — callers that need to
// distinguish "missing" from "empty" should stat separately.
func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if out == nil {
		out = make(map[string]any)
	}
	return out, nil
}

func writeSettings(path string, settings map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
