package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockVersionChecker struct {
	currentVersion string
	latestVersion  string
	err            error
}

func (m *MockVersionChecker) GetCurrentVersion() string {
	return m.currentVersion
}

func (m *MockVersionChecker) GetLatestVersion(ctx context.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.latestVersion, nil
}

func TestCheckerGetLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]string{"tag_name": "v1.2.3"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	checker := NewGitHubChecker(server.URL+"/repos/owner/repo", "owner", "repo")
	ctx := context.Background()

	version, err := checker.GetLatestVersion(ctx)
	if err != nil {
		t.Fatalf("GetLatestVersion() error = %v", err)
	}
	if version != "v1.2.3" {
		t.Errorf("GetLatestVersion() = %v, want v1.2.3", version)
	}
}

func TestCheckerGetLatestVersionHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewGitHubChecker(server.URL, "owner", "repo")
	ctx := context.Background()

	_, err := checker.GetLatestVersion(ctx)
	if err == nil {
		t.Error("GetLatestVersion() expected error, got nil")
	}
}

func TestCheckerGetLatestVersionInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	checker := NewGitHubChecker(server.URL, "owner", "repo")
	ctx := context.Background()

	_, err := checker.GetLatestVersion(ctx)
	if err == nil {
		t.Error("GetLatestVersion() expected error for invalid JSON, got nil")
	}
}

func TestCheckerGetCurrentVersion(t *testing.T) {
	checker := NewGitHubChecker("http://example.com", "owner", "repo")
	version := checker.GetCurrentVersion()
	if version == "" {
		t.Error("GetCurrentVersion() returned empty string")
	}
}
