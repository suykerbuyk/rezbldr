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
)

// Config holds runtime configuration for all tool handlers.
type Config struct {
	VaultPath string
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var vaultPath string
	flag.StringVar(&vaultPath, "vault", "", "path to ResumeCTL vault")
	flag.Parse()

	if vaultPath == "" {
		vaultPath = os.Getenv("REZBLDR_VAULT")
	}
	if vaultPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home directory: %v", err)
		}
		vaultPath = filepath.Join(home, "obsidian", "ResumeCTL")
	}

	// Validate vault path exists.
	if _, err := os.Stat(filepath.Join(vaultPath, "profile", "contact.md")); err != nil {
		log.Fatalf("vault not found at %s: %v", vaultPath, err)
	}

	cfg := &Config{VaultPath: vaultPath}

	s := server.NewMCPServer(
		"rezbldr",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	registerRankTool(s, cfg)
	registerExportTool(s, cfg)
	registerResolveTool(s, cfg)

	log.Printf("rezbldr MCP server starting (vault: %s)", cfg.VaultPath)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
