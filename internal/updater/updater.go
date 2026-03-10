package updater

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type SystemUpdater struct{}

func NewSystemUpdater() *SystemUpdater {
	return &SystemUpdater{}
}

func (u *SystemUpdater) ApplyUpdate(binaryPath string) error {
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("binary is not executable")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable: %w", err)
	}

	currentInfo, err := os.Stat(currentExe)
	if err != nil {
		return fmt.Errorf("failed to stat current executable: %w", err)
	}

	if err := os.Chmod(binaryPath, currentInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(binaryPath, currentExe); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	return nil
}

func (u *SystemUpdater) ApplyUpdateWithBackup(newPath, backupPath string) error {
	currentExe, err := os.Executable()
	if err != nil {
		currentExe = ""
	}

	currentExeInfo, err := os.Stat(currentExe)
	if err != nil {
		currentExe = ""
	}

	currentDir := filepath.Dir(newPath)
	currentInDir := filepath.Join(currentDir, "current")
	if info, err := os.Stat(currentInDir); err == nil && info.Mode().IsRegular() {
		currentExe = currentInDir
		currentExeInfo, _ = os.Stat(currentExe)
	}

	if currentExe == "" {
		return fmt.Errorf("cannot determine current binary")
	}

	if err := copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	if currentExeInfo != nil {
		if err := os.Chmod(newPath, currentExeInfo.Mode()); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	if err := os.Rename(newPath, currentExe); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	return nil
}

func (u *SystemUpdater) Rollback(backupPath, currentPath string) error {
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup: %w", err)
	}

	if err := os.Rename(backupPath, currentPath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	if err := os.Chmod(currentPath, backupInfo.Mode()); err != nil {
		return fmt.Errorf("failed to restore permissions: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}
