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

// rankResult is the JSON output shape for rezbldr_rank.
type rankResult struct {
	Job        rankJobInfo                `json:"job"`
	Ranked     []scoring.ScoredExperience `json:"ranked"`
	MatchScore scoring.MatchScore         `json:"match_score"`
}

type rankJobInfo struct {
	Title          string `json:"title"`
	Company        string `json:"company"`
	RequiredCount  int    `json:"required_count"`
	PreferredCount int    `json:"preferred_count"`
	FilePath       string `json:"file_path"`
}

func registerRankTool(s *server.MCPServer, cfg *Config) {
	tool := mcp.NewTool("rezbldr_rank",
		mcp.WithDescription("Rank experience files against a job posting using tag intersection scoring"),
		mcp.WithString("job_file",
			mcp.Required(),
			mcp.Description("Path to job file, or \"latest\" for most recent"),
		),
		mcp.WithNumber("top_n",
			mcp.Description("Number of results to return (default: 8)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		jobFile := mcp.ExtractString(args, "job_file")
		if jobFile == "" {
			return mcp.NewToolResultError("job_file is required"), nil
		}

		topN := 8
		if n, ok := args["top_n"]; ok {
			if f, ok := n.(float64); ok && f > 0 {
				topN = int(f)
			}
		}

		v, err := vault.Open(cfg.VaultPath)
		if err != nil {
			return mcp.NewToolResultError("failed to open vault: " + err.Error()), nil
		}

		var job *vault.Job
		if jobFile == "latest" {
			job, err = v.LatestJob()
		} else {
			if !filepath.IsAbs(jobFile) {
				jobFile = filepath.Join(cfg.VaultPath, jobFile)
			}
			job, err = v.LoadJob(jobFile)
		}
		if err != nil {
			return mcp.NewToolResultError("failed to load job: " + err.Error()), nil
		}

		ranked := scoring.Rank(job, v.Experiences)
		matchScore := scoring.Score(job, v.Experiences, ranked)

		if topN < len(ranked) {
			ranked = ranked[:topN]
		}

		result := rankResult{
			Job: rankJobInfo{
				Title:          job.Title,
				Company:        job.Company,
				RequiredCount:  len(job.RequiredSkills),
				PreferredCount: len(job.PreferredSkills),
				FilePath:       job.FilePath,
			},
			Ranked:     ranked,
			MatchScore: matchScore,
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})
}
