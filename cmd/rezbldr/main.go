// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"
	installer "github.com/suykerbuyk/claude-plugin-installer"
	"github.com/suykerbuyk/rezbldr/internal/check"
	"github.com/suykerbuyk/rezbldr/internal/install"
)

// rezbldrIdentity returns the Identity used when installing rezbldr as a
// Claude Code plugin. Centralized so cmdSetup, cmdUninstall, cmdCheck, and
// the check package all agree on the same values.
func rezbldrIdentity() installer.Identity {
	return installer.Identity{
		PluginName:          "rezbldr",
		PluginDesc:          "MCP server for deterministic resume pipeline operations",
		McpArgs:             []string{"serve"},
		LegacyMcpServerName: "rezbldr",
	}.WithDefaults()
}

// Version variables injected via ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Config holds runtime configuration for all tool handlers.
type Config struct {
	VaultPath string
	// ExtraVaults maps a short name (e.g. "vibe") to the absolute path of
	// an additional git repo that rezbldr_wrap is allowed to operate on.
	// The set is server-configured; MCP callers only see the names, never
	// the paths.
	ExtraVaults map[string]string
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	subcmd := "serve"
	if len(os.Args) > 1 {
		subcmd = os.Args[1]
	}

	switch subcmd {
	case "serve":
		cmdServe(os.Args[1:]) // pass remaining args (may include --vault)
	case "version":
		cmdVersion()
	case "setup":
		cmdSetup(os.Args[2:])
	case "install":
		fmt.Fprintln(os.Stderr, "Warning: 'install' is deprecated, use 'setup' instead")
		cmdSetup(os.Args[2:])
	case "check":
		cmdCheck(os.Args[2:])
	case "uninstall":
		cmdUninstall(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		// If the first arg looks like a flag, treat it as args to serve.
		if len(os.Args) > 1 && os.Args[1][0] == '-' {
			cmdServe(os.Args[1:])
		} else {
			fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", subcmd)
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: rezbldr <command> [options]

Commands:
  serve       Start the MCP server (default)
  setup       Install binary and register as global MCP server
  check       Validate vault and configuration
  uninstall   Remove rezbldr from MCP client config
  version     Print version information
  help        Show this help message

Run 'rezbldr serve --help' for serve-specific options.
`)
}

func cmdVersion() {
	fmt.Printf("rezbldr %s (commit: %s, built: %s)\n", version, commit, date)
}

func cmdCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	var vaultPath string
	fs.StringVar(&vaultPath, "vault", "", "path to Obsidian vault for resume documents (default: ~/obsidian/RezBldrVault)")
	fs.Parse(args)

	if vaultPath == "" {
		vaultPath = os.Getenv("REZBLDR_VAULT")
	}
	if vaultPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home directory: %v", err)
		}
		vaultPath = filepath.Join(home, "obsidian", "RezBldrVault")
	}

	results := check.Run(vaultPath)

	hasFail := false
	for _, r := range results {
		var icon string
		switch r.Status {
		case "ok":
			icon = "\u2713" // ✓
		case "warn":
			icon = "!"
		default:
			icon = "\u2717" // ✗
			hasFail = true
		}
		fmt.Printf("[%s] %s: %s\n", icon, r.Name, r.Detail)
	}

	if hasFail {
		os.Exit(1)
	}
}

func cmdSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	var vaultPath, prefix string
	fs.StringVar(&vaultPath, "vault", "", "path to Obsidian vault (default: ~/obsidian/RezBldrVault)")
	fs.StringVar(&prefix, "prefix", "", "installation prefix; binary at PREFIX/bin/rezbldr (default: ~/.local)")
	extraVaults := newExtraVaultFlag()
	fs.Var(extraVaults, "extra-vault", "additional named vault for rezbldr_wrap (repeatable, format: name=path). Persisted into the plugin's serve args.")
	fs.Parse(args)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot determine home directory: %v", err)
	}

	if vaultPath == "" {
		vaultPath = os.Getenv("REZBLDR_VAULT")
	}

	if prefix == "" {
		prefix = filepath.Join(home, ".local")
	}
	binaryPath := filepath.Join(prefix, "bin", "rezbldr")

	if err := install.CopyBinary(binaryPath); err != nil {
		log.Fatalf("copying binary: %v", err)
	}

	paths, err := installer.Default(rezbldrIdentity())
	if err != nil {
		log.Fatalf("resolving plugin paths: %v", err)
	}
	cfg := installer.Config{
		Version:    version,
		BinaryPath: binaryPath,
	}
	if vaultPath != "" {
		cfg.ExtraArgs = append(cfg.ExtraArgs, "--vault", vaultPath)
	}
	for _, entry := range extraVaults.sortedEntries() {
		cfg.ExtraArgs = append(cfg.ExtraArgs, "--extra-vault", entry)
	}

	if err := installer.Install(paths, cfg); err != nil {
		log.Fatalf("plugin install: %v", err)
	}

	// Legacy cleanup: remove iteration-11 project-scoped entries from
	// ~/.claude.json and iteration-early .mcp.json entries.
	claudeJsonPath := filepath.Join(home, ".claude.json")
	cleaned, err := install.MigrateProjectScoped(claudeJsonPath)
	if err != nil {
		log.Fatalf("migrating project-scoped entries: %v", err)
	}
	for _, p := range cleaned {
		fmt.Printf("Migrated: removed rezbldr from project %s in %s\n", p, filepath.Base(claudeJsonPath))
	}
	if err := install.CleanupLegacyMcpJson(paths.ClaudeDir); err != nil {
		log.Fatalf("cleaning legacy .mcp.json: %v", err)
	}

	fmt.Printf("\nSetup complete.\n")
	fmt.Printf("  Marketplace: %s\n", paths.MarketplaceRoot)
	fmt.Printf("  Settings:    %s\n", paths.Settings)
	fmt.Printf("  Cache:       %s\n", paths.CacheVersionDir(version))
	fmt.Println("Restart Claude Code to load the rezbldr MCP server.")
}

func cmdUninstall(args []string) {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	var prefix string
	var keepBinary bool
	fs.StringVar(&prefix, "prefix", "", "installation prefix; binary is at PREFIX/bin/rezbldr (default: ~/.local)")
	fs.BoolVar(&keepBinary, "keep-binary", false, "leave the installed binary in place")
	fs.Parse(args)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot determine home directory: %v", err)
	}

	if prefix == "" {
		prefix = filepath.Join(home, ".local")
	}

	paths, err := installer.Default(rezbldrIdentity())
	if err != nil {
		log.Fatalf("resolving plugin paths: %v", err)
	}

	if err := installer.Uninstall(paths); err != nil {
		log.Fatalf("plugin uninstall: %v", err)
	}

	// Legacy cleanup: remove any iteration-11 project-scoped entries and
	// iteration-early ~/.claude/.mcp.json residue.
	claudeJsonPath := filepath.Join(home, ".claude.json")
	if _, err := install.MigrateProjectScoped(claudeJsonPath); err != nil {
		log.Fatalf("cleaning project-scoped entries: %v", err)
	}
	if err := install.CleanupLegacyMcpJson(paths.ClaudeDir); err != nil {
		log.Fatalf("cleaning legacy .mcp.json: %v", err)
	}

	if !keepBinary {
		binaryPath := filepath.Join(prefix, "bin", "rezbldr")
		if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
			log.Fatalf("removing binary %s: %v", binaryPath, err)
		} else if err == nil {
			fmt.Printf("Removed binary %s\n", binaryPath)
		}
	}

	fmt.Println("Uninstall complete.")
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	var vaultPath string
	fs.StringVar(&vaultPath, "vault", "", "path to Obsidian vault for resume documents (default: ~/obsidian/RezBldrVault)")
	extraVaults := newExtraVaultFlag()
	fs.Var(extraVaults, "extra-vault", "additional named vault for rezbldr_wrap (repeatable, format: name=path)")
	// Skip the "serve" subcommand itself if present.
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}
	fs.Parse(args)

	if vaultPath == "" {
		vaultPath = os.Getenv("REZBLDR_VAULT")
	}
	if vaultPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home directory: %v", err)
		}
		vaultPath = filepath.Join(home, "obsidian", "RezBldrVault")
	}

	// Validate vault path exists.
	if _, err := os.Stat(filepath.Join(vaultPath, "profile", "contact.md")); err != nil {
		log.Fatalf("vault not found at %s: %v", vaultPath, err)
	}

	envExtras, err := parseExtraVaultsEnv(os.Getenv("REZBLDR_EXTRA_VAULTS"))
	if err != nil {
		log.Fatalf("parsing REZBLDR_EXTRA_VAULTS: %v", err)
	}

	cfg := &Config{
		VaultPath:   vaultPath,
		ExtraVaults: mergeExtraVaults(extraVaults.values, envExtras),
	}

	s := server.NewMCPServer(
		"rezbldr",
		version,
		server.WithToolCapabilities(true),
	)

	registerRankTool(s, cfg)
	registerExportTool(s, cfg)
	registerResolveTool(s, cfg)
	registerFrontmatterTool(s, cfg)
	registerScoreDiffTool(s, cfg)
	registerValidateTool(s, cfg)
	registerWrapTool(s, cfg)

	log.Printf("rezbldr MCP server starting (vault: %s)", cfg.VaultPath)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
