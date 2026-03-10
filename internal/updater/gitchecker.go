package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"singularity/internal/version"
)

// GitChecker checks for RELEASE commits on GitHub
type GitChecker struct {
	owner     string
	repo      string
	branch    string
	githubAPI string
	repoPath  string
}

type GitHubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

// NewGitCheckerFromRepo creates a GitChecker by detecting the remote origin automatically
func NewGitCheckerFromRepo(repoPath string) (*GitChecker, error) {
	// Get the remote origin URL
	remoteURL, err := getRemoteOrigin(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote origin: %w", err)
	}

	// Parse owner and repo from URL
	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}

	// Get the current branch
	branch, err := getCurrentBranch(repoPath)
	if err != nil {
		branch = "main" // default to main
	}

	return &GitChecker{
		owner:     owner,
		repo:      repo,
		branch:    branch,
		githubAPI: "https://api.github.com",
		repoPath:  repoPath,
	}, nil
}

func NewGitChecker(owner, repo, branch, repoPath string) *GitChecker {
	return &GitChecker{
		owner:     owner,
		repo:      repo,
		branch:    branch,
		githubAPI: "https://api.github.com",
		repoPath:  repoPath,
	}
}

// getRemoteOrigin gets the URL of the 'origin' remote
func getRemoteOrigin(repoPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no remote origin found: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// parseGitHubURL parses a GitHub URL to extract owner and repo
// Supports:
// - https://github.com/owner/repo
// - https://github.com/owner/repo.git
// - git@github.com:owner/repo.git
func parseGitHubURL(remoteURL string) (string, string, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	remoteURL = strings.TrimSuffix(remoteURL, "/")

	// Handle SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@") {
		// Remove git@ prefix
		remoteURL = strings.TrimPrefix(remoteURL, "git@")
		// Replace : with /
		remoteURL = strings.Replace(remoteURL, ":", "/", 1)
		// Remove .git suffix
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		parts := strings.Split(remoteURL, "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid SSH URL format: %s", remoteURL)
		}
		return parts[len(parts)-2], parts[len(parts)-1], nil
	}

	// Handle HTTPS format
	u, err := url.Parse(remoteURL)
	if err != nil {
		return "", "", err
	}

	// Validate it's github.com (exact match or subdomain)
	host := u.Host
	if host != "github.com" && !strings.HasSuffix(host, ".github.com") {
		return "", "", fmt.Errorf("not a GitHub URL: %s", remoteURL)
	}

	// Remove leading slash from path
	path := strings.TrimPrefix(u.Path, "/")
	// Remove .git suffix
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL: %s", remoteURL)
	}
	return parts[len(parts)-2], parts[len(parts)-1], nil
}

// getCurrentBranch gets the current checked out branch
func getCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCurrentCommit returns the current git commit hash
func (c *GitChecker) GetCurrentCommit() string {
	// Try to get from version package first (if set at build time)
	if version.Version != "" && len(version.Version) == 40 {
		return version.Version
	}

	// Otherwise get from git
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = c.repoPath
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GetLatestReleaseCommit returns the latest commit message starting with RELEASE
func (c *GitChecker) GetLatestReleaseCommit(ctx context.Context) (string, string, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/commits?sha=%s&per_page=100",
		c.githubAPI, c.owner, c.repo, c.branch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to fetch commits: status %d", resp.StatusCode)
	}

	var commits []GitHubCommit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Find latest commit starting with RELEASE
	for _, commit := range commits {
		if strings.HasPrefix(commit.Commit.Message, "RELEASE") {
			return commit.SHA, commit.Commit.Message, nil
		}
	}

	return "", "", fmt.Errorf("no RELEASE commit found on branch %s", c.branch)
}

// IsUpdateAvailable checks if there's a newer RELEASE commit
func (c *GitChecker) IsUpdateAvailable(ctx context.Context) (bool, string, string, error) {
	current := c.GetCurrentCommit()
	latest, message, err := c.GetLatestReleaseCommit(ctx)
	if err != nil {
		return false, "", "", err
	}

	// If no current commit or different from latest, update is available
	if current == "" || current != latest {
		return true, latest, message, nil
	}

	return false, latest, message, nil
}
