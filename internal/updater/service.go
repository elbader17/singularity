package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"singularity/internal/version"
	"strings"
	"sync"
	"time"
)

type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
}

type Service struct {
	checker     VersionChecker
	downloader  Downloader
	updater     Updater
	downloadURL string
	tempDir     string
	hooks       struct {
		beforeUpdate func()
		afterUpdate  func()
	}
	stopChan chan struct{}
	mu       sync.RWMutex
}

func NewService(checker VersionChecker, downloader Downloader, updater Updater, downloadURL, tempDir string) *Service {
	return &Service{
		checker:     checker,
		downloader:  downloader,
		updater:     updater,
		downloadURL: downloadURL,
		tempDir:     tempDir,
		stopChan:    make(chan struct{}),
	}
}

func NewServiceWithHooks(checker VersionChecker, downloader Downloader, updater Updater, downloadURL, tempDir string, beforeUpdate func(), afterUpdate func()) *Service {
	svc := NewService(checker, downloader, updater, downloadURL, tempDir)
	svc.hooks.beforeUpdate = beforeUpdate
	svc.hooks.afterUpdate = afterUpdate
	return svc
}

func (s *Service) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	currentVersion := s.checker.GetCurrentVersion()
	latestVersion, err := s.checker.GetLatestVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	available := version.IsUpdateAvailable(currentVersion, latestVersion)

	return &UpdateInfo{
		Available:      available,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
	}, nil
}

func (s *Service) DownloadUpdate(ctx context.Context, version string) (string, error) {
	url := s.buildDownloadURL(version)

	destPath := filepath.Join(s.tempDir, fmt.Sprintf("singularity-%s", version))
	if strings.HasSuffix(version, ".exe") || strings.Contains(version, "windows") {
		destPath += ".exe"
	}

	if err := s.downloader.Download(ctx, url, destPath); err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return "", fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return destPath, nil
}

func (s *Service) ApplyUpdate(binaryPath string) error {
	if s.hooks.beforeUpdate != nil {
		s.hooks.beforeUpdate()
	}

	if err := s.updater.ApplyUpdate(binaryPath); err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}

	if s.hooks.afterUpdate != nil {
		s.hooks.afterUpdate()
	}

	return nil
}

func (s *Service) StartAutoUpdate(interval time.Duration) error {
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
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

				// Check for updates
				updateInfo, err := s.CheckForUpdates(ctx)
				if err != nil {
					cancel()
					continue
				}

				// If update available, download and apply
				if updateInfo.Available {
					if s.hooks.beforeUpdate != nil {
						s.hooks.beforeUpdate()
					}

					downloadCtx, downloadCancel := context.WithTimeout(context.Background(), 120*time.Second)
					binaryPath, err := s.DownloadUpdate(downloadCtx, updateInfo.LatestVersion)
					downloadCancel()

					if err != nil {
						cancel()
						continue
					}

					if err := s.ApplyUpdate(binaryPath); err != nil {
						cancel()
						continue
					}
				}

				cancel()
			case <-s.stopChan:
				return
			}
		}
	}()

	return nil
}

func (s *Service) StopAutoUpdate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.stopChan:
		return
	default:
		close(s.stopChan)
	}
}

func (s *Service) buildDownloadURL(version string) string {
	version = strings.TrimPrefix(version, "v")
	return fmt.Sprintf("%s/singularity-%s", s.downloadURL, version)
}
