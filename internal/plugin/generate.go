// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Config captures the inputs needed to generate rezbldr's plugin marketplace.
type Config struct {
	// Version is the plugin version (typically the rezbldr binary's version).
	Version string
	// BinaryPath is the absolute path to the rezbldr binary Claude Code will
	// launch. If empty, Generate resolves it via exec.LookPath("rezbldr"),
	// falling back to os.Executable().
	BinaryPath string
	// ExtraArgs are appended after "serve" in .mcp.json (e.g. ["--vault", path]).
	ExtraArgs []string
}

const (
	marketplaceSchemaURL = "https://anthropic.com/claude-code/marketplace.schema.json"
	marketplaceDesc      = "Local rezbldr plugin marketplace"
	pluginDesc           = "MCP server for deterministic resume pipeline operations"
	ownerName            = "rezbldr"
	ownerEmail           = "noreply@rezbldr.dev"
)

// Generate writes marketplace.json, plugin.json, and .mcp.json under the
// marketplace tree described by paths, using cfg for plugin-specific fields.
// Parent directories are created with 0o755; files written with 0o644.
// The function is idempotent: repeated calls with the same inputs produce
// byte-identical files (sort order preserved via explicit field ordering).
func Generate(paths Paths, cfg Config) error {
	binaryPath, err := resolveBinary(cfg.BinaryPath)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}
	version := cfg.Version
	if version == "" {
		version = "0.0.0-dev"
	}

	marketplace := buildMarketplaceManifest()
	plugin := buildPluginManifest(version)
	mcp := buildMcpManifest(binaryPath, cfg.ExtraArgs)

	if err := writeJSON(paths.MarketplaceManifest, marketplace); err != nil {
		return fmt.Errorf("writing marketplace.json: %w", err)
	}
	if err := writeJSON(paths.PluginManifest, plugin); err != nil {
		return fmt.Errorf("writing plugin.json: %w", err)
	}
	if err := writeJSON(paths.McpJson, mcp); err != nil {
		return fmt.Errorf("writing .mcp.json: %w", err)
	}
	return nil
}

// RemoveMarketplace deletes the entire marketplace tree at paths.MarketplaceRoot.
// Returns nil if the tree does not exist.
func RemoveMarketplace(paths Paths) error {
	if err := os.RemoveAll(paths.MarketplaceRoot); err != nil {
		return fmt.Errorf("removing %s: %w", paths.MarketplaceRoot, err)
	}
	return nil
}

func resolveBinary(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if found, err := exec.LookPath(PluginName); err == nil {
		abs, aerr := filepath.Abs(found)
		if aerr == nil {
			return abs, nil
		}
		return found, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("os.Executable: %w", err)
	}
	return exe, nil
}

// marketplaceManifest is the JSON shape Claude Code expects at
// .claude-plugin/marketplace.json. Field ordering here is preserved in
// encoding/json output via struct tag order.
type marketplaceManifest struct {
	Schema      string                  `json:"$schema"`
	Description string                  `json:"description"`
	Name        string                  `json:"name"`
	Owner       marketplaceOwner        `json:"owner"`
	Plugins     []marketplacePluginRef  `json:"plugins"`
}

type marketplaceOwner struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type marketplacePluginRef struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Source      string `json:"source"`
}

// pluginManifest is the JSON shape at <plugin>/.claude-plugin/plugin.json.
type pluginManifest struct {
	Author      pluginAuthor `json:"author"`
	Description string       `json:"description"`
	Name        string       `json:"name"`
	Version     string       `json:"version"`
}

type pluginAuthor struct {
	Name string `json:"name"`
}

// mcpServerEntry matches the shape used inside .mcp.json (one entry per server).
type mcpServerEntry struct {
	Args    []string `json:"args"`
	Command string   `json:"command"`
}

func buildMarketplaceManifest() marketplaceManifest {
	return marketplaceManifest{
		Schema:      marketplaceSchemaURL,
		Description: marketplaceDesc,
		Name:        MarketplaceName,
		Owner: marketplaceOwner{
			Email: ownerEmail,
			Name:  ownerName,
		},
		Plugins: []marketplacePluginRef{
			{
				Description: pluginDesc,
				Name:        PluginName,
				Source:      "./" + PluginName,
			},
		},
	}
}

func buildPluginManifest(version string) pluginManifest {
	return pluginManifest{
		Author:      pluginAuthor{Name: ownerName},
		Description: pluginDesc,
		Name:        PluginName,
		Version:     version,
	}
}

func buildMcpManifest(binaryPath string, extraArgs []string) map[string]mcpServerEntry {
	args := append([]string{"serve"}, extraArgs...)
	return map[string]mcpServerEntry{
		PluginName: {
			Args:    args,
			Command: binaryPath,
		},
	}
}

// writeJSON marshals v as pretty-printed JSON (2-space indent) with a trailing
// newline, creating parent directories as needed.
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
