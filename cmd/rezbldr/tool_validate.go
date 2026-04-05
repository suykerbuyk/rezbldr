// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/suykerbuyk/rezbldr/internal/validate"
	"github.com/suykerbuyk/rezbldr/internal/vault"
)

func registerValidateTool(s *server.MCPServer, cfg *Config) {
	tool := mcp.NewTool("rezbldr_validate",
		mcp.WithDescription("Validate a generated resume against vault data (word count, headings, skills, companies, contact)"),
		mcp.WithString("resume_path",
			mcp.Required(),
			mcp.Description("Path to the generated resume .md file"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		resumePath := mcp.ExtractString(args, "resume_path")
		if resumePath == "" {
			return mcp.NewToolResultError("resume_path is required"), nil
		}

		if !filepath.IsAbs(resumePath) {
			resumePath = filepath.Join(cfg.VaultPath, resumePath)
		}

		content, err := os.ReadFile(resumePath)
		if err != nil {
			return mcp.NewToolResultError("failed to read resume: " + err.Error()), nil
		}

		body := vault.Strip(content)

		v, err := vault.Open(cfg.VaultPath)
		if err != nil {
			return mcp.NewToolResultError("failed to open vault: " + err.Error()), nil
		}

		result := validate.Resume(string(body), v)

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})
}
