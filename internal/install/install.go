// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package install manages registration and removal of the rezbldr MCP server
// in Claude Code settings files.
package install

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// configFile is the Claude Code user config file (~/.claude.json) where MCP
// servers were previously registered under projects[projectPath].mcpServers.
// Now considered a legacy location — setup uses global settings.json instead.
const configFile = ".claude.json"

// legacySettingsFile and legacyFallbackFile are old locations we wrote to
// before discovering the correct registration targets. Note that
// legacySettingsFile is also the current global registration target
// (~/.claude/settings.json mcpServers) — the same file, different key paths.
const legacySettingsFile = "settings.json"
const legacyFallbackFile = "settings.local.json"

// legacyMcpJsonFile is the legacy per-directory MCP config file.
const legacyMcpJsonFile = ".mcp.json"
const serverKey = "rezbldr"

// CopyBinary copies the currently running executable to dstPath, creating
// parent directories as needed. If the source and destination are the same
// file, it is a no-op.
func CopyBinary(dstPath string) error {
	srcPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determining current executable: %w", err)
	}
	srcPath, err = filepath.EvalSymlinks(srcPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	dstResolved, _ := filepath.EvalSymlinks(dstPath)
	if srcPath == dstResolved {
		fmt.Printf("Binary already at %s\n", dstPath)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", filepath.Dir(dstPath), err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("opening source binary: %w", err)
	}
	defer src.Close()

	// Write to a temp file in the same directory, then rename for atomicity.
	tmp, err := os.CreateTemp(filepath.Dir(dstPath), ".rezbldr-install-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("copying binary: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpPath, dstPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("installing binary to %s: %w", dstPath, err)
	}

	fmt.Printf("Installed binary to %s\n", dstPath)
	return nil
}

// Setup copies the binary and registers rezbldr globally in Claude Code's
// settings.json, then migrates any project-scoped and legacy entries.
// settingsPath is the full path to ~/.claude/settings.json.
// claudeJsonPath is the full path to ~/.claude.json.
// claudeDir is ~/.claude (for legacy .mcp.json cleanup).
// vaultPath is the vault path to configure (optional — omit from args if empty).
func Setup(binaryPath, settingsPath, claudeJsonPath, claudeDir, vaultPath string) error {
	if err := CopyBinary(binaryPath); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}

	if err := RegisterGlobal(binaryPath, settingsPath, vaultPath); err != nil {
		return fmt.Errorf("global registration: %w", err)
	}

	cleaned, err := MigrateProjectScoped(claudeJsonPath)
	if err != nil {
		return fmt.Errorf("migrating project-scoped entries: %w", err)
	}
	for _, p := range cleaned {
		fmt.Printf("Migrated: removed rezbldr from project %s in %s\n", p, filepath.Base(claudeJsonPath))
	}

	if err := CleanupLegacyMcpJson(claudeDir); err != nil {
		return fmt.Errorf("cleaning up legacy .mcp.json: %w", err)
	}

	return nil
}

// RegisterGlobal adds or updates the rezbldr MCP server stanza in Claude Code's
// global settings file (~/.claude/settings.json) under mcpServers.
func RegisterGlobal(binaryPath, settingsPath, vaultPath string) error {
	settings, existed, err := readSettingsFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", settingsPath, err)
	}

	args := []interface{}{"serve"}
	if vaultPath != "" {
		args = append(args, "--vault", vaultPath)
	}

	// Match vibe-vault convention: no "type" or "env" keys.
	stanza := map[string]interface{}{
		"command": binaryPath,
		"args":    args,
	}

	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}
	mcpServers[serverKey] = stanza
	settings["mcpServers"] = mcpServers

	if err := writeSettingsFile(settingsPath, settings); err != nil {
		return err
	}

	if existed {
		fmt.Printf("Updated rezbldr in %s (global)\n", settingsPath)
	} else {
		fmt.Printf("Registered rezbldr in %s (global)\n", settingsPath)
	}
	return nil
}

// UnregisterGlobal removes the rezbldr MCP server stanza from the global
// settings file (~/.claude/settings.json).
func UnregisterGlobal(settingsPath string) error {
	settings, existed, err := readSettingsFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", settingsPath, err)
	}
	if !existed {
		return nil
	}

	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		return nil
	}
	if _, found := mcpServers[serverKey]; !found {
		return nil
	}

	delete(mcpServers, serverKey)
	if len(mcpServers) == 0 {
		delete(settings, "mcpServers")
	} else {
		settings["mcpServers"] = mcpServers
	}

	if err := writeSettingsFile(settingsPath, settings); err != nil {
		return err
	}
	fmt.Printf("Removed rezbldr from %s (global)\n", settingsPath)
	return nil
}

// MigrateProjectScoped scans all projects in ~/.claude.json and removes any
// rezbldr MCP server entries. Returns the list of project paths that were cleaned.
func MigrateProjectScoped(claudeJsonPath string) ([]string, error) {
	config, existed, err := readSettingsFile(claudeJsonPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", claudeJsonPath, err)
	}
	if !existed {
		return nil, nil
	}

	projects, ok := config["projects"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	var cleaned []string
	modified := false

	for projectPath, projectVal := range projects {
		project, ok := projectVal.(map[string]interface{})
		if !ok {
			continue
		}
		mcpServers, ok := project["mcpServers"].(map[string]interface{})
		if !ok {
			continue
		}
		if _, found := mcpServers[serverKey]; !found {
			continue
		}

		delete(mcpServers, serverKey)
		if len(mcpServers) == 0 {
			delete(project, "mcpServers")
		} else {
			project["mcpServers"] = mcpServers
		}
		projects[projectPath] = project
		cleaned = append(cleaned, projectPath)
		modified = true
	}

	if modified {
		config["projects"] = projects
		if err := writeSettingsFile(claudeJsonPath, config); err != nil {
			return cleaned, err
		}
	}
	return cleaned, nil
}

// CleanupLegacyMcpJson removes rezbldr from ~/.claude/.mcp.json if present.
func CleanupLegacyMcpJson(claudeDir string) error {
	path := filepath.Join(claudeDir, legacyMcpJsonFile)
	config, existed, err := readSettingsFile(path)
	if err != nil || !existed {
		return nil
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return nil
	}
	if _, found := mcpServers[serverKey]; !found {
		return nil
	}

	delete(mcpServers, serverKey)
	if len(mcpServers) == 0 {
		delete(config, "mcpServers")
	} else {
		config["mcpServers"] = mcpServers
	}

	if err := writeSettingsFile(path, config); err != nil {
		return err
	}
	fmt.Printf("Cleaned up legacy rezbldr entry from %s\n", path)
	return nil
}

// Install copies the running binary to binaryPath and registers the MCP
// server stanza in Claude Code's user config.
// Deprecated: Use Setup instead for global registration.
func Install(binaryPath, configPath, projectDir, vaultPath string) error {
	if err := CopyBinary(binaryPath); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}
	return Register(binaryPath, configPath, projectDir, vaultPath)
}

// Register adds or updates the rezbldr MCP server stanza in Claude Code's
// user config (~/.claude.json) under projects[projectDir].mcpServers.
func Register(binaryPath, configPath, projectDir, vaultPath string) error {
	config, existed, err := readSettingsFile(configPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	// Build the rezbldr stanza.
	args := []interface{}{"serve"}
	if vaultPath != "" {
		args = append(args, "--vault", vaultPath)
	}

	stanza := map[string]interface{}{
		"type":    "stdio",
		"command": binaryPath,
		"args":    args,
		"env":     map[string]interface{}{},
	}

	// Navigate to projects[projectDir].mcpServers.
	projects, ok := config["projects"].(map[string]interface{})
	if !ok {
		projects = make(map[string]interface{})
	}

	project, ok := projects[projectDir].(map[string]interface{})
	if !ok {
		project = make(map[string]interface{})
	}

	mcpServers, ok := project["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	mcpServers[serverKey] = stanza
	project["mcpServers"] = mcpServers
	projects[projectDir] = project
	config["projects"] = projects

	if err := writeSettingsFile(configPath, config); err != nil {
		return err
	}

	if existed {
		fmt.Printf("Updated rezbldr MCP server in %s [project: %s]\n", configPath, projectDir)
	} else {
		fmt.Printf("Installed rezbldr MCP server in %s [project: %s]\n", configPath, projectDir)
	}
	return nil
}

// Uninstall removes the rezbldr MCP server stanza from all known locations
// and optionally removes the installed binary.
// configPath is the full path to ~/.claude.json.
// projectDir is the absolute project path to unregister from.
// legacyDir is the legacy settings directory (~/.claude) for cleanup.
func Uninstall(configPath, projectDir, legacyDir, binaryPath string) error {
	// Remove the binary if it exists.
	if binaryPath != "" {
		if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing binary %s: %w", binaryPath, err)
		} else if err == nil {
			fmt.Printf("Removed binary %s\n", binaryPath)
		}
	}

	removed := false

	// Remove from ~/.claude.json projects[projectDir].mcpServers.
	if config, existed, err := readSettingsFile(configPath); err != nil {
		return fmt.Errorf("reading %s: %w", configPath, err)
	} else if existed {
		if projects, ok := config["projects"].(map[string]interface{}); ok {
			if project, ok := projects[projectDir].(map[string]interface{}); ok {
				if mcpServers, ok := project["mcpServers"].(map[string]interface{}); ok {
					if _, found := mcpServers[serverKey]; found {
						delete(mcpServers, serverKey)
						if len(mcpServers) == 0 {
							delete(project, "mcpServers")
						} else {
							project["mcpServers"] = mcpServers
						}
						projects[projectDir] = project
						config["projects"] = projects
						if err := writeSettingsFile(configPath, config); err != nil {
							return err
						}
						fmt.Printf("Removed rezbldr from %s [project: %s]\n", configPath, projectDir)
						removed = true
					}
				}
			}
		}
	}

	// Clean up legacy .mcp.json.
	if err := CleanupLegacyMcpJson(legacyDir); err != nil {
		return fmt.Errorf("cleaning legacy .mcp.json: %w", err)
	}

	// Clean up settings.json (also the global registration target) and settings.local.json.
	for _, name := range []string{legacySettingsFile, legacyFallbackFile} {
		path := filepath.Join(legacyDir, name)
		settings, existed, err := readSettingsFile(path)
		if err != nil || !existed {
			continue
		}
		mcpServers, ok := settings["mcpServers"].(map[string]interface{})
		if !ok {
			continue
		}
		if _, found := mcpServers[serverKey]; !found {
			continue
		}
		delete(mcpServers, serverKey)
		if len(mcpServers) == 0 {
			delete(settings, "mcpServers")
		} else {
			settings["mcpServers"] = mcpServers
		}
		if err := writeSettingsFile(path, settings); err != nil {
			return err
		}
		fmt.Printf("Cleaned up legacy rezbldr entry from %s\n", path)
		removed = true
	}

	if !removed {
		fmt.Println("rezbldr not found in settings")
	}
	return nil
}

// readSettingsFile reads and parses a JSON settings file.
// Returns the parsed map, whether the file existed, and any error.
// If the file does not exist, returns an empty map with existed=false.
func readSettingsFile(path string) (map[string]interface{}, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]interface{}), false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, true, fmt.Errorf("parsing JSON in %s: %w", path, err)
	}
	return settings, true, nil
}

// writeSettingsFile writes a settings map as pretty-printed JSON with a trailing newline.
func writeSettingsFile(path string, settings map[string]interface{}) error {
	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling JSON: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
