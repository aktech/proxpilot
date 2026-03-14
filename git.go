package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitOps handles git pull, commit, and push operations.
type GitOps struct {
	repoDir        string
	repoURL        string
	committerEmail string
	committerName  string
	logger         *slog.Logger
}

// NewGitOps creates a git operations handler.
func NewGitOps(repoDir, repoURL, committerEmail, committerName string, logger *slog.Logger) *GitOps {
	return &GitOps{
		repoDir:        repoDir,
		repoURL:        repoURL,
		committerEmail: committerEmail,
		committerName:  committerName,
		logger:         logger,
	}
}

// EnsureCloned clones the repo if it doesn't exist, or pulls if it does.
func (g *GitOps) EnsureCloned() error {
	if _, err := os.Stat(filepath.Join(g.repoDir, ".git")); os.IsNotExist(err) {
		g.logger.Info("cloning repo", "dir", g.repoDir)
		if err := os.MkdirAll(filepath.Dir(g.repoDir), 0755); err != nil {
			return fmt.Errorf("create parent dir: %w", err)
		}
		cmd := exec.Command("git", "clone", "--depth", "1", g.repoURL, g.repoDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return nil
}

// Pull fetches and resets to origin/main.
func (g *GitOps) Pull() error {
	g.logger.Debug("pulling latest changes")
	if err := g.run("git", "-C", g.repoDir, "fetch", "origin", "main"); err != nil {
		return fmt.Errorf("git fetch: %w", err)
	}
	if err := g.run("git", "-C", g.repoDir, "reset", "--hard", "origin/main"); err != nil {
		return fmt.Errorf("git reset: %w", err)
	}
	return nil
}

// HasChanges returns true if the working tree has uncommitted changes.
func (g *GitOps) HasChanges() (bool, error) {
	output, err := g.output("git", "-C", g.repoDir, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

// CommitAndPush stages all changes, commits with the given message, pulls to
// rebase on any upstream changes, and pushes.
func (g *GitOps) CommitAndPush(message string) error {
	if err := g.run("git", "-C", g.repoDir, "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	// Configure committer identity
	if err := g.run("git", "-C", g.repoDir, "config", "user.email", g.committerEmail); err != nil {
		return fmt.Errorf("git config email: %w", err)
	}
	if err := g.run("git", "-C", g.repoDir, "config", "user.name", g.committerName); err != nil {
		return fmt.Errorf("git config name: %w", err)
	}

	if err := g.run("git", "-C", g.repoDir, "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// Pull with rebase to incorporate any upstream changes before pushing
	g.logger.Info("pulling before push (rebase)")
	if err := g.run("git", "-C", g.repoDir, "pull", "--rebase", "origin", "main"); err != nil {
		return fmt.Errorf("git pull --rebase: %w", err)
	}

	g.logger.Info("pushing changes", "message", message)
	if err := g.run("git", "-C", g.repoDir, "push", "origin", "main"); err != nil {
		g.logger.Warn("git push failed (token may lack write access)", "error", err)
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

// run executes a git command and logs it.
func (g *GitOps) run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = g.repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// output executes a command and returns its stdout.
func (g *GitOps) output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = g.repoDir
	out, err := cmd.Output()
	return string(out), err
}
