// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

//go:build integration

package installintegration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	installer "github.com/suykerbuyk/claude-plugin-installer"
)

// rezbldrIdentity mirrors the identity the real cmd/rezbldr constructs.
// Keeping it in the test file means the integration test is locked to the
// production values without a wider refactor.
func rezbldrIdentity() installer.Identity {
	return installer.Identity{
		PluginName:          "rezbldr",
		PluginDesc:          "MCP server for deterministic resume pipeline operations",
		McpArgs:             []string{"serve"},
		LegacyMcpServerName: "rezbldr",
	}.WithDefaults()
}

// TestIntegration_InstallProbeUninstall is the full end-to-end validation:
// build the rezbldr binary, install it as a plugin under a fake HOME,
// speak MCP JSON-RPC to the installed binary, verify the tool list
// includes rezbldr_wrap, then uninstall and verify a clean state.
//
// Runs only with -tags integration. Requires `go build` to be available
// on PATH.
func TestIntegration_InstallProbeUninstall(t *testing.T) {
	home := t.TempDir()
	paths := installer.FromHome(home, rezbldrIdentity())

	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir binDir: %v", err)
	}
	binaryPath := filepath.Join(binDir, "rezbldr")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/rezbldr")
	buildCmd.Dir = findRepoRoot(t)
	buildCmd.Stdout = os.Stderr
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("building rezbldr: %v", err)
	}

	cfg := installer.Config{
		Version:    "0.0.0-integration",
		BinaryPath: binaryPath,
	}
	if err := installer.Install(paths, cfg); err != nil {
		t.Fatalf("installer.Install: %v", err)
	}

	for _, p := range []string{
		paths.MarketplaceManifest, paths.PluginManifest, paths.McpJson,
		paths.Settings, paths.KnownMarketplaces, paths.InstalledPlugins,
		paths.CachePluginManifest(cfg.Version), paths.CacheMcpJson(cfg.Version),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s: %v", p, err)
		}
	}
	if !installer.HealthCheck(paths).Healthy() {
		t.Fatalf("HealthCheck not healthy after install: %+v", installer.HealthCheck(paths))
	}

	tools := probeMCPTools(t, binaryPath)
	if len(tools) == 0 {
		t.Fatal("no tools reported by rezbldr serve")
	}
	want := map[string]bool{
		"rezbldr_wrap":        false,
		"rezbldr_rank":        false,
		"rezbldr_export":      false,
		"rezbldr_resolve":     false,
		"rezbldr_frontmatter": false,
		"rezbldr_score_diff":  false,
		"rezbldr_validate":    false,
	}
	for _, name := range tools {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("expected tool %q in server response, got %v", name, tools)
		}
	}

	if err := installer.Uninstall(paths); err != nil {
		t.Fatalf("installer.Uninstall: %v", err)
	}
	for _, p := range []string{paths.MarketplaceManifest, paths.KnownMarketplaces, paths.InstalledPlugins} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed after Uninstall (err=%v)", p, err)
		}
	}
	if installer.HealthCheck(paths).Healthy() {
		t.Error("HealthCheck reports healthy after uninstall")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found walking up from %s", cwd)
		}
		dir = parent
	}
}

func probeMCPTools(t *testing.T, binaryPath string) []string {
	t.Helper()

	vault := t.TempDir()
	_ = os.MkdirAll(filepath.Join(vault, "profile"), 0o755)
	_ = os.MkdirAll(filepath.Join(vault, "jobs", "target"), 0o755)
	_ = os.MkdirAll(filepath.Join(vault, "resumes"), 0o755)
	_ = os.WriteFile(filepath.Join(vault, "profile", "contact.md"), []byte("---\nname: T\n---\n"), 0o644)

	cmd := exec.Command(binaryPath, "serve", "--vault", vault)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start serve: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	req := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"integration","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	}, "\n") + "\n"

	if _, err := io.WriteString(stdin, req); err != nil {
		t.Fatalf("write: %v", err)
	}

	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		scan := make([]byte, 16*1024)
		for {
			n, err := stdout.Read(scan)
			if n > 0 {
				buf.Write(scan[:n])
				if bytes.Contains(buf.Bytes(), []byte(`"id":2`)) {
					done <- buf.Bytes()
					return
				}
			}
			if err != nil {
				done <- buf.Bytes()
				return
			}
		}
	}()

	var raw []byte
	select {
	case raw = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for tools/list response")
	}

	var tools []string
	for _, line := range bytes.Split(raw, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var resp struct {
			ID     int `json:"id"`
			Result struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		if resp.ID == 2 {
			for _, tl := range resp.Result.Tools {
				tools = append(tools, tl.Name)
			}
			break
		}
	}
	return tools
}
