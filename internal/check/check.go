// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package check

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Result describes the outcome of a single check.
type Result struct {
	Name   string // e.g. "go"
	Status string // "ok", "warn", "fail"
	Detail string // human-readable detail
}

// Run performs all environment and vault checks and returns the results.
func Run(vaultPath string) []Result {
	var results []Result

	results = append(results, checkGo())
	results = append(results, checkPandoc())
	results = append(results, checkGit())
	results = append(results, checkVaultPath(vaultPath))
	results = append(results, checkVaultStructure(vaultPath))
	results = append(results, checkContactFile(vaultPath))
	results = append(results, checkGlobalRegistration())
	results = append(results, checkLegacyRegistrations())

	return results
}

func checkGo() Result {
	return Result{
		Name:   "go",
		Status: "ok",
		Detail: runtime.Version(),
	}
}

func checkPandoc() Result {
	out, err := exec.Command("pandoc", "--version").Output()
	if err != nil {
		return Result{
			Name:   "pandoc",
			Status: "warn",
			Detail: "not found (export will not work)",
		}
	}
	firstLine := strings.SplitN(string(out), "\n", 2)[0]
	return Result{
		Name:   "pandoc",
		Status: "ok",
		Detail: firstLine,
	}
}

func checkGit() Result {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return Result{
			Name:   "git",
			Status: "fail",
			Detail: "not found",
		}
	}
	firstLine := strings.SplitN(string(out), "\n", 2)[0]
	return Result{
		Name:   "git",
		Status: "ok",
		Detail: firstLine,
	}
}

func checkVaultPath(vaultPath string) Result {
	info, err := os.Stat(vaultPath)
	if err != nil || !info.IsDir() {
		return Result{
			Name:   "vault",
			Status: "fail",
			Detail: fmt.Sprintf("directory not found: %s", vaultPath),
		}
	}
	return Result{
		Name:   "vault",
		Status: "ok",
		Detail: vaultPath,
	}
}

// requiredDirs are the subdirectories that must exist inside the vault.
var requiredDirs = []string{"profile", "jobs/target", "resumes"}

func checkVaultStructure(vaultPath string) Result {
	var missing []string
	for _, sub := range requiredDirs {
		p := filepath.Join(vaultPath, sub)
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			missing = append(missing, sub)
		}
	}
	if len(missing) > 0 {
		return Result{
			Name:   "vault-structure",
			Status: "fail",
			Detail: fmt.Sprintf("missing directories: %s", strings.Join(missing, ", ")),
		}
	}
	return Result{
		Name:   "vault-structure",
		Status: "ok",
		Detail: fmt.Sprintf("found %s", strings.Join(requiredDirs, ", ")),
	}
}

func checkContactFile(vaultPath string) Result {
	p := filepath.Join(vaultPath, "profile", "contact.md")
	if _, err := os.Stat(p); err != nil {
		return Result{
			Name:   "contact",
			Status: "fail",
			Detail: "profile/contact.md not found (required for export)",
		}
	}
	return Result{
		Name:   "contact",
		Status: "ok",
		Detail: "profile/contact.md",
	}
}

func checkGlobalRegistration() Result {
	home, err := os.UserHomeDir()
	if err != nil {
		return Result{
			Name:   "mcp-global",
			Status: "warn",
			Detail: "cannot determine home directory",
		}
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	return CheckGlobalConfig(settingsPath)
}

// CheckGlobalConfig checks whether rezbldr is registered globally in the
// given Claude Code settings file under mcpServers. Exported for testing.
func CheckGlobalConfig(settingsPath string) Result {
	found, binaryPath := findGlobalMCPServer(settingsPath)
	if !found {
		return Result{
			Name:   "mcp-global",
			Status: "warn",
			Detail: "rezbldr not found in global settings (run rezbldr setup)",
		}
	}

	if binaryPath != "" {
		if _, err := os.Stat(binaryPath); err != nil {
			return Result{
				Name:   "mcp-global",
				Status: "warn",
				Detail: fmt.Sprintf("registered globally but binary not found at %s (run make install)", binaryPath),
			}
		}
	}
	return Result{
		Name:   "mcp-global",
		Status: "ok",
		Detail: fmt.Sprintf("rezbldr registered in %s", filepath.Base(settingsPath)),
	}
}

func checkLegacyRegistrations() Result {
	home, err := os.UserHomeDir()
	if err != nil {
		return Result{
			Name:   "mcp-legacy",
			Status: "ok",
			Detail: "no legacy entries (cannot determine home)",
		}
	}

	var stale []string

	// Check ~/.claude.json projects for any rezbldr entries.
	claudeJsonPath := filepath.Join(home, ".claude.json")
	if paths := findProjectScopedEntries(claudeJsonPath); len(paths) > 0 {
		for _, p := range paths {
			stale = append(stale, fmt.Sprintf(".claude.json project %s", p))
		}
	}

	// Check ~/.claude/.mcp.json.
	mcpJsonPath := filepath.Join(home, ".claude", ".mcp.json")
	if found, _ := findGlobalMCPServer(mcpJsonPath); found {
		stale = append(stale, ".claude/.mcp.json")
	}

	if len(stale) > 0 {
		return Result{
			Name:   "mcp-legacy",
			Status: "warn",
			Detail: fmt.Sprintf("stale entries in %s (run rezbldr setup to migrate)", strings.Join(stale, ", ")),
		}
	}
	return Result{
		Name:   "mcp-legacy",
		Status: "ok",
		Detail: "no stale entries",
	}
}

// findGlobalMCPServer looks for rezbldr in a settings file under
// mcpServers.rezbldr (top-level, not project-scoped).
func findGlobalMCPServer(path string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, ""
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		return false, ""
	}

	mcpRaw, ok := config["mcpServers"]
	if !ok {
		return false, ""
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(mcpRaw, &servers); err != nil {
		return false, ""
	}

	raw, ok := servers["rezbldr"]
	if !ok {
		return false, ""
	}

	var stanza struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(raw, &stanza); err == nil && stanza.Command != "" {
		return true, stanza.Command
	}
	return true, ""
}

// findProjectScopedEntries returns project paths in ~/.claude.json that
// still have rezbldr in their mcpServers.
func findProjectScopedEntries(claudeJsonPath string) []string {
	data, err := os.ReadFile(claudeJsonPath)
	if err != nil {
		return nil
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	projectsRaw, ok := config["projects"]
	if !ok {
		return nil
	}

	var projects map[string]json.RawMessage
	if err := json.Unmarshal(projectsRaw, &projects); err != nil {
		return nil
	}

	var found []string
	for projectPath, projectRaw := range projects {
		var project map[string]json.RawMessage
		if err := json.Unmarshal(projectRaw, &project); err != nil {
			continue
		}
		mcpRaw, ok := project["mcpServers"]
		if !ok {
			continue
		}
		var servers map[string]json.RawMessage
		if err := json.Unmarshal(mcpRaw, &servers); err != nil {
			continue
		}
		if _, ok := servers["rezbldr"]; ok {
			found = append(found, projectPath)
		}
	}
	return found
}
