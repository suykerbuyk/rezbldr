// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/suykerbuyk/rezbldr/internal/export"
	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// exportResult is the JSON output shape for rezbldr_export.
type exportResult struct {
	Resume *export.Result `json:"resume"`
	Cover  *export.Result `json:"cover,omitempty"`
	Errors []string       `json:"errors,omitempty"`
}

func registerExportTool(s *server.MCPServer, cfg *Config) {
	tool := mcp.NewTool("rezbldr_export",
		mcp.WithDescription("Export a resume (and matching cover letter) to DOCX or PDF via pandoc"),
		mcp.WithString("source",
			mcp.Description("Path to resume .md file, or \"latest\" (default: latest)"),
		),
		mcp.WithString("format",
			mcp.Description("Output format: \"docx\" or \"pdf\" (default: docx)"),
			mcp.Enum("docx", "pdf"),
		),
		mcp.WithString("template",
			mcp.Description("Optional path to reference doc for pandoc"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		source := mcp.ExtractString(args, "source")
		format := mcp.ExtractString(args, "format")
		template := mcp.ExtractString(args, "template")

		if format == "" {
			format = "docx"
		}

		// Resolve source path.
		if source == "" || source == "latest" {
			v, err := vault.Open(cfg.VaultPath)
			if err != nil {
				return mcp.NewToolResultError("failed to open vault: " + err.Error()), nil
			}
			resume, err := v.LatestResume()
			if err != nil {
				return mcp.NewToolResultError("failed to find latest resume: " + err.Error()), nil
			}
			source = resume.FilePath
		}

		var result exportResult
		var errs []string

		// Export resume.
		res, err := export.Export(export.Request{
			Source:   source,
			Format:   format,
			Template: template,
		})
		if err != nil {
			return mcp.NewToolResultError("export failed: " + err.Error()), nil
		}
		result.Resume = res

		// Look for matching cover letter.
		coverPath := export.FindMatchingCoverLetter(cfg.VaultPath, source)
		if coverPath != "" {
			coverRes, err := export.Export(export.Request{
				Source:   coverPath,
				Format:   format,
				Template: template,
			})
			if err != nil {
				errs = append(errs, "cover letter export failed: "+err.Error())
			} else {
				result.Cover = coverRes
			}
		}

		if len(errs) > 0 {
			result.Errors = errs
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})
}
