package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GitUpdateInfo contains update information for git-based updates
type GitUpdateInfo struct {
	Available     bool
	CurrentCommit string
	LatestCommit  string
	Message       string
}

// GitService handles git-based auto-updates
type GitService struct {
	checker *GitChecker
	puller  *GitPuller
	builder *Builder
	hooks   struct {
		beforeUpdate func()
		afterUpdate  func()
		onUpdate     func(newCommit, message string)
	}
	stopChan chan struct{}
	mu       sync.RWMutex
}

// NewGitServiceAuto creates a new git-based update service by detecting remote origin automatically
func NewGitServiceAuto(outputPath string) (*GitService, error) {
	// Get the repo path from the current executable or use default
	repoPath, err := getRepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine repo path: %w", err)
	}

	// Create checker from remote origin
	checker, err := NewGitCheckerFromRepo(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create git checker: %w", err)
	}

	mainPath := filepath.Join(repoPath, "cmd/singularity")

	return &GitService{
		checker:  checker,
		puller:   NewGitPuller(repoPath),
		builder:  NewBuilder(repoPath, mainPath, outputPath),
		stopChan: make(chan struct{}),
	}, nil
}

// NewGitServiceAutoFromPath creates a git service from a specific repo path
func NewGitServiceAutoFromPath(repoPath, outputPath string) (*GitService, error) {
	checker, err := NewGitCheckerFromRepo(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create git checker: %w", err)
	}

	mainPath := filepath.Join(repoPath, "cmd/singularity")

	return &GitService{
		checker:  checker,
		puller:   NewGitPuller(repoPath),
		builder:  NewBuilder(repoPath, mainPath, outputPath),
		stopChan: make(chan struct{}),
	}, nil
}

// getRepoPath determines the repository path
func getRepoPath() (string, error) {
	// Try to get from git rev-parse
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}

	// Fallback: use current working directory
	return os.Getwd()
}

// NewGitService creates a new git-based update service
func NewGitService(owner, repo, branch, repoPath, outputPath string) *GitService {
	mainPath := "cmd/singularity"
	if repoPath != "" {
		mainPath = filepath.Join(repoPath, "cmd/singularity")
	}

	return &GitService{
		checker:  NewGitChecker(owner, repo, branch, repoPath),
		puller:   NewGitPuller(repoPath),
		builder:  NewBuilder(repoPath, mainPath, outputPath),
		stopChan: make(chan struct{}),
	}
}

// NewGitServiceWithHooks creates a git service with callbacks
func NewGitServiceWithHooks(
	owner, repo, branch, repoPath, outputPath string,
	beforeUpdate, afterUpdate func(),
	onUpdate func(newCommit, message string),
) *GitService {
	svc := NewGitService(owner, repo, branch, repoPath, outputPath)
	svc.hooks.beforeUpdate = beforeUpdate
	svc.hooks.afterUpdate = afterUpdate
	svc.hooks.onUpdate = onUpdate
	return svc
}

// CheckForUpdates checks if there's a newer RELEASE commit
func (s *GitService) CheckForUpdates(ctx context.Context) (*GitUpdateInfo, error) {
	currentCommit := s.checker.GetCurrentCommit()
	available, latestCommit, message, err := s.checker.IsUpdateAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	return &GitUpdateInfo{
		Available:     available,
		CurrentCommit: currentCommit,
		LatestCommit:  latestCommit,
		Message:       message,
	}, nil
}

// Update fetches latest code and builds
func (s *GitService) Update(ctx context.Context) (string, error) {
	// Get latest commit info
	latestCommit, message, err := s.checker.GetLatestReleaseCommit(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get latest release: %w", err)
	}

	// Run before update hook
	if s.hooks.beforeUpdate != nil {
		s.hooks.beforeUpdate()
	}

	// Fetch and checkout to the release commit
	if err := s.puller.FetchAndCheckout(ctx, latestCommit); err != nil {
		return "", fmt.Errorf("failed to fetch latest code: %w", err)
	}

	// Build the application
	// Extract version from commit message (e.g., "RELEASE v1.2.3")
	version := extractVersionFromMessage(message)
	if version == "" {
		version = latestCommit[:7] // Use short commit hash
	}

	if err := s.builder.BuildWithVersion(ctx, version, latestCommit); err != nil {
		return "", fmt.Errorf("failed to build: %w", err)
	}

	// Run on update hook
	if s.hooks.onUpdate != nil {
		s.hooks.onUpdate(latestCommit, message)
	}

	// Run after update hook
	if s.hooks.afterUpdate != nil {
		s.hooks.afterUpdate()
	}

	return latestCommit, nil
}

// StartAutoUpdate starts periodic update checks
func (s *GitService) StartAutoUpdate(interval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.stopChan:
		s.stopChan = make(chan struct{})
	default:
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)

				// Check for updates
				updateInfo, err := s.CheckForUpdates(ctx)
				cancel()

				if err != nil {
					continue
				}

				// If update available, fetch, build, and apply
				if updateInfo.Available {
					if s.hooks.beforeUpdate != nil {
						s.hooks.beforeUpdate()
					}

					buildCtx, buildCancel := context.WithTimeout(context.Background(), 300*time.Second)
					_, err := s.Update(buildCtx)
					buildCancel()

					if err != nil {
						continue
					}
				}

			case <-s.stopChan:
				return
			}
		}
	}()

	return nil
}

// StopAutoUpdate stops the periodic update checks
func (s *GitService) StopAutoUpdate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.stopChan:
		return
	default:
		close(s.stopChan)
	}
}

// GetCurrentCommit returns the current git commit
func (s *GitService) GetCurrentCommit() string {
	return s.checker.GetCurrentCommit()
}

// GetBuilder returns the builder for custom builds
func (s *GitService) GetBuilder() *Builder {
	return s.builder
}

// GetPuller returns the git puller for custom operations
func (s *GitService) GetPuller() *GitPuller {
	return s.puller
}

// extractVersionFromMessage extracts version from commit message
func extractVersionFromMessage(message string) string {
	// Try to find version pattern like v1.2.3
	message = strings.TrimSpace(message)
	parts := strings.Fields(message)
	for _, part := range parts {
		if strings.HasPrefix(part, "v") && len(part) > 1 {
			// Check if it looks like a version (contains at least one dot)
			if strings.Contains(part, ".") {
				return part
			}
		}
	}
	return ""
}

// BackupAndUpdate creates a backup before updating
func (s *GitService) BackupAndUpdate(ctx context.Context, backupDir string) (string, error) {
	currentBinary := s.builder.GetOutputPath()

	// Create backup if current binary exists
	if _, err := os.Stat(currentBinary); err == nil {
		backupPath := filepath.Join(backupDir, "singularity-backup")
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create backup directory: %w", err)
		}

		// Copy current binary to backup
		data, err := os.ReadFile(currentBinary)
		if err != nil {
			return "", fmt.Errorf("failed to read current binary: %w", err)
		}

		if err := os.WriteFile(backupPath, data, 0755); err != nil {
			return "", fmt.Errorf("failed to write backup: %w", err)
		}
	}

	// Perform update
	return s.Update(ctx)
}
