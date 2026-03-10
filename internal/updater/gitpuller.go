package updater

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitPuller handles git fetch/pull operations
type GitPuller struct {
	repoPath   string
	remoteName string
}

func NewGitPuller(repoPath string) *GitPuller {
	return &GitPuller{
		repoPath:   repoPath,
		remoteName: "origin",
	}
}

// Fetch pulls the latest changes from remote
func (g *GitPuller) Fetch(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "fetch", g.remoteName, "--tags")
	cmd.Dir = g.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %w - %s", err, string(output))
	}
	return nil
}

// Pull fetches and merges changes from remote branch
func (g *GitPuller) Pull(ctx context.Context, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "pull", g.remoteName, branch)
	cmd.Dir = g.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %w - %s", err, string(output))
	}
	return nil
}

// Checkout switches to a specific commit
func (g *GitPuller) Checkout(ctx context.Context, commitSHA string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", commitSHA)
	cmd.Dir = g.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %w - %s", err, string(output))
	}
	return nil
}

// GetRemoteCommit gets the commit hash from remote
func (g *GitPuller) GetRemoteCommit(ctx context.Context, branch string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", g.remoteName, branch)
	cmd.Dir = g.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote failed: %w", err)
	}

	// Output format: <hash>\t<refs/heads/branch>
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 && lines[0] != "" {
		parts := strings.Fields(lines[0])
		if len(parts) >= 1 {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("could not parse remote commit")
}

// FetchAndPull fetches latest and checks out to specific commit
func (g *GitPuller) FetchAndCheckout(ctx context.Context, commitSHA string) error {
	if err := g.Fetch(ctx); err != nil {
		return err
	}
	return g.Checkout(ctx, commitSHA)
}
