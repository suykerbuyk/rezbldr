// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WrapRequest describes a git stage + commit + push operation.
type WrapRequest struct {
	RepoDir       string
	CommitMessage string
	Files         []string
}

// PushResult captures the outcome of pushing to a single remote.
type PushResult struct {
	Remote  string `json:"remote"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// WrapResult captures the outcome of the full wrap operation.
type WrapResult struct {
	Committed   bool         `json:"committed"`
	Hash        string       `json:"hash"`
	PushResults []PushResult `json:"push_results"`
}

// Wrap stages the specified files, commits with the given message,
// discovers all remotes, and pushes to each one.
func Wrap(req WrapRequest) (*WrapResult, error) {
	if req.RepoDir == "" {
		return nil, fmt.Errorf("repo directory is required")
	}
	if req.CommitMessage == "" {
		return nil, fmt.Errorf("commit message is required")
	}
	if len(req.Files) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}

	// Stage each file individually.
	for _, f := range req.Files {
		if err := gitCmd(req.RepoDir, "add", f); err != nil {
			return nil, fmt.Errorf("git add %s: %w", f, err)
		}
	}

	// Commit using a temp file for the message to handle multiline safely.
	msgFile, err := writeTempMessage(req.CommitMessage)
	if err != nil {
		return nil, fmt.Errorf("write commit message: %w", err)
	}
	defer os.Remove(msgFile)

	if err := gitCmd(req.RepoDir, "commit", "-F", msgFile); err != nil {
		return nil, fmt.Errorf("git commit: %w", err)
	}

	// Get the commit hash.
	hash, err := gitOutput(req.RepoDir, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git rev-parse: %w", err)
	}

	// Discover the current branch.
	branch, err := gitOutput(req.RepoDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git branch: %w", err)
	}

	// Discover remotes and push to each.
	remotes, err := discoverRemotes(req.RepoDir)
	if err != nil {
		return &WrapResult{
			Committed: true,
			Hash:      hash,
		}, nil // Committed but couldn't discover remotes.
	}

	var pushResults []PushResult
	for _, remote := range remotes {
		pr := PushResult{Remote: remote, Success: true}
		if err := gitCmd(req.RepoDir, "push", remote, branch); err != nil {
			pr.Success = false
			pr.Error = err.Error()
		}
		pushResults = append(pushResults, pr)
	}

	return &WrapResult{
		Committed:   true,
		Hash:        hash,
		PushResults: pushResults,
	}, nil
}

// discoverRemotes returns the names of all configured git remotes.
func discoverRemotes(repoDir string) ([]string, error) {
	output, err := gitOutput(repoDir, "remote")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}
	var remotes []string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			remotes = append(remotes, trimmed)
		}
	}
	return remotes, nil
}

// gitCmd runs a git command in the specified directory.
func gitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// gitOutput runs a git command and returns trimmed stdout.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// writeTempMessage writes the commit message to a temp file and returns its path.
func writeTempMessage(msg string) (string, error) {
	f, err := os.CreateTemp("", "rezbldr-commit-*.txt")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(msg); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	name := f.Name()
	f.Close()
	return filepath.Clean(name), nil
}
