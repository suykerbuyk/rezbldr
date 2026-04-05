// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/suykerbuyk/rezbldr/internal/scoring"
	"github.com/suykerbuyk/rezbldr/internal/vault"
)

func registerScoreDiffTool(s *server.MCPServer, cfg *Config) {
	tool := mcp.NewTool("rezbldr_score_diff",
		mcp.WithDescription("Compute score change after vault edits during coaching loop"),
		mcp.WithString("job_file",
			mcp.Required(),
			mcp.Description("Path to job file, or \"latest\" for most recent"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		jobFile := mcp.ExtractString(args, "job_file")
		if jobFile == "" {
			return mcp.NewToolResultError("job_file is required"), nil
		}

		// First open: capture previous state.
		prevVault, err := vault.Open(cfg.VaultPath)
		if err != nil {
			return mcp.NewToolResultError("failed to open vault: " + err.Error()), nil
		}

		var job *vault.Job
		if jobFile == "latest" {
			job, err = prevVault.LatestJob()
		} else {
			if !filepath.IsAbs(jobFile) {
				jobFile = filepath.Join(cfg.VaultPath, jobFile)
			}
			job, err = prevVault.LoadJob(jobFile)
		}
		if err != nil {
			return mcp.NewToolResultError("failed to load job: " + err.Error()), nil
		}

		prevRanked := scoring.Rank(job, prevVault.Experiences)
		prevScore := scoring.Score(job, prevVault.Experiences, prevRanked)

		// Second open: pick up any file changes made during coaching.
		newVault, err := vault.Open(cfg.VaultPath)
		if err != nil {
			return mcp.NewToolResultError("failed to reload vault: " + err.Error()), nil
		}

		diff := scoring.Diff(job, newVault.Experiences, prevRanked, prevScore)

		data, err := json.Marshal(diff)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})
}
