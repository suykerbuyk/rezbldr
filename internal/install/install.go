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
// servers are registered under projects[projectPath].mcpServers.
const configFile = ".claude.json"

// legacySettingsFile is the old location we wrote to before discovering that
// Claude Code reads MCP servers from ~/.claude.json, not ~/.claude/settings.json.
const legacySettingsFile = "settings.json"
const legacyFallbackFile = "settings.local.json"
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

// Install copies the running binary to binaryPath and registers the MCP
// server stanza in Claude Code's user config.
// configPath is the full path to ~/.claude.json.
// projectDir is the absolute path to the project directory.
// vaultPath is the vault path to configure (optional — omit from args if empty).
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

// Uninstall removes the rezbldr MCP server stanza from Claude Code's user
// config and optionally removes the installed binary.
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

	// Clean up legacy locations (~/.claude/settings.json, settings.local.json).
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
