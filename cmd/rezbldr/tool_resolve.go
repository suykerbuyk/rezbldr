// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/suykerbuyk/rezbldr/internal/resolve"
	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// resolveResult is the JSON output shape for rezbldr_resolve.
type resolveResult struct {
	Path         string   `json:"path"`
	Exists       bool     `json:"exists"`
	Alternatives []string `json:"alternatives,omitempty"`
}

func registerResolveTool(s *server.MCPServer, cfg *Config) {
	tool := mcp.NewTool("rezbldr_resolve",
		mcp.WithDescription("Resolve file paths in the vault using naming conventions"),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("File type: job, resume, cover, or experience"),
			mcp.Enum("job", "resume", "cover", "experience"),
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: latest, generate, or exists"),
			mcp.Enum("latest", "generate", "exists"),
		),
		mcp.WithString("slug",
			mcp.Description("Company or file slug (required for generate and exists)"),
		),
		mcp.WithString("date",
			mcp.Description("Date in YYYY-MM-DD format (for generate and exists)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		typeStr := mcp.ExtractString(args, "type")
		action := mcp.ExtractString(args, "action")
		slug := mcp.ExtractString(args, "slug")
		date := mcp.ExtractString(args, "date")

		ft, err := resolve.ParseFileType(typeStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var result resolveResult

		switch action {
		case "latest":
			path, err := resolve.Latest(cfg.VaultPath, ft)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result.Path = path
			result.Exists = true

		case "generate":
			if slug == "" {
				return mcp.NewToolResultError("slug is required for generate action"), nil
			}
			// Get candidate name from contact profile.
			candidate := "candidate"
			v, err := vault.Open(cfg.VaultPath)
			if err == nil && v.Contact != nil && v.Contact.Name != "" {
				candidate = slugify(v.Contact.Name)
			}
			if date == "" {
				return mcp.NewToolResultError("date is required for generate action on resume/cover types"), nil
			}
			result.Path = resolve.Generate(cfg.VaultPath, ft, slug, date, candidate)

		case "exists":
			if slug == "" && date == "" {
				return mcp.NewToolResultError("slug or date required for exists action"), nil
			}
			path, exists, alts := resolve.Exists(cfg.VaultPath, ft, slug, date)
			result.Path = path
			result.Exists = exists
			result.Alternatives = alts

		default:
			return mcp.NewToolResultError("unknown action: " + action), nil
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})
}

// slugify converts a name like "John Suykerbuyk" to "john-suykerbuyk".
func slugify(name string) string {
	var b []byte
	for _, c := range []byte(name) {
		switch {
		case c >= 'A' && c <= 'Z':
			b = append(b, c+32) // lowercase
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b = append(b, c)
		case c == ' ' || c == '_':
			if len(b) > 0 && b[len(b)-1] != '-' {
				b = append(b, '-')
			}
		}
	}
	// Trim trailing hyphen.
	if len(b) > 0 && b[len(b)-1] == '-' {
		b = b[:len(b)-1]
	}
	return string(b)
}
