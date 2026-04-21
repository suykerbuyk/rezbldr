// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/suykerbuyk/rezbldr/internal/gitops"
)

func registerWrapTool(s *server.MCPServer, cfg *Config) {
	tool := mcp.NewTool("rezbldr_wrap",
		mcp.WithDescription("Stage, commit, and push vault files to all remotes"),
		mcp.WithString("commit_message",
			mcp.Required(),
			mcp.Description("Git commit message"),
		),
		mcp.WithArray("files",
			mcp.Required(),
			mcp.Description("Vault file paths to stage and commit (relative to vault root)"),
			mcp.WithStringItems(),
			mcp.MinItems(1),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		commitMsg := mcp.ExtractString(args, "commit_message")
		if commitMsg == "" {
			return mcp.NewToolResultError("commit_message is required"), nil
		}

		// Extract files array from arguments.
		filesRaw, ok := args["files"]
		if !ok {
			return mcp.NewToolResultError("files is required"), nil
		}
		filesArr, ok := filesRaw.([]any)
		if !ok || len(filesArr) == 0 {
			return mcp.NewToolResultError("files must be a non-empty array of strings"), nil
		}

		var files []string
		for _, f := range filesArr {
			s, ok := f.(string)
			if !ok {
				return mcp.NewToolResultError("each file must be a string"), nil
			}
			files = append(files, s)
		}

		result, err := gitops.Wrap(gitops.WrapRequest{
			RepoDir:       cfg.VaultPath,
			CommitMessage: commitMsg,
			Files:         files,
		})
		if err != nil {
			return mcp.NewToolResultError("wrap failed: " + err.Error()), nil
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})
}
