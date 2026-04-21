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
	"github.com/suykerbuyk/rezbldr/internal/check"
	"github.com/suykerbuyk/rezbldr/internal/install"
)

// Version variables injected via ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Config holds runtime configuration for all tool handlers.
type Config struct {
	VaultPath string
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
	var vaultPath, prefix, settingsPath string
	fs.StringVar(&vaultPath, "vault", "", "path to Obsidian vault (default: ~/obsidian/RezBldrVault)")
	fs.StringVar(&prefix, "prefix", "", "installation prefix; binary at PREFIX/bin/rezbldr (default: ~/.local)")
	fs.StringVar(&settingsPath, "settings", "", "Claude Code settings file (default: ~/.claude/settings.json)")
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

	claudeDir := filepath.Join(home, ".claude")
	if settingsPath == "" {
		settingsPath = filepath.Join(claudeDir, "settings.json")
	}
	claudeJsonPath := filepath.Join(home, ".claude.json")

	if err := install.Setup(binaryPath, settingsPath, claudeJsonPath, claudeDir, vaultPath); err != nil {
		log.Fatalf("setup failed: %v", err)
	}

	fmt.Println("\nSetup complete. Restart Claude Code to load the rezbldr MCP server.")
}

func cmdUninstall(args []string) {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	var prefix string
	fs.StringVar(&prefix, "prefix", "", "installation prefix; binary is at PREFIX/bin/rezbldr (default: ~/.local)")
	fs.Parse(args)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot determine home directory: %v", err)
	}

	if prefix == "" {
		prefix = filepath.Join(home, ".local")
	}

	configPath := filepath.Join(home, ".claude.json")
	binaryPath := filepath.Join(prefix, "bin", "rezbldr")
	legacyDir := filepath.Join(home, ".claude")

	// Get cwd for project-scoped cleanup, but don't fail if unavailable.
	projectDir, _ := os.Getwd()

	if err := install.Uninstall(configPath, projectDir, legacyDir, binaryPath); err != nil {
		log.Fatalf("uninstall failed: %v", err)
	}
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	var vaultPath string
	fs.StringVar(&vaultPath, "vault", "", "path to Obsidian vault for resume documents (default: ~/obsidian/RezBldrVault)")
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

	cfg := &Config{VaultPath: vaultPath}

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
