package updater

import "context"

type VersionChecker interface {
	GetCurrentVersion() string
	GetLatestVersion(ctx context.Context) (string, error)
}

type Downloader interface {
	Download(ctx context.Context, url, destPath string) error
}

type Updater interface {
	ApplyUpdate(binaryPath string) error
	Rollback(backupPath, currentPath string) error
}
