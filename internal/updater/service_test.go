package updater

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockChecker struct {
	current string
	latest  string
	err     error
}

func (m *mockChecker) GetCurrentVersion() string {
	return m.current
}

func (m *mockChecker) GetLatestVersion(ctx context.Context) (string, error) {
	return m.latest, m.err
}

type mockDownloader struct {
	err error
}

func (m *mockDownloader) Download(ctx context.Context, version, destPath string) error {
	return m.err
}

type mockUpdater struct {
	err error
}

func (m *mockUpdater) ApplyUpdate(binaryPath string) error {
	return m.err
}

func (m *mockUpdater) Rollback(backupPath, currentPath string) error {
	return m.err
}

func TestServiceCheckForUpdates(t *testing.T) {
	checker := &mockChecker{current: "1.0.0", latest: "1.1.0"}
	downloader := &mockDownloader{}
	updater := &mockUpdater{}

	svc := NewService(checker, downloader, updater, "http://example.com", "/tmp")
	ctx := context.Background()

	update, err := svc.CheckForUpdates(ctx)
	if err != nil {
		t.Fatalf("CheckForUpdates() error = %v", err)
	}
	if !update.Available {
		t.Error("CheckForUpdates() expected update to be available")
	}
	if update.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", update.CurrentVersion, "1.0.0")
	}
	if update.LatestVersion != "1.1.0" {
		t.Errorf("LatestVersion = %q, want %q", update.LatestVersion, "1.1.0")
	}
}

func TestServiceNoUpdateAvailable(t *testing.T) {
	checker := &mockChecker{current: "1.1.0", latest: "1.0.0"}
	downloader := &mockDownloader{}
	updater := &mockUpdater{}

	svc := NewService(checker, downloader, updater, "http://example.com", "/tmp")
	ctx := context.Background()

	update, err := svc.CheckForUpdates(ctx)
	if err != nil {
		t.Fatalf("CheckForUpdates() error = %v", err)
	}
	if update.Available {
		t.Error("CheckForUpdates() expected no update to be available")
	}
}

func TestServiceCheckForUpdatesError(t *testing.T) {
	checker := &mockChecker{err: errors.New("network error")}
	downloader := &mockDownloader{}
	updater := &mockUpdater{}

	svc := NewService(checker, downloader, updater, "http://example.com", "/tmp")
	ctx := context.Background()

	_, err := svc.CheckForUpdates(ctx)
	if err == nil {
		t.Error("CheckForUpdates() expected error, got nil")
	}
}

func TestServiceDownloadUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("binary content"))
	}))
	defer server.Close()

	checker := &mockChecker{current: "1.0.0", latest: "1.1.0"}
	downloader := NewHTTPDownloader()
	updater := &mockUpdater{}

	svc := NewService(checker, downloader, updater, server.URL, t.TempDir())
	ctx := context.Background()

	path, err := svc.DownloadUpdate(ctx, "1.1.0")
	if err != nil {
		t.Fatalf("DownloadUpdate() error = %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("DownloadUpdate() did not create downloaded file")
	}
}

func TestServiceDownloadUpdateError(t *testing.T) {
	checker := &mockChecker{}
	downloader := &mockDownloader{err: errors.New("download failed")}
	updater := &mockUpdater{}

	svc := NewService(checker, downloader, updater, "http://invalid", "/tmp")
	ctx := context.Background()

	_, err := svc.DownloadUpdate(ctx, "1.1.0")
	if err == nil {
		t.Error("DownloadUpdate() expected error, got nil")
	}
}

func TestServiceApplyUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "update")

	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash"), 0755); err != nil {
		t.Fatalf("failed to create test binary: %v", err)
	}

	checker := &mockChecker{}
	downloader := &mockDownloader{}
	updater := &mockUpdater{}

	svc := NewService(checker, downloader, updater, "http://example.com", tmpDir)

	err := svc.ApplyUpdate(binaryPath)
	if err != nil {
		t.Fatalf("ApplyUpdate() error = %v", err)
	}
}

func TestServiceApplyUpdateError(t *testing.T) {
	checker := &mockChecker{}
	downloader := &mockDownloader{}
	updater := &mockUpdater{err: errors.New("apply failed")}

	svc := NewService(checker, downloader, updater, "http://example.com", "/tmp")

	err := svc.ApplyUpdate("/nonexistent")
	if err == nil {
		t.Error("ApplyUpdate() expected error, got nil")
	}
}

func TestServiceFullUpdateFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("new binary"))
	}))
	defer server.Close()

	checker := &mockChecker{current: "1.0.0", latest: "1.1.0"}
	downloader := NewHTTPDownloader()
	updater := NewSystemUpdater()

	tmpDir := t.TempDir()
	svc := NewService(checker, downloader, updater, server.URL, tmpDir)
	ctx := context.Background()

	update, err := svc.CheckForUpdates(ctx)
	if err != nil {
		t.Fatalf("CheckForUpdates() error = %v", err)
	}
	if !update.Available {
		t.Fatal("expected update to be available")
	}

	path, err := svc.DownloadUpdate(ctx, update.LatestVersion)
	if err != nil {
		t.Fatalf("DownloadUpdate() error = %v", err)
	}

	err = svc.ApplyUpdate(path)
	if err != nil {
		t.Fatalf("ApplyUpdate() error = %v", err)
	}
}

func TestServiceAutoUpdateWithSchedule(t *testing.T) {
	checker := &mockChecker{current: "1.0.0", latest: "1.1.0"}
	downloader := &mockDownloader{}
	updater := &mockUpdater{}

	svc := NewService(checker, downloader, updater, "http://example.com", "/tmp")

	err := svc.StartAutoUpdate(24 * time.Hour)
	if err != nil {
		t.Fatalf("StartAutoUpdate() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	svc.StopAutoUpdate()
}

func TestServiceAutoUpdateCheck(t *testing.T) {
	checkCallCount := 0
	checker := &mockChecker{current: "1.0.0", latest: "1.1.0"}
	downloader := &mockDownloader{}
	updater := &mockUpdater{}

	svc := NewServiceWithHooks(
		checker, downloader, updater,
		"http://example.com", "/tmp",
		func() { checkCallCount++ },
		nil,
	)

	svc.StartAutoUpdate(50 * time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	svc.StopAutoUpdate()

	if checkCallCount < 2 {
		t.Errorf("expected at least 2 checks, got %d", checkCallCount)
	}
}
