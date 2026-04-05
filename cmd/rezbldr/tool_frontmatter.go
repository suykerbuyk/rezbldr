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
	"github.com/suykerbuyk/rezbldr/internal/vault"
	"gopkg.in/yaml.v3"
)

type frontmatterResult struct {
	Action      string         `json:"action"`
	File        string         `json:"file,omitempty"`
	Frontmatter map[string]any `json:"frontmatter,omitempty"`
	Body        string         `json:"body,omitempty"`
	Content     string         `json:"content,omitempty"`
}

func registerFrontmatterTool(s *server.MCPServer, cfg *Config) {
	tool := mcp.NewTool("rezbldr_frontmatter",
		mcp.WithDescription("Parse, strip, or generate YAML frontmatter for vault files"),
		mcp.WithString("file",
			mcp.Description("Path to file (required for parse/strip, ignored for generate)"),
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action to perform: \"parse\", \"strip\", or \"generate\""),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		action := mcp.ExtractString(args, "action")
		file := mcp.ExtractString(args, "file")

		switch action {
		case "parse":
			return handleFrontmatterParse(cfg, file)
		case "strip":
			return handleFrontmatterStrip(cfg, file)
		case "generate":
			data, _ := args["data"].(map[string]any)
			if data == nil {
				return mcp.NewToolResultError("data is required for generate action"), nil
			}
			return handleFrontmatterGenerate(data)
		default:
			return mcp.NewToolResultError("action must be \"parse\", \"strip\", or \"generate\""), nil
		}
	})
}

func handleFrontmatterParse(cfg *Config, file string) (*mcp.CallToolResult, error) {
	if file == "" {
		return mcp.NewToolResultError("file is required for parse action"), nil
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(cfg.VaultPath, file)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return mcp.NewToolResultError("failed to read file: " + err.Error()), nil
	}

	fm, body, err := vault.Parse(content)
	if err != nil {
		return mcp.NewToolResultError("failed to parse frontmatter: " + err.Error()), nil
	}

	var parsed map[string]any
	if fm != nil && len(fm) > 0 {
		if err := yaml.Unmarshal(fm, &parsed); err != nil {
			return mcp.NewToolResultError("failed to unmarshal YAML: " + err.Error()), nil
		}
	}

	result := frontmatterResult{
		Action:      "parse",
		File:        file,
		Frontmatter: parsed,
		Body:        string(body),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func handleFrontmatterStrip(cfg *Config, file string) (*mcp.CallToolResult, error) {
	if file == "" {
		return mcp.NewToolResultError("file is required for strip action"), nil
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(cfg.VaultPath, file)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return mcp.NewToolResultError("failed to read file: " + err.Error()), nil
	}

	body := vault.Strip(content)

	result := frontmatterResult{
		Action: "strip",
		File:   file,
		Body:   string(body),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func handleFrontmatterGenerate(fmData map[string]any) (*mcp.CallToolResult, error) {
	generated, err := vault.Generate(fmData)
	if err != nil {
		return mcp.NewToolResultError("failed to generate frontmatter: " + err.Error()), nil
	}

	result := frontmatterResult{
		Action:  "generate",
		Content: string(generated),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
