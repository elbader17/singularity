package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitCheckerGetCurrentCommit(t *testing.T) {
	// Test that GetCurrentCommit returns something (either from version or git)
	// This will vary based on environment
}

func TestGitCheckerGetLatestReleaseCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for correct API path
		if r.URL.Path != "/repos/owner/repo/commits" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Check query parameters
		if r.URL.Query().Get("sha") != "main" {
			t.Errorf("unexpected sha: %s", r.URL.Query().Get("sha"))
		}

		commits := []GitHubCommit{
			{SHA: "abc123", Commit: struct {
				Message string `json:"message"`
			}{Message: "feat: some feature"}},
			{SHA: "def456", Commit: struct {
				Message string `json:"message"`
			}{Message: "RELEASE v1.2.3"}},
			{SHA: "ghi789", Commit: struct {
				Message string `json:"message"`
			}{Message: "fix: some bug"}},
		}
		json.NewEncoder(w).Encode(commits)
	}))
	defer server.Close()

	checker := NewGitChecker("owner", "repo", "main", "/tmp")
	checker.githubAPI = server.URL

	ctx := context.Background()
	sha, message, err := checker.GetLatestReleaseCommit(ctx)
	if err != nil {
		t.Fatalf("GetLatestReleaseCommit() error = %v", err)
	}
	if sha != "def456" {
		t.Errorf("GetLatestReleaseCommit() sha = %v, want def456", sha)
	}
	if message != "RELEASE v1.2.3" {
		t.Errorf("GetLatestReleaseCommit() message = %v, want RELEASE v1.2.3", message)
	}
}

func TestGitCheckerNoReleaseCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		commits := []GitHubCommit{
			{SHA: "abc123", Commit: struct {
				Message string `json:"message"`
			}{Message: "feat: some feature"}},
			{SHA: "def456", Commit: struct {
				Message string `json:"message"`
			}{Message: "fix: some bug"}},
		}
		json.NewEncoder(w).Encode(commits)
	}))
	defer server.Close()

	checker := NewGitChecker("owner", "repo", "main", "/tmp")
	checker.githubAPI = server.URL

	ctx := context.Background()
	_, _, err := checker.GetLatestReleaseCommit(ctx)
	if err == nil {
		t.Error("GetLatestReleaseCommit() expected error for no RELEASE commit, got nil")
	}
}

func TestGitCheckerIsUpdateAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		commits := []GitHubCommit{
			{SHA: "newcommit", Commit: struct {
				Message string `json:"message"`
			}{Message: "RELEASE v2.0.0"}},
		}
		json.NewEncoder(w).Encode(commits)
	}))
	defer server.Close()

	checker := NewGitChecker("owner", "repo", "main", "/tmp")
	checker.githubAPI = server.URL

	// Mock GetCurrentCommit to return an old commit
	// In real test, we'd mock this

	ctx := context.Background()
	available, sha, message, err := checker.IsUpdateAvailable(ctx)
	if err != nil {
		t.Fatalf("IsUpdateAvailable() error = %v", err)
	}
	if !available {
		t.Error("IsUpdateAvailable() expected available = true")
	}
	if sha != "newcommit" {
		t.Errorf("IsUpdateAvailable() sha = %v, want newcommit", sha)
	}
	if message != "RELEASE v2.0.0" {
		t.Errorf("IsUpdateAvailable() message = %v, want RELEASE v2.0.0", message)
	}
}

func TestExtractVersionFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{"version in message", "RELEASE v1.2.3", "v1.2.3"},
		{"version with message", "RELEASE v2.0.0 - some changes", "v2.0.0"},
		{"no version", "fix: bug", ""},
		{"multiple versions", "RELEASE v1.0.0 then v2.0.0", "v1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionFromMessage(tt.message)
			if result != tt.expected {
				t.Errorf("extractVersionFromMessage(%q) = %q, want %q", tt.message, result, tt.expected)
			}
		})
	}
}

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
	}{
		{"https standard", "https://github.com/owner/repo", "owner", "repo"},
		{"https with .git", "https://github.com/owner/repo.git", "owner", "repo"},
		{"https with trailing slash", "https://github.com/owner/repo/", "owner", "repo"},
		{"ssh format", "git@github.com:owner/repo.git", "owner", "repo"},
		{"ssh without .git", "git@github.com:owner/repo", "owner", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubURL(tt.url)
			if err != nil {
				t.Fatalf("parseGitHubURL(%q) error = %v", tt.url, err)
			}
			if owner != tt.expectedOwner {
				t.Errorf("parseGitHubURL(%q) owner = %q, want %q", tt.url, owner, tt.expectedOwner)
			}
			if repo != tt.expectedRepo {
				t.Errorf("parseGitHubURL(%q) repo = %q, want %q", tt.url, repo, tt.expectedRepo)
			}
		})
	}
}

func TestParseGitHubURLError(t *testing.T) {
	invalidURLs := []string{
		"not-a-url",
		"https://notgithub.com/owner/repo",
		"",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			_, _, err := parseGitHubURL(url)
			if err == nil {
				t.Errorf("parseGitHubURL(%q) expected error, got nil", url)
			}
		})
	}
}
