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

// settingsFile is the primary target for MCP registration. Claude Code
// reliably reads mcpServers from settings.json but not settings.local.json
// (upstream bug as of 2026-04). We write to settings.json and fall back to
// settings.local.json for uninstall/check compatibility.
const settingsFile = "settings.json"
const fallbackFile = "settings.local.json"
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
// server stanza in Claude Code settings.
// settingsDir is the directory containing settings files (typically ~/.claude).
// vaultPath is the vault path to configure (optional — omit from args if empty).
func Install(binaryPath, settingsDir, vaultPath string) error {
	if err := CopyBinary(binaryPath); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}
	return Register(binaryPath, settingsDir, vaultPath)
}

// Register adds or updates the rezbldr MCP server stanza in Claude Code
// settings without copying the binary. Used by Install after CopyBinary,
// and directly in tests.
func Register(binaryPath, settingsDir, vaultPath string) error {
	path := filepath.Join(settingsDir, settingsFile)

	settings, existed, err := readSettingsFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	// Build the rezbldr stanza.
	args := []interface{}{"serve"}
	if vaultPath != "" {
		args = append(args, "--vault", vaultPath)
	}

	stanza := map[string]interface{}{
		"command": binaryPath,
		"args":    args,
		"env":     map[string]interface{}{},
	}

	// Ensure mcpServers map exists.
	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	mcpServers[serverKey] = stanza
	settings["mcpServers"] = mcpServers

	if err := writeSettingsFile(path, settings); err != nil {
		return err
	}

	if existed {
		fmt.Printf("Updated rezbldr MCP server in %s\n", path)
	} else {
		fmt.Printf("Installed rezbldr MCP server in %s\n", path)
	}
	return nil
}

// Uninstall removes the rezbldr MCP server stanza from Claude Code settings
// and optionally removes the installed binary.
func Uninstall(settingsDir, binaryPath string) error {
	// Remove the binary if it exists.
	if binaryPath != "" {
		if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing binary %s: %w", binaryPath, err)
		} else if err == nil {
			fmt.Printf("Removed binary %s\n", binaryPath)
		}
	}
	// Check settings.local.json first, then settings.json.
	for _, name := range []string{settingsFile, fallbackFile} {
		path := filepath.Join(settingsDir, name)

		settings, existed, err := readSettingsFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		if !existed {
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

		// If mcpServers is now empty, remove the key entirely.
		if len(mcpServers) == 0 {
			delete(settings, "mcpServers")
		} else {
			settings["mcpServers"] = mcpServers
		}

		if err := writeSettingsFile(path, settings); err != nil {
			return err
		}

		fmt.Printf("Removed rezbldr from %s\n", path)
		return nil
	}

	fmt.Println("rezbldr not found in settings")
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
