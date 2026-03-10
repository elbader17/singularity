package updater

import (
	"os"
	"path/filepath"
	"testing"
)

type MockUpdater struct {
	applyErr error
}

func (m *MockUpdater) ApplyUpdate(binaryPath string) error {
	return m.applyErr
}

func TestUpdaterApplyUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "new-binary")

	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("failed to create test binary: %v", err)
	}

	updater := NewSystemUpdater()

	err := updater.ApplyUpdate(binaryPath)
	if err != nil {
		t.Fatalf("ApplyUpdate() error = %v", err)
	}
}

func TestUpdaterApplyUpdateFileNotFound(t *testing.T) {
	updater := NewSystemUpdater()

	err := updater.ApplyUpdate("/nonexistent/binary")
	if err == nil {
		t.Error("ApplyUpdate() expected error for missing file, got nil")
	}
}

func TestUpdaterApplyUpdateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "current")
	newBinary := filepath.Join(tmpDir, "new")
	backupPath := filepath.Join(tmpDir, "backup")

	if err := os.WriteFile(currentBinary, []byte("old content"), 0755); err != nil {
		t.Fatalf("failed to create test binary: %v", err)
	}
	if err := os.WriteFile(newBinary, []byte("new content"), 0755); err != nil {
		t.Fatalf("failed to create new binary: %v", err)
	}

	updater := NewSystemUpdater()

	err := updater.ApplyUpdateWithBackup(newBinary, backupPath)
	if err != nil {
		t.Fatalf("ApplyUpdateWithBackup() error = %v", err)
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(backupData) != "old content" {
		t.Errorf("backup content = %q, want %q", string(backupData), "old content")
	}
}

func TestUpdaterApplyUpdatePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "binary")

	if err := os.WriteFile(binaryPath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test binary: %v", err)
	}

	updater := NewSystemUpdater()

	err := updater.ApplyUpdate(binaryPath)
	if err == nil {
		t.Error("ApplyUpdate() expected error for non-executable file, got nil")
	}
}

func TestUpdaterRollback(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "current")
	backupPath := filepath.Join(tmpDir, "backup")

	if err := os.WriteFile(currentBinary, []byte("current"), 0755); err != nil {
		t.Fatalf("failed to create test binary: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("backup"), 0755); err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	updater := NewSystemUpdater()

	err := updater.Rollback(backupPath, currentBinary)
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	currentData, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("failed to read current: %v", err)
	}
	if string(currentData) != "backup" {
		t.Errorf("after rollback, current = %q, want %q", string(currentData), "backup")
	}
}
