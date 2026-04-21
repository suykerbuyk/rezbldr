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

	"github.com/suykerbuyk/rezbldr/internal/plugin"
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
	results = append(results, checkPluginInstall()...)
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

// checkPluginInstall reports on the four layers of the plugin install:
// marketplace files, settings entries, cache, and registry files. Each is
// surfaced as a separate Result so `rezbldr check` gives actionable
// diagnostics when one layer is out of sync.
func checkPluginInstall() []Result {
	paths, err := plugin.Default()
	if err != nil {
		return []Result{{
			Name:   "mcp-plugin",
			Status: "warn",
			Detail: fmt.Sprintf("cannot determine home directory: %v", err),
		}}
	}
	return CheckPluginAt(paths)
}

// CheckPluginAt is CheckPluginInstall parameterized on plugin.Paths for
// testing against a fake HOME.
func CheckPluginAt(paths plugin.Paths) []Result {
	status := plugin.HealthCheck(paths)
	if status.FirstError != nil {
		return []Result{{
			Name:   "mcp-plugin",
			Status: "fail",
			Detail: fmt.Sprintf("reading plugin state: %v", status.FirstError),
		}}
	}

	var results []Result

	// Marketplace files.
	if status.MarketplaceFiles {
		results = append(results, Result{
			Name:   "mcp-plugin-files",
			Status: "ok",
			Detail: paths.MarketplaceRoot,
		})
	} else {
		results = append(results, Result{
			Name:   "mcp-plugin-files",
			Status: "fail",
			Detail: fmt.Sprintf("marketplace files missing under %s (run rezbldr setup)", paths.MarketplaceRoot),
		})
	}

	// Settings entries.
	if status.SettingsEntries {
		results = append(results, Result{
			Name:   "mcp-plugin-settings",
			Status: "ok",
			Detail: fmt.Sprintf("%s registered in %s", plugin.PluginKey, filepath.Base(paths.Settings)),
		})
	} else {
		results = append(results, Result{
			Name:   "mcp-plugin-settings",
			Status: "fail",
			Detail: fmt.Sprintf("extraKnownMarketplaces/enabledPlugins entries missing in %s (run rezbldr setup)", filepath.Base(paths.Settings)),
		})
	}

	// Cache + registries (combined for brevity — all three must be present
	// for Claude Code's plugin loader to succeed).
	cacheOk := status.CacheInstalled && status.MarketplaceInReg && status.InstalledPluginInReg
	if cacheOk {
		results = append(results, Result{
			Name:   "mcp-plugin-cache",
			Status: "ok",
			Detail: "cache and registry entries present",
		})
	} else {
		var missing []string
		if !status.CacheInstalled {
			missing = append(missing, "cache dir")
		}
		if !status.MarketplaceInReg {
			missing = append(missing, "known_marketplaces.json")
		}
		if !status.InstalledPluginInReg {
			missing = append(missing, "installed_plugins.json")
		}
		results = append(results, Result{
			Name:   "mcp-plugin-cache",
			Status: "fail",
			Detail: fmt.Sprintf("missing: %s (run rezbldr setup)", strings.Join(missing, ", ")),
		})
	}

	return results
}

// checkLegacyRegistrations warns about residual entries from pre-plugin
// iterations: ~/.claude/settings.json mcpServers.rezbldr (iteration 12 —
// bug #2682 registration), ~/.claude.json projects[*].mcpServers.rezbldr
// (iteration 11 — project-scoped), and ~/.claude/.mcp.json (early).
func checkLegacyRegistrations() Result {
	paths, err := plugin.Default()
	if err != nil {
		return Result{
			Name:   "mcp-legacy",
			Status: "ok",
			Detail: "no legacy entries (cannot determine home)",
		}
	}

	var stale []string

	if hasLegacy, _ := plugin.HasLegacyMcpServer(paths); hasLegacy {
		stale = append(stale, filepath.Base(paths.Settings)+" mcpServers.rezbldr")
	}

	// ~/.claude.json projects[*].mcpServers.rezbldr.
	home, _ := os.UserHomeDir()
	claudeJsonPath := filepath.Join(home, ".claude.json")
	if projects := findProjectScopedEntries(claudeJsonPath); len(projects) > 0 {
		for _, p := range projects {
			stale = append(stale, fmt.Sprintf(".claude.json project %s", p))
		}
	}

	// ~/.claude/.mcp.json.
	mcpJsonPath := filepath.Join(paths.ClaudeDir, ".mcp.json")
	if found, _ := findMcpJsonEntry(mcpJsonPath); found {
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
		if _, ok := servers[plugin.PluginName]; ok {
			found = append(found, projectPath)
		}
	}
	return found
}

// findMcpJsonEntry reports whether ~/.claude/.mcp.json contains a rezbldr
// entry (legacy iteration-57-era registration).
func findMcpJsonEntry(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		return false, err
	}
	serversRaw, ok := config["mcpServers"]
	if !ok {
		return false, nil
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(serversRaw, &servers); err != nil {
		return false, err
	}
	_, found := servers[plugin.PluginName]
	return found, nil
}
