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

	cwd, err := os.Getwd()
	if err != nil {
		return Result{
			Name:   "claude-settings",
			Status: "warn",
			Detail: "cannot determine working directory",
		}
	}

	configPath := filepath.Join(home, ".claude.json")
	return CheckClaudeConfig(configPath, cwd)
}

// CheckClaudeConfig checks whether rezbldr is registered in the given
// Claude Code config file for the given project directory. Exported for
// testing with controlled paths.
func CheckClaudeConfig(configPath, projectDir string) Result {
	if found, binaryPath := checkConfigFile(configPath, projectDir); found {
		if binaryPath != "" {
			if _, err := os.Stat(binaryPath); err != nil {
				return Result{
					Name:   "claude-settings",
					Status: "warn",
					Detail: fmt.Sprintf("registered in %s but binary not found at %s (run rezbldr install)", filepath.Base(configPath), binaryPath),
				}
			}
		}
		return Result{
			Name:   "claude-settings",
			Status: "ok",
			Detail: fmt.Sprintf("rezbldr registered in %s", filepath.Base(configPath)),
		}
	}

	return Result{
		Name:   "claude-settings",
		Status: "warn",
		Detail: "rezbldr not found in Claude Code MCP settings (run rezbldr install)",
	}
}

// checkConfigFile looks for rezbldr in ~/.claude.json under
// projects[projectDir].mcpServers.rezbldr. Returns whether found and the
// binary command path if extractable.
func checkConfigFile(path, projectDir string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, ""
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		return false, ""
	}

	projectsRaw, ok := config["projects"]
	if !ok {
		return false, ""
	}

	var projects map[string]json.RawMessage
	if err := json.Unmarshal(projectsRaw, &projects); err != nil {
		return false, ""
	}

	projectRaw, ok := projects[projectDir]
	if !ok {
		return false, ""
	}

	var project map[string]json.RawMessage
	if err := json.Unmarshal(projectRaw, &project); err != nil {
		return false, ""
	}

	mcpRaw, ok := project["mcpServers"]
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
