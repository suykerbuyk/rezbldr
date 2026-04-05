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
	results = append(results, checkClaudeSettings())

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

func checkClaudeSettings() Result {
	home, err := os.UserHomeDir()
	if err != nil {
		return Result{
			Name:   "claude-settings",
			Status: "warn",
			Detail: "cannot determine home directory",
		}
	}

	// Check settings.local.json first, then settings.json.
	paths := []string{
		filepath.Join(home, ".claude", "settings.local.json"),
		filepath.Join(home, ".claude", "settings.json"),
	}

	for _, p := range paths {
		if found, file := checkSettingsFile(p); found {
			return Result{
				Name:   "claude-settings",
				Status: "ok",
				Detail: fmt.Sprintf("rezbldr registered in %s", filepath.Base(file)),
			}
		}
	}

	return Result{
		Name:   "claude-settings",
		Status: "warn",
		Detail: "rezbldr not found in Claude Code MCP settings",
	}
}

// checkSettingsFile returns true if the given JSON file contains an
// mcpServers.rezbldr key.
func checkSettingsFile(path string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, ""
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		return false, ""
	}

	mcpRaw, ok := settings["mcpServers"]
	if !ok {
		return false, ""
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(mcpRaw, &servers); err != nil {
		return false, ""
	}

	if _, ok := servers["rezbldr"]; ok {
		return true, path
	}
	return false, ""
}
