// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package install handles the binary-copy step of rezbldr installation and
// cleanup of legacy registration entries left by pre-plugin iterations of
// the installer. Current MCP registration lives in the plugin package.
package install

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// legacyMcpJsonFile is the legacy per-user MCP config file at ~/.claude/.mcp.json
// where earlier rezbldr versions attempted to register themselves.
const legacyMcpJsonFile = ".mcp.json"

// serverKey is the rezbldr entry name inside any legacy mcpServers map.
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

// MigrateProjectScoped scans all projects in ~/.claude.json and removes any
// rezbldr MCP server entries left over from iteration-11 project-scoped
// registration. Returns the list of project paths that were cleaned.
func MigrateProjectScoped(claudeJsonPath string) ([]string, error) {
	config, existed, err := readSettingsFile(claudeJsonPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", claudeJsonPath, err)
	}
	if !existed {
		return nil, nil
	}

	projects, ok := config["projects"].(map[string]any)
	if !ok {
		return nil, nil
	}

	var cleaned []string
	modified := false

	for projectPath, projectVal := range projects {
		project, ok := projectVal.(map[string]any)
		if !ok {
			continue
		}
		mcpServers, ok := project["mcpServers"].(map[string]any)
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

// CleanupLegacyMcpJson removes rezbldr from ~/.claude/.mcp.json if present —
// a location some early iterations wrote to before the plugin mechanism was
// adopted.
func CleanupLegacyMcpJson(claudeDir string) error {
	path := filepath.Join(claudeDir, legacyMcpJsonFile)
	config, existed, err := readSettingsFile(path)
	if err != nil || !existed {
		return nil
	}

	mcpServers, ok := config["mcpServers"].(map[string]any)
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

// readSettingsFile reads and parses a JSON settings file.
// Returns the parsed map, whether the file existed, and any error.
// If the file does not exist, returns an empty map with existed=false.
func readSettingsFile(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]any), false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, true, fmt.Errorf("parsing JSON in %s: %w", path, err)
	}
	return settings, true, nil
}

// writeSettingsFile writes a settings map as pretty-printed JSON with a trailing newline.
func writeSettingsFile(path string, settings map[string]any) error {
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
