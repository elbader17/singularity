package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"singularity/internal/version"
	"strings"
)

type GitHubChecker struct {
	baseURL string
	owner   string
	repo    string
}

func NewGitHubChecker(baseURL, owner, repo string) *GitHubChecker {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &GitHubChecker{
		baseURL: baseURL,
		owner:   owner,
		repo:    repo,
	}
}

func (c *GitHubChecker) GetCurrentVersion() string {
	return version.Version
}

func (c *GitHubChecker) GetLatestVersion(ctx context.Context) (string, error) {
	parsedURL, err := url.Parse(c.baseURL)
	var apiURL string

	if err == nil && parsedURL.Path != "" && parsedURL.Path != "/" {
		apiURL = c.baseURL + "/releases/latest"
	} else {
		apiURL = fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, c.owner, c.repo)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest release: status %d", resp.StatusCode)
	}

	var result struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.TagName, nil
}
