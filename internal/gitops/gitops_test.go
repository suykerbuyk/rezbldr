// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupRepo creates a git repo in a temp directory with an initial commit.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "Test User")

	// Create an initial commit so HEAD exists.
	initial := filepath.Join(dir, "README.md")
	if err := os.WriteFile(initial, []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "README.md")
	run(t, dir, "git", "commit", "-m", "initial commit")

	return dir
}

// setupBareRemote creates a bare repo and adds it as a remote to the given repo.
func setupBareRemote(t *testing.T, repoDir, remoteName string) string {
	t.Helper()
	bare := t.TempDir()
	run(t, bare, "git", "init", "--bare")

	// Push initial state to the bare remote so it has a matching branch.
	run(t, repoDir, "git", "remote", "add", remoteName, bare)
	run(t, repoDir, "git", "push", remoteName, "main")

	return bare
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"HOME="+dir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func TestWrapBasicCommit(t *testing.T) {
	repo := setupRepo(t)

	// Create a file to commit.
	testFile := filepath.Join(repo, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Wrap(WrapRequest{
		RepoDir:       repo,
		CommitMessage: "add test file",
		Files:         []string{"test.txt"},
	})
	if err != nil {
		t.Fatalf("Wrap() error: %v", err)
	}

	if !result.Committed {
		t.Error("expected Committed to be true")
	}
	if result.Hash == "" {
		t.Error("expected non-empty commit hash")
	}
	if len(result.Hash) < 7 {
		t.Errorf("commit hash too short: %q", result.Hash)
	}
}

func TestWrapWithRemote(t *testing.T) {
	repo := setupRepo(t)
	setupBareRemote(t, repo, "origin")

	testFile := filepath.Join(repo, "pushed.txt")
	if err := os.WriteFile(testFile, []byte("pushed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Wrap(WrapRequest{
		RepoDir:       repo,
		CommitMessage: "add pushed file",
		Files:         []string{"pushed.txt"},
	})
	if err != nil {
		t.Fatalf("Wrap() error: %v", err)
	}

	if !result.Committed {
		t.Error("expected Committed to be true")
	}
	if len(result.PushResults) != 1 {
		t.Fatalf("expected 1 push result, got %d", len(result.PushResults))
	}
	if !result.PushResults[0].Success {
		t.Errorf("push to origin failed: %s", result.PushResults[0].Error)
	}
}

func TestWrapMultipleRemotes(t *testing.T) {
	repo := setupRepo(t)
	setupBareRemote(t, repo, "github")
	setupBareRemote(t, repo, "backup")

	testFile := filepath.Join(repo, "multi.txt")
	if err := os.WriteFile(testFile, []byte("multi\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Wrap(WrapRequest{
		RepoDir:       repo,
		CommitMessage: "add multi file",
		Files:         []string{"multi.txt"},
	})
	if err != nil {
		t.Fatalf("Wrap() error: %v", err)
	}

	if len(result.PushResults) != 2 {
		t.Fatalf("expected 2 push results, got %d", len(result.PushResults))
	}
	for _, pr := range result.PushResults {
		if !pr.Success {
			t.Errorf("push to %s failed: %s", pr.Remote, pr.Error)
		}
	}
}

func TestWrapMultilineMessage(t *testing.T) {
	repo := setupRepo(t)

	testFile := filepath.Join(repo, "multiline.txt")
	if err := os.WriteFile(testFile, []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	msg := "First line\n\nDetailed description\nwith multiple lines."
	result, err := Wrap(WrapRequest{
		RepoDir:       repo,
		CommitMessage: msg,
		Files:         []string{"multiline.txt"},
	})
	if err != nil {
		t.Fatalf("Wrap() error: %v", err)
	}

	if !result.Committed {
		t.Error("expected Committed to be true")
	}

	// Verify the commit message was preserved.
	out, err := gitOutput(repo, "log", "-1", "--format=%B")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Detailed description") {
		t.Errorf("commit message not preserved: %q", out)
	}
}

func TestWrapValidation(t *testing.T) {
	tests := []struct {
		name string
		req  WrapRequest
	}{
		{"empty repo dir", WrapRequest{CommitMessage: "msg", Files: []string{"f"}}},
		{"empty message", WrapRequest{RepoDir: "/tmp", Files: []string{"f"}}},
		{"no files", WrapRequest{RepoDir: "/tmp", CommitMessage: "msg"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Wrap(tt.req)
			if err == nil {
				t.Error("expected error for invalid request")
			}
		})
	}
}

func TestDiscoverRemotes(t *testing.T) {
	repo := setupRepo(t)

	// No remotes initially.
	remotes, err := discoverRemotes(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(remotes) != 0 {
		t.Errorf("expected 0 remotes, got %d", len(remotes))
	}

	// Add remotes.
	bare1 := t.TempDir()
	run(t, bare1, "git", "init", "--bare")
	run(t, repo, "git", "remote", "add", "alpha", bare1)

	bare2 := t.TempDir()
	run(t, bare2, "git", "init", "--bare")
	run(t, repo, "git", "remote", "add", "beta", bare2)

	remotes, err = discoverRemotes(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(remotes) != 2 {
		t.Errorf("expected 2 remotes, got %d: %v", len(remotes), remotes)
	}
}

func TestWrapFailedPush(t *testing.T) {
	repo := setupRepo(t)

	// Add a remote that will fail to push (non-existent path).
	run(t, repo, "git", "remote", "add", "broken", "/nonexistent/repo.git")

	testFile := filepath.Join(repo, "fail.txt")
	if err := os.WriteFile(testFile, []byte("fail\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Wrap(WrapRequest{
		RepoDir:       repo,
		CommitMessage: "test failed push",
		Files:         []string{"fail.txt"},
	})
	if err != nil {
		t.Fatalf("Wrap() error: %v", err)
	}

	if !result.Committed {
		t.Error("expected Committed to be true despite push failure")
	}
	if len(result.PushResults) != 1 {
		t.Fatalf("expected 1 push result, got %d", len(result.PushResults))
	}
	if result.PushResults[0].Success {
		t.Error("expected push to broken remote to fail")
	}
	if result.PushResults[0].Error == "" {
		t.Error("expected non-empty error message for failed push")
	}
}

func TestGitOutputError(t *testing.T) {
	// Run a git command that will fail with an ExitError.
	_, err := gitOutput(t.TempDir(), "log")
	if err == nil {
		t.Error("expected error for git log in non-repo directory")
	}
}

func TestGitCmdError(t *testing.T) {
	err := gitCmd(t.TempDir(), "log")
	if err == nil {
		t.Error("expected error for git log in non-repo directory")
	}
}

func TestWriteTempMessage(t *testing.T) {
	msg := "test commit message\nwith newlines"
	path, err := writeTempMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != msg {
		t.Errorf("got %q, want %q", string(content), msg)
	}
}
